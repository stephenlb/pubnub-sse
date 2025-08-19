#!/usr/bin/env node

const PubNub = require('./pubnub.js');

const CHANNEL = process.argv[2] || 'mychannel';
const SUBSCRIBE_KEY = process.env.PUBNUB_SUBSCRIBE_KEY || 'demo';
const USER_ID = process.env.PUBNUB_USER_ID || 'message-counter';

let totalMessages = 0;
let messagesThisSecond = 0;

console.log(`Starting message counter for channel: ${CHANNEL}`);
console.log(`Using subscribe key: ${SUBSCRIBE_KEY}`);
console.log(`User ID: ${USER_ID}`);
console.log('---');

const pubnub = PubNub({
    subscribeKey: SUBSCRIBE_KEY,
    userId: USER_ID
});

const subscription = pubnub.subscribe({
    channel: CHANNEL,
    messages: (message) => {
        totalMessages++;
        messagesThisSecond++;
    }
});

setInterval(() => {
    console.log(`Messages this second: ${messagesThisSecond}, Total messages: ${totalMessages}`);
    messagesThisSecond = 0;
}, 1000);

process.on('SIGINT', () => {
    console.log('\nShutting down...');
    subscription.unsubscribe();
    process.exit(0);
});

console.log('Waiting for messages... (Press Ctrl+C to stop)');
