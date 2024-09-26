// test/pubnub.test.js
const { expect } = require('chai');
const PubNub = require('../pubnub.js');
const PubNubCryptor = require('pubnub');

const pubkey = 'demo';
const subkey = 'demo';
const authKey = 'demo-auth-key';
const userId = 'test-user-id';
const cipherKey = 'pubnubenigma';

describe('PubNub SDK Tests', () => {
    const pubnubInstance = PubNub({
        publishKey: pubkey,
        subscribeKey: subkey,
        authKey: authKey,
        userId: userId,
    });
    const pubnubCryptor = new PubNubCryptor({
        subscribeKey: subkey,
        publishKey: pubkey,
        uuid: userId,
        authKey: authKey,
        cipherKey: cipherKey,
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

    describe('encryption', async () => {
        it('should encrypt and decrypt messages', async () => {
            const message = { text: "Hello World" };
            const stringData = JSON.stringify(message);
            const encrypted = pubnubCryptor.encrypt(stringData, cipherKey);
            expect(message).to.be.an('object');
            expect(encrypted).to.be.a('string');

            const decrypted = pubnubCryptor.decrypt(encrypted, cipherKey);
            expect(decrypted).to.be.an('object');
            expect(message).to.deep.equal(decrypted);
        });

        it('should send and recieve encrypted messages', async () => {
            const message = { text: "Hello World" };
            const stringData = JSON.stringify(message);
            const encrypted = pubnubCryptor.encrypt(stringData, cipherKey);
            const channel = `test-channel-${Math.random()}`;
            const subscription = pubnubInstance.subscribe({channel: channel});

            // Publish
            setTimeout(async () => {
                await pubnubInstance.publish({ channel: channel, message: encrypted});
            }, 1000);

            // Subscription Stream
            for await (const encryptedMessage of subscription) {
                const decrypted = pubnubCryptor.decrypt(encryptedMessage, cipherKey);
                expect(encryptedMessage).to.equal(encrypted);
                expect(encryptedMessage).to.be.a('string');
                expect(decrypted).to.be.an('object');
                expect(message).to.deep.equal(decrypted);
                break;
            }

            subscription.unsubscribe();
        });
    });

});
