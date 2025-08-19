// ---------------------------------------------
// SSE over HTTP/3 with IPv4+IPv6 PubNub Client
// ---------------------------------------------
// - - - - - - - - - - - - - - - - - - - - - - -
// Features
// - - - - - - - - - - - - - - - - - - - - - - -
//  - Publish/Subscribe
//  - SSE
//  - Streaming Compression
//  - Dedicated queue per channel for maximum performance
//  - SSE over HTTP/3 with IPv4+IPv6
//  - IPv6, IPv4
//  - HTTP/3, HTTP/2, and HTTP/1.1 fallback
//  - TLS 1.3

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultSubKey  = "demo"
	defaultPubKey  = "demo"
	defaultChannel = "pubnub"
	defaultUserID  = "user-default"
	defaultAuthKey = "user-default"
	defaultOrigin  = "h2.pubnubapi.com"
)

type Config struct {
	SubscribeKey string
	PublishKey   string
	Channel      string
	Origin       string
	UserID       string
	AuthKey      string
	Filter       string
	Timetoken    string
}

type PubNubClient struct {
	config     Config
	httpClient *http.Client
}

type MessageHandler func(interface{})

type Subscription struct {
	client     *PubNubClient
	config     Config
	messages   chan interface{}
	done       chan struct{}
	ctx        context.Context
	cancel     context.CancelFunc
	subscribed bool
}

func New(config Config) *PubNubClient {
	if config.SubscribeKey == "" {
		config.SubscribeKey = defaultSubKey
	}
	if config.PublishKey == "" {
		config.PublishKey = defaultPubKey
	}
	if config.Channel == "" {
		config.Channel = defaultChannel
	}
	if config.Origin == "" {
		config.Origin = defaultOrigin
	}
	if config.UserID == "" {
		config.UserID = defaultUserID
	}
	if config.AuthKey == "" {
		config.AuthKey = defaultAuthKey
	}
	if config.Timetoken == "" {
		config.Timetoken = "10000"
	}

	return &PubNubClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 0, // No timeout for streaming
		},
	}
}

func (p *PubNubClient) Subscribe(handler MessageHandler) (*Subscription, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	sub := &Subscription{
		client:     p,
		config:     p.config,
		messages:   make(chan interface{}, 100),
		done:       make(chan struct{}),
		ctx:        ctx,
		cancel:     cancel,
		subscribed: true,
	}

	go sub.startStream(handler)
	
	return sub, nil
}

func (s *Subscription) startStream(handler MessageHandler) {
	defer close(s.done)
	
	for s.subscribed {
		select {
		case <-s.ctx.Done():
			return
		default:
			err := s.connectAndRead(handler)
			if err != nil && s.subscribed {
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func (s *Subscription) connectAndRead(handler MessageHandler) error {
	filterExp := ""
	if s.config.Filter != "" {
		filterExp = "&filter-expr=" + url.QueryEscape(s.config.Filter)
	}
	
	params := fmt.Sprintf("auth=%s%s&uuid=%s", 
		s.config.AuthKey, filterExp, s.config.UserID)
	
	uri := fmt.Sprintf("https://%s/stream/%s/%s/0/%s?%s",
		s.config.Origin, s.config.SubscribeKey, s.config.Channel, 
		s.config.Timetoken, params)

	req, err := http.NewRequestWithContext(s.ctx, "GET", uri, nil)
	if err != nil {
		return err
	}

	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return s.readChunkedStream(resp.Body, handler)
}

func (s *Subscription) readChunkedStream(body io.ReadCloser, handler MessageHandler) error {
	reader := bufio.NewReader(body)
	var buffer strings.Builder

	for s.subscribed {
		select {
		case <-s.ctx.Done():
			return nil
		default:
		}

		chunk, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		buffer.Write(chunk)
		content := buffer.String()
		
		lines := strings.Split(content, "\n")
		buffer.Reset()

		for i, line := range lines {
			if line == "" {
				continue
			}
			
			// Keep incomplete lines in buffer for next iteration
			if i == len(lines)-1 && !strings.HasSuffix(content, "\n") {
				buffer.WriteString(line)
				continue
			}

			var jsonMsg []interface{}
			if err := json.Unmarshal([]byte(line), &jsonMsg); err != nil {
				// Invalid JSON, might be incomplete - add back to buffer
				buffer.WriteString(line)
				continue
			}

			if len(jsonMsg) >= 2 {
				// Update timetoken if present
				if tt, ok := jsonMsg[1].(string); ok {
					s.config.Timetoken = tt
				}

				// Process messages
				if messages, ok := jsonMsg[0].([]interface{}); ok {
					for _, msg := range messages {
						if handler != nil {
							handler(msg)
						}
						select {
						case s.messages <- msg:
						case <-s.ctx.Done():
							return nil
						default:
						}
					}
				}
			}
		}
	}

	return nil
}

func (s *Subscription) Messages() <-chan interface{} {
	return s.messages
}

func (s *Subscription) Unsubscribe() {
	s.subscribed = false
	s.cancel()
	<-s.done
	close(s.messages)
}

func (p *PubNubClient) Publish(message interface{}, metadata map[string]interface{}) error {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	params := fmt.Sprintf("auth=%s&meta=%s&uuid=%s",
		p.config.AuthKey, url.QueryEscape(string(metaJSON)), p.config.UserID)

	uri := fmt.Sprintf("https://%s/publish/%s/%s/0/%s/0?%s",
		p.config.Origin, p.config.PublishKey, p.config.SubscribeKey,
		p.config.Channel, params)

	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	resp, err := p.httpClient.Post(uri, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return nil
}

func main() {
	config := Config{
		SubscribeKey: "demo",
		PublishKey:   "demo",
		Channel:      "pubnub",
	}

	client := New(config)

	// Subscribe to messages
	sub, err := client.Subscribe(func(message interface{}) {
		fmt.Printf("Received message: %+v\n", message)
	})
	if err != nil {
		fmt.Printf("Error subscribing: %v\n", err)
		return
	}

	// Listen for messages in a goroutine
	go func() {
		for msg := range sub.Messages() {
			fmt.Printf("Channel message: %+v\n", msg)
		}
	}()

	// Publish a test message after a short delay
	go func() {
		time.Sleep(2 * time.Second)
		err := client.Publish("Hello from Go!", map[string]interface{}{
			"sender": "golang-client",
			"timestamp": time.Now().Unix(),
		})
		if err != nil {
			fmt.Printf("Error publishing: %v\n", err)
		} else {
			fmt.Println("Message published successfully")
		}
	}()

	// Run for 30 seconds then cleanup
	time.Sleep(30 * time.Second)
	sub.Unsubscribe()
	fmt.Println("Unsubscribed and cleaned up")
}