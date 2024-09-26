# PubNub SSE

Easy to use PubNub SDK with SSE enabled by default.


### NPM Install

```shell
npm install pubnub-sse
```

### Run a Quick Demo

```shell
git clone https://github.com/stephenlb/pubnub-sse.git
cd pubnub-sse
open index.html
```

#### Important Files:

 - `pubnub.js` PubNub SSE Streaming SDK
 - `index.html` Example app open to see a demo using streaming data.

![PubNub SSE Screenshot](media/screenshot.png)

Setup the SDK as follows in the example.

### Subscription Async Iterator

```javascript
const pubnub = PubNub({ subscribeKey: 'demo', publishKey: 'demo'});
const subscription = pubnub.subscribe({channel: 'test'});

let count = 0;
for await (const msg of subscription) {
    console.log(msg);
    if (count++ >= 2) break;
}
```

### Subscription Callback
```javascript
const pubnub = PubNub({ subscribeKey: 'demo', publishKey: 'demo'});
const subscription = pubnub.subscribe({channel: 'test', messages: reciever});

function reciever(msg) {
    console.log(msg);
}
```


### Encryption Example

```javascript
const PubNub = require('../pubnub.js');  // npm install pubnub-sse
const PubNubCryptor = require('pubnub'); // npm install pubnub

const pubkey = 'demo';
const subkey = 'demo';
const authKey = 'demo-auth-key';
const userId = 'test-user-id';
const cipherKey = 'pubnubenigma';

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
```

### Example Code

```html
<script src="pubnub.js"></script>
<script>

// PubNub Setup
const userId = 'user-id';
const authKey = 'auth-key';
const channel = 'Commands';
const pubkey = 'pub-c-a88f5e0f-af28-4847-ad52-30495d0cbcb8';
const subkey = 'sub-c-a8cbfccb-676b-4034-9681-dfed95af8d7e';
const pubnub = PubNub({
    subscribeKey: subkey,
    publishKey: pubkey,
    authKey: authKey,
    userId: userId,
});

// Subscribe to Events "Starts the Stream"
const subscription = pubnub.subscribe({
    channel: channel,
    messages: receiveEvents,
});

// End subscription
// subscription.unsubscribe();

// Event Processing
function receiveEvents(event) {
    console.log(event);
}

// Publish Events Example
let eventId = 0;
setInterval(() => {
    pubnub.publish({
        channel: channel,
        message: {
            eventId: ++eventId, 
            userId: userId,
            data: `Event ${eventId} from ${userId} at ${new Date().toISOString()}`,
        },
    });
}, 1000);

</script>
```
