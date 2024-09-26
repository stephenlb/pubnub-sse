// test/pubnub.test.js
const { expect } = require('chai');
const nock = require('nock');
const PubNub = require('../pubnub.js');

const pubkey = 'demo';
const subkey = 'demo';
const authKey = 'demo-auth-key';
const origin = 'ps.pndsn.com';
const userId = 'test-user-id';

describe('PubNub Client', () => {
    const pubnubInstance = PubNub({
        publishKey: pubkey,
        subscribeKey: subkey,
        authKey: authKey,
        userId: userId,
    });

    describe('publish', () => {
        it('should successfully publish a message', async () => {
            const testMessage = { text: 'Hello World' };
            const channel = `test-channel-${Math.random()}`;
            const response = await pubnubInstance.publish({
                channel: channel,
                message: testMessage,
            });
            const data = await response.json();
            expect(data).to.be.an('array');
        });

        it('should fail to publish a message with wrong URL', async () => {
            const testMessage = { text: 'Hello World' };
            const channel = `test-channel-${Math.random()}`;
            const result = await pubnubInstance.publish({
                channel: channel,
                message: testMessage,
                origin: 'wrong-origin',
            });
            expect(result).to.be.false;
        });
    });

    describe('subscribe', async () => {
        it('should handle messages from subscription', async () => {
            const testMessage1 = { text: 'Hello World 1' };
            const testMessage2 = { text: 'Hello World 2' };
            const channel = `test-channel-${Math.random()}`;
            const subscription = pubnubInstance.subscribe({channel: channel});

            // Publish two messages
            setTimeout(async () => {
                await pubnubInstance.publish({ channel: channel, message: testMessage1});
                await pubnubInstance.publish({ channel: channel, message: testMessage2});
            }, 1000);

            const receivedMessages = [];
            for await (const msg of subscription) {
                receivedMessages.push(msg);
                if (receivedMessages.length >= 2) break;
            }

            expect(receivedMessages).to.have.lengthOf(2);
            expect(receivedMessages[0]).to.deep.equal(testMessage1);
            expect(receivedMessages[1]).to.deep.equal(testMessage2);

            subscription.unsubscribe();
        });
    });
});
