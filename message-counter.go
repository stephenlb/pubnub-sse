package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	defaultSubKey  = "demo"
	defaultChannel = "mychannel"
	defaultUserID  = "message-counter"
	defaultOrigin  = "h2.pubnubapi.com"
)

type MessageCounterClient struct {
	subscribeKey string
	userID       string
	origin       string
	httpClient   *http.Client
}

type CounterSubscription struct {
	client         *MessageCounterClient
	channel        string
	totalMessages  int64
	messagesThisSecond int64
	ctx            context.Context
	cancel         context.CancelFunc
	subscribed     bool
}

func NewMessageCounter(subscribeKey, userID string) *MessageCounterClient {
	if subscribeKey == "" {
		subscribeKey = defaultSubKey
	}
	if userID == "" {
		userID = defaultUserID
	}

	return &MessageCounterClient{
		subscribeKey: subscribeKey,
		userID:       userID,
		origin:       defaultOrigin,
		httpClient: &http.Client{
			Timeout: 0, // No timeout for streaming
		},
	}
}

func (m *MessageCounterClient) Subscribe(channel string) (*CounterSubscription, error) {
	if channel == "" {
		channel = defaultChannel
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	sub := &CounterSubscription{
		client:     m,
		channel:    channel,
		ctx:        ctx,
		cancel:     cancel,
		subscribed: true,
	}

	go sub.startStream()
	
	return sub, nil
}

func (s *CounterSubscription) startStream() {
	timetoken := "10000"
	
	for s.subscribed {
		select {
		case <-s.ctx.Done():
			return
		default:
			newTimetoken, err := s.connectAndRead(timetoken)
			if err != nil && s.subscribed {
				time.Sleep(1 * time.Second)
			} else if newTimetoken != "" {
				timetoken = newTimetoken
			}
		}
	}
}

func (s *CounterSubscription) connectAndRead(timetoken string) (string, error) {
	params := fmt.Sprintf("uuid=%s", s.client.userID)
	
	uri := fmt.Sprintf("https://%s/stream/%s/%s/0/%s?%s",
		s.client.origin, s.client.subscribeKey, s.channel, timetoken, params)

	req, err := http.NewRequestWithContext(s.ctx, "GET", uri, nil)
	if err != nil {
		return "", err
	}

	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return s.readChunkedStream(resp.Body, timetoken)
}

func (s *CounterSubscription) readChunkedStream(body io.ReadCloser, currentTimetoken string) (string, error) {
	reader := bufio.NewReader(body)
	var buffer strings.Builder
	newTimetoken := currentTimetoken

	for s.subscribed {
		select {
		case <-s.ctx.Done():
			return newTimetoken, nil
		default:
		}

		chunk, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return newTimetoken, nil
			}
			return newTimetoken, err
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
					newTimetoken = tt
				}

				// Count messages
				if messages, ok := jsonMsg[0].([]interface{}); ok {
					messageCount := int64(len(messages))
					atomic.AddInt64(&s.totalMessages, messageCount)
					atomic.AddInt64(&s.messagesThisSecond, messageCount)
				}
			}
		}
	}

	return newTimetoken, nil
}

func (s *CounterSubscription) GetTotalMessages() int64 {
	return atomic.LoadInt64(&s.totalMessages)
}

func (s *CounterSubscription) GetAndResetMessagesThisSecond() int64 {
	return atomic.SwapInt64(&s.messagesThisSecond, 0)
}

func (s *CounterSubscription) Unsubscribe() {
	s.subscribed = false
	s.cancel()
}

func main() {
	// Parse command line arguments and environment variables
	channel := defaultChannel
	if len(os.Args) > 1 {
		channel = os.Args[1]
	}

	subscribeKey := os.Getenv("PUBNUB_SUBSCRIBE_KEY")
	if subscribeKey == "" {
		subscribeKey = defaultSubKey
	}

	userID := os.Getenv("PUBNUB_USER_ID")
	if userID == "" {
		userID = defaultUserID
	}

	fmt.Printf("Starting message counter for channel: %s\n", channel)
	fmt.Printf("Using subscribe key: %s\n", subscribeKey)
	fmt.Printf("User ID: %s\n", userID)
	fmt.Println("---")

	// Create client and subscribe
	client := NewMessageCounter(subscribeKey, userID)
	subscription, err := client.Subscribe(channel)
	if err != nil {
		fmt.Printf("Error subscribing: %v\n", err)
		os.Exit(1)
	}

	// Start counter display ticker
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Waiting for messages... (Press Ctrl+C to stop)")

	// Main loop
	go func() {
		for {
			select {
			case <-ticker.C:
				messagesThisSecond := subscription.GetAndResetMessagesThisSecond()
				totalMessages := subscription.GetTotalMessages()
				fmt.Printf("Messages this second: %d, Total messages: %d\n", 
					messagesThisSecond, totalMessages)
			case <-sigChan:
				fmt.Println("\nShutting down...")
				subscription.Unsubscribe()
				os.Exit(0)
			}
		}
	}()

	// Keep main goroutine alive
	select {}
}