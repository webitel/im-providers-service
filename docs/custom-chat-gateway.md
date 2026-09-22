# Custom chat gateway — integration guide

A custom chat gateway connects an arbitrary external communication system to
Webitel as a chat channel. Anything that can make and receive one signed HTTP
request qualifies: a company's own messenger, an AI bot fronting several
channels, a website widget, a banking portal. Nothing is built on the Webitel
side — a gateway is configuration.

Two endpoints carry the whole integration:

| Direction | Endpoint | Who calls |
| --- | --- | --- |
| into Webitel | `POST {public_url}/wh/custom/{uri}` | your system |
| out of Webitel | `POST {callback_url}` | Webitel |

Both are `application/json` over HTTPS, and both are authenticated the same way.

> **Migrating from the previous Custom Chat Gateway?** The envelope, the field
> names and the signature are unchanged, so an existing integration moves over
> by configuration alone. See [Migrating from v1](#migrating-from-v1).

---

## 1. Configure the gateway

Create the gateway over REST:

```bash
curl -X POST "$WEBITEL/v1/gates/custom" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Partner middleware",
    "peer": {"sub": "<flow schema bot sub>", "iss": "<issuer>"},
    "callback_url": "https://partner.example.org/webitel/hook",
    "allowed_ips": ["203.0.113.0/24"],
    "request_timeout_ms": 5000,
    "retry_attempts": 3
  }'
```

| Field | Required | Meaning |
| --- | --- | --- |
| `name` | yes | Arbitrary label shown in the gateway list |
| `peer` | yes | The Webitel bot the channel routes inbound messages to; this is what binds the gateway to a flow schema |
| `callback_url` | yes | Absolute `http(s)` URL on your side that receives operator replies and events |
| `app_secret` | no | Shared signing secret. **Generated for you when omitted** — read it back from the response |
| `allowed_ips` | no | Addresses or CIDR blocks allowed to call the inbound webhook. Empty accepts any source, leaving the signature as the only authentication |
| `request_timeout_ms` | no | Timeout of a single outbound attempt (default 5000) |
| `retry_attempts` | no | Retries before a message is marked failed (default 3) |

The response carries the two values you need:

```json
{
  "item": {
    "id": "0192f0c2-...",
    "webhook_url": "https://webitel.example.com/wh/custom/9f2c...48b",
    "callback_url": "https://partner.example.org/webitel/hook",
    "app_secret": "7c1d...",
    "status": "PROVIDER_STATUS_ACTIVE"
  }
}
```

`GET`, `PATCH` and `DELETE` on `/v1/gates/custom/{id}` read, update and remove
it. **Changing `app_secret` breaks the integration** until your side is updated
too; leave the field out of a `PATCH` to keep the current secret.

### System message templates

The gateway's `Templates` tab configures what the customer sees when an agent
joins, leaves, transfers the chat, or the conversation ends. Templates are Go
`text/template` bodies keyed by event type:

| Event type | Raised when |
| --- | --- |
| `member_added` | an agent or bot joined the conversation |
| `member_removed` | a participant left it |
| `transferred` | the conversation moved to another agent |

```bash
curl -X PUT "$WEBITEL/im/gates/$GATE_ID/templates/member_added" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"template": "{{ .new_member_name }} joined the chat"}'
```

An event with no template sends nothing at all.

---

## 2. Authenticate every request

Both directions sign the **raw request body** with the gateway secret and put
the result in `X-Webitel-Sign`:

```
X-Webitel-Sign = hex( HMAC_SHA256( raw_body_bytes, app_secret ) )
```

```javascript
const signature = crypto.createHmac('sha256', appSecret)
                        .update(rawBody)          // the bytes, not a re-serialised object
                        .digest('hex');
```

Sign the bytes you actually send. Re-serialising the parsed object can reorder
keys or change whitespace, and the signature will not match.

Webitel rejects an inbound request with:

| Situation | Status |
| --- | --- |
| signature missing or wrong | `403` |
| source address not in `allowed_ips` | `403` |
| unknown `{uri}` | `404` |
| unparsable or empty payload | `500` with `{"success": false, "error": "..."}` |

Verify the header on your endpoint too — Webitel signs its outbound calls with
the same secret, and the endpoint is reachable from the internet.

---

## 3. Send messages to Webitel

Every inbound call is one envelope with exactly one of `message`, `status` or
`broadcast`.

### message

```http
POST /wh/custom/9f2c...48b HTTP/1.1
X-Webitel-Sign: f7e109787f6f012883352ce63ec90849a5e76011db2c7da132742c277414935a
Content-Type: application/json
```

```jsonc
{
  "message": {
    // Required. Unique within your system; this is the deduplication key, so a
    // redelivered webhook does not become a second message.
    "id": "ext-1",

    // Required. Your conversation id. A new chatId starts a new conversation.
    "chatId": "conv-42",

    "sender": {
      "id": "u-1",          // required
      "type": "viber",      // source channel, optional
      "name": "John Doe",
      "nickname": "john"
    },

    "date": 1710348821689,  // unix milliseconds
    "text": "Hi! Can you help me?",

    // Extra data about the caller. Only read from the message that opens a
    // conversation, and then available to the flow schema as thread variables.
    "metadata": {"phone": "+380000000000", "order_id": "42"},

    // A file your system hosts. Webitel downloads it and stores its own copy.
    "file": {
      "url": "https://partner.example.org/files/a1b2",
      "mime": "image/png",
      "size": 179491,
      "name": "screenshot.png"
    },

    "location": {"lat": 50.45, "lon": 30.52, "title": "Office", "address": "Khreshchatyk 1"},
    "contact":  {"name": "Jane", "phone": "+380000000000", "email": "jane@example.org"},

    // A pressed menu button. messageId echoes the id of the message that
    // carried the menu.
    "callback": {"code": "rate_5", "messageId": "3f2a...9c", "data": "5"},

    // Your id of the message being quoted.
    "replyTo": "ext-0"
  }
}
```

Send one kind of content per message. When several are present they are
resolved in this order: `callback`, `file`, `location`, `contact`, `text` —
`text` alongside a `file` is used as its caption.

Response: `200 {"success": true}`.

### The caller's identity

A caller is identified by `sender.type` and `sender.id` together, as
`{type}|{id}`. One gateway can therefore front several source channels without
their user ids colliding. `sender.type` also reaches the flow schema as the
`source` variable.

### status

Report what happened to a message Webitel sent you. `messageId` is the `id`
from that message.

```json
{"status": {"chatId": "conv-42", "messageId": "3f2a...9c", "status": "delivered", "at": 1710348821689}}
```

`status` is one of `delivered`, `read`, `failed`. A `failed` status may carry a
`reason`. Reporting these is what makes delivery marks appear for the operator;
a channel that reports nothing leaves messages as sent, which is honest.

### broadcast (response)

The asynchronous outcome of an operator-initiated message — see
[Outgoing chat initiation](#5-outgoing-chat-initiation).

---

## 4. Receive messages from Webitel

Webitel posts the same envelope shape to your `callback_url`, signed the same
way. Reply `200` with `{"success": true}`, or `{"success": false, "error": "…"}`
to refuse the payload.

```jsonc
{
  "message": {
    // The Webitel message id. Echo it back in a callback or a status report.
    "id": "3f2a...9c",
    "chatId": "conv-42",

    "sender": {
      "id": "<member id>",
      "type": "webitel",
      // The operator's chat name — what the customer should see. Webitel keeps
      // the real name in its own history.
      "name": "Support"
    },

    "date": 1710348821689,
    "text": "How can I help?",

    // A file from the operator, as a signed URL with a limited lifetime.
    "file": {"url": "https://webitel.example.com/any/file/...", "mime": "application/pdf", "size": 1024, "name": "invoice.pdf"},

    "location": {"lat": 50.45, "lon": 30.52, "title": "Office"},
    "contact":  {"name": "Jane", "phone": "+380000000000"},

    // Options for the customer to choose from.
    "menu": {
      "text": "Rate the chat",
      "singleUse": true,
      "placement": "inline",
      "rows": [
        [
          {"code": "rate_5", "text": "5", "data": "5"},
          {"code": "site",   "text": "Open", "url": "https://example.org"},
          {"code": "geo",    "text": "Share location", "action": "location"}
        ]
      ]
    },

    "replyTo": "3f2a...9b"
  }
}
```

A button is one of three kinds: `data` is a callback payload, `url` opens a
link, `action` asks the customer's client for something (`location`, `phone`,
`contact`, `email`). A `sections` array replaces `rows` for list-style menus.

When the customer presses a `data` button, post a `callback` back with its
`code` and the `id` of the message that carried the menu.

---

## 5. Outgoing chat initiation

An operator can write to a customer who has never contacted you. There is no
conversation to address, so the message arrives as a `broadcast`:

```json
{
  "broadcast": {
    "eventId": "3f2a...9c",
    "recipients": [{"id": "u-1", "type": "viber"}],
    "text": "Your order is ready for pickup"
  }
}
```

`recipients` always holds exactly one entry for this channel. Deliver the
message and, **only if it failed**, post the outcome back asynchronously with
the same `eventId`:

```json
{
  "broadcast": {
    "eventId": "3f2a...9c",
    "recipients": [{"id": "u-1", "type": "viber", "error": "user blocked the bot"}]
  }
}
```

A successful delivery needs no confirmation. Once the customer replies, the
conversation has a `chatId` and later messages arrive as ordinary `message`
events.

---

## 6. Reliability and idempotency

**Deduplicate on `id`.** Webitel deduplicates inbound messages per gateway on
`message.id` for 24 hours. Do the same with the `id` of the messages Webitel
sends you: a retried delivery repeats the same id.

**Retries.** When your endpoint is unreachable or answers with a non-2xx
status, Webitel retries with exponential backoff and jitter, up to
`retry_attempts`, keeping the order of messages within one conversation. When
the attempts run out the message is marked failed and the operator sees that it
did not arrive.

A `{"success": false, "error": "…"}` reply is treated differently: it is your
system's verdict on the payload, so it is **not** retried and surfaces to the
operator immediately.

**Files.** Inbound `file.url` is fetched once and copied into Webitel storage,
so the link may be short-lived. It has to be reachable from Webitel, and it
must either resolve to a public address or live on the same host as your
`callback_url`. Inbound files are capped at 25 MiB.

---

## 7. A minimal receiver

```javascript
const express = require('express');
const crypto = require('crypto');

const APP_SECRET = process.env.WEBITEL_APP_SECRET;
const app = express();

// The signature covers the raw bytes, so keep them.
app.use(express.raw({ type: 'application/json' }));

app.post('/webitel/hook', (req, res) => {
  const expected = crypto.createHmac('sha256', APP_SECRET).update(req.body).digest('hex');
  const given = String(req.get('X-Webitel-Sign') || '');

  if (given.length !== expected.length ||
      !crypto.timingSafeEqual(Buffer.from(given), Buffer.from(expected))) {
    return res.status(403).json({ success: false, error: 'bad signature' });
  }

  const event = JSON.parse(req.body.toString('utf8'));

  if (event.message)   deliverToUser(event.message);
  if (event.broadcast) startConversation(event.broadcast);

  res.json({ success: true });
});

app.listen(8080);
```

Sending a message in, from a shell:

```bash
BODY='{"message":{"id":"e1","chatId":"c1","sender":{"id":"u1","type":"web","name":"John"},"date":'$(date +%s000)',"text":"hi"}}'

# $NF, not $2: OpenSSL prints "SHA2-256(stdin)= <hash>", while the LibreSSL
# that ships with macOS prints the bare hash. Taking the last field works on both.
SIGN=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "$APP_SECRET" -hex | awk '{print $NF}')

curl -i -X POST "$WEBHOOK_URL" \
  -H "X-Webitel-Sign: $SIGN" \
  -H 'Content-Type: application/json' \
  -d "$BODY"
```

---

## Migrating from v1

The envelope, the field names and the signature are the same, so an existing
integration keeps working against a new `custom` gateway once the URL and
secret are updated. Everything below is an addition; a v1 client neither sends
nor receives any of it.

| Added | Direction |
| --- | --- |
| `message.location`, `message.contact` | both |
| `message.menu` and the `callback` that answers it | out / in |
| `message.replyTo` | both |
| `status` events | in |

Behaviour that changed for the better:

- Outbound calls are **retried** and end in a visible delivery failure. v1 sent
  once and logged the loss.
- Inbound messages are **deduplicated** on `message.id`.

## Choosing this over the native API

The native messaging API and its SDKs are the better fit when Webitel talks to
the **end user's own device** — a web widget, a mobile app — where each user
holds their own token and their own stream. A custom gateway is the fit when
Webitel talks to **your server**, which is itself the messenger for many users:
two HTTP endpoints instead of one live connection per user, plus a flow schema,
templates and channel attribution that come with being a gateway.
