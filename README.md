# PubNub SSE

Easy to use PubNub SDK with SSE enabled by default.


### Quick Run Demo

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
