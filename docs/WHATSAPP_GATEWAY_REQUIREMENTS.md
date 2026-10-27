# WhatsApp Gateway — Requirements Specification

**Project:** Event Face Photo Finder + WhatsApp Delivery Platform
**Document:** WHATSAPP-GATEWAY-REQ v1.0
**Purpose:** Define exactly what the existing company WhatsApp gateway must provide so the FaceFinder backend (Go) can integrate with it. The gateway is a transport layer only — all event, session, face-search, and gallery logic lives in the FaceFinder backend.

---

## 1. Integration Model

```
Guest ──WhatsApp──▶ [ WhatsApp Gateway ] ──webhook──▶ [ FaceFinder Go Backend ]
Guest ◀─WhatsApp──  [ WhatsApp Gateway ] ◀──API call── [ FaceFinder Go Backend ]
```

- One WhatsApp business number serves **unlimited events**. Events are distinguished by a token in the guest's first message (`START <event-token>`), never by phone number.
- The gateway must not implement event/session logic. It only delivers messages/media to the backend and sends what the backend asks it to send.
- A WhatsApp phone number is a **delivery address only** — it is never treated as verified biometric identity.

---

## 2. Required Functionalities

### 2.1 Must-Have (MVP — integration cannot work without these)

| ID | Requirement | Used for |
|---|---|---|
| WA-01 | Incoming message webhook (text + image) | Receiving `START <token>`, guest selfies, button replies |
| WA-02 | Media download API by media ID | Downloading the guest selfie for face search |
| WA-03 | Send text message API | Bot prompts, errors, and result messages |
| WA-04 | Send image message API (image via public URL + optional caption) | Sending matched photo previews (max 5 per search) |

### 2.2 Required for Full Experience

| ID | Requirement | Fallback if unavailable |
|---|---|---|
| WA-05 | Send interactive message with a CTA-URL button ("View All Photos") | Gallery link sent as plain text |
| WA-06 | Send interactive message with reply buttons (max 3) | Guest types commands as text (`SEARCH_AGAIN`, etc.) |
| WA-07 | Message status webhook (sent / delivered / read / failed) | No delivery tracking; support relies on guest reports |
| WA-08 | Button-reply events delivered through the incoming webhook with the button ID intact | Buttons in WA-05/WA-06 become display-only |

### 2.3 Nice-to-Have

| ID | Requirement |
|---|---|
| WA-09 | Typing indicator / read receipt marking ("blue ticks") before long searches |
| WA-10 | Send document API (for future "Download all as ZIP") |
| WA-11 | Per-message cost/usage reporting for billing reconciliation |

---

## 3. Webhook Specification (Gateway → FaceFinder)

### 3.1 Endpoints to configure in the gateway portal

| Purpose | URL (backend) | Method |
|---|---|---|
| Incoming messages | `https://<facefinder-domain>/api/v1/webhooks/whatsapp` | POST |
| Message status | `https://<facefinder-domain>/api/v1/webhooks/whatsapp/status` | POST |

For local development the same paths are exposed on `http://localhost:8080` and reached via a tunnel (ngrok / Cloudflare Tunnel).

### 3.2 Incoming message payload

Content type: `application/json`. One webhook call per message.

| Field | Type | Required | Description |
|---|---|---|---|
| `phone_number` | string | Yes | Guest number in E.164 (`+91XXXXXXXXXX` or `91XXXXXXXXXX`) |
| `gateway_number` | string | Yes | The business number that received the message (distinguishes numbers if several are connected) |
| `message_id` | string | Yes | Gateway/WhatsApp message ID, unique per message (used for deduplication) |
| `message_type` | string | Yes | `text`, `image`, or `interactive` (button reply) |
| `text` | string | For text | Message body, e.g. `START abc123...` |
| `media_id` | string | For image | ID passed to the Media Download API (WA-02) |
| `mime_type` | string | For image | e.g. `image/jpeg` |
| `button_id` | string | For interactive | ID of the tapped button, e.g. `SEARCH_AGAIN`, `CHANGE_EVENT` |
| `timestamp` | number | Yes | Unix epoch seconds |

Example — event entry:

```json
{
  "phone_number": "+919876543210",
  "gateway_number": "919000011122",
  "message_id": "wamid.HBg...",
  "message_type": "text",
  "text": "START 9f2c7ab41e...",
  "timestamp": 1793700000
}
```

Example — selfie:

```json
{
  "phone_number": "+919876543210",
  "gateway_number": "919000011122",
  "message_id": "wamid.HBg...",
  "message_type": "image",
  "media_id": "media_8841207",
  "mime_type": "image/jpeg",
  "timestamp": 1793700060
}
```

> If the gateway's native field names differ, provide one real sample payload per message type — the FaceFinder parser will be adapted to the gateway, not the reverse.

### 3.3 Webhook behaviour contract

- Backend responds `200 {"ok":true}` on successful receipt (including handled errors such as "no active event session").
- Gateway should retry on non-200 responses / timeouts with backoff (suggested: 3 attempts over 5 minutes). Retries must reuse the same `message_id` so the backend can deduplicate.
- Webhook delivery timeout expected by gateway should be ≥ 10 seconds (face search itself runs asynchronously; the webhook returns before results are sent).

### 3.4 Webhook security

At least one of the following must be supported; the backend implements option (a) today, (b)/(c) on request:

a. Shared secret header sent with every webhook (e.g. `X-WhatsApp-Secret: <secret>`), configured identically in the portal and in backend env `WHATSAPP_WEBHOOK_SECRET`.
b. HMAC-SHA256 signature of the raw request body in a header (preferred for production; provide header name and signing scheme).
c. Source IP allow-listing (provide gateway egress IPs).

### 3.5 Message status payload (WA-07)

```json
{
  "message_id": "wamid.HBg...",
  "phone_number": "+919876543210",
  "status": "sent | delivered | read | failed",
  "timestamp": 1793700100,
  "error": "optional failure reason"
}
```

---

## 4. API Specification (FaceFinder → Gateway)

All calls are HTTPS, JSON, server-to-server. Authentication: **Bearer API key** (`Authorization: Bearer <key>`) unless the gateway documents a different scheme — in which case provide it and the backend adapts.

### 4.1 Send text (WA-03)

`POST {BASE_URL}/send/text`

```json
{ "to": "+919876543210", "text": "Welcome to Rahul Wedding! Please send one clear selfie containing only your face." }
```

Expected: HTTP 2xx with a gateway message ID in the response (field name to be confirmed from gateway docs).

### 4.2 Send image (WA-04)

`POST {BASE_URL}/send/image`

```json
{ "to": "+919876543210", "image_url": "https://<signed-r2-url>", "caption": "Rahul Wedding" }
```

- `image_url` is a **short-lived signed URL (15 minutes)** to a private Cloudflare R2 object. The gateway must fetch it promptly and must support HTTPS GET of JPEG images up to ~5 MB.
- The gateway must not require the image to be hosted on a public/permanent URL.

### 4.3 Send interactive (WA-05 / WA-06)

`POST {BASE_URL}/send/interactive`

```json
{
  "to": "+919876543210",
  "text": "We found 12 photos of you in Rahul Wedding.",
  "buttons": { "view_all": "https://<web>/g/<gallery-token>", "search_again": "SEARCH_AGAIN" }
}
```

Semantic requirement (representation may differ per gateway API):
- `view_all` → CTA URL button that opens the gallery link in a browser.
- Other entries → reply buttons; tapping one must produce an incoming webhook (WA-08) carrying that button's ID.

### 4.4 Media download (WA-02)

`GET {BASE_URL}/media/{media_id}` (Authorization as above)

- Returns the raw image bytes (`Content-Type: image/jpeg` or actual type), max expected size 25 MB.
- Must be callable at least 1 hour after the webhook that referenced the media (backend processes asynchronously and may retry).
- Must return a non-2xx status (not an empty body) when media is expired/unavailable so the backend can ask the guest to resend.

---

## 5. Conversation Flow the Gateway Must Support

1. Guest scans event QR → WhatsApp opens to the business number with prefilled `START <event-token>` → guest sends it. *(Deep link is standard `https://wa.me/<number>?text=...`; no gateway involvement.)*
2. Backend creates the event session (phone + gateway number, TTL 24h default) and replies via Send Text: *"Please send one clear selfie containing only your face."*
3. Guest sends selfie → image webhook → backend downloads media → face validation:
   - 0 faces → Send Text: *"No clear face was detected…"*
   - >1 face → Send Text: *"Please send a selfie containing only one person."*
4. Search runs only against that event's face collection. Results:
   - 0 matches → Send Text asking for a clearer selfie.
   - 1–5 matches → Send Image previews.
   - >5 matches → 3–5 Send Image previews + interactive/text: *"We found X photos of you"* + gallery link.
5. Guest taps **View All Photos** (opens private web gallery) / **Search Again** (button webhook → backend re-prompts) / **Change Event** (button webhook → backend asks them to scan the new event's QR).
6. Selfie without an active session → backend replies *"Please scan the QR code available at your event before sending your selfie."* The gateway requires no special handling.

Throughput expectation: one search produces at most ~7 outbound messages. The gateway should sustain at least **20 outbound messages/second** burst for event peaks (e.g. reception exit) — confirm actual rate limits from the portal.

---

## 6. Configuration Checklist (Portal Side)

- [ ] Business number connected and verified; number provided to FaceFinder team.
- [ ] Incoming webhook URL set (§3.1) and a test message confirmed delivered.
- [ ] Message status webhook URL set (if WA-07 supported).
- [ ] Webhook secret/signature configured and value handed over securely (goes into backend `.env`, never into source code or chat).
- [ ] API key issued for server-to-server sending/media download.
- [ ] Rate limits, daily limits, and media retention period documented.
- [ ] Template-message policy clarified: the flow above uses **session (24h customer-service window) messages** opened by the guest, so pre-approved templates are not required for replies within 24h of the guest's last message. Confirm the gateway/WhatsApp setup agrees; if proactive messages outside the window are ever needed, templates will be specified separately.

## 7. Credentials (to be filled in backend `.env` by the FaceFinder operator)

```text
WHATSAPP_GATEWAY_BASE_URL=      # e.g. https://gateway.example.com/api
WHATSAPP_GATEWAY_API_KEY=       # server-to-server API key
WHATSAPP_WEBHOOK_SECRET=        # webhook shared secret (§3.4), if used
WHATSAPP_BUSINESS_NUMBER=       # country code + number, no '+' or spaces
```

## 8. Deliverables Requested from the Gateway Team

1. Base URL + API reference (or Postman/OpenAPI collection) for send-text, send-image, send-interactive, and media-download.
2. One real sample payload each: incoming text webhook, incoming image webhook, button-reply webhook, status webhook.
3. Authentication scheme for API calls and webhook security scheme (§3.4).
4. Rate limits, message-size limits, media retention period, and error-code list.
5. Business number(s) connected to the gateway.

## 9. Acceptance Tests (joint, after integration)

| # | Test | Expected |
|---|---|---|
| T1 | Send `START <valid-token>` | Welcome text received; session created for correct event |
| T2 | Send `START <invalid-token>` | "Invalid or expired event link" text |
| T3 | Send selfie with no session | "Please scan the QR code…" text; **no** search performed |
| T4 | Send selfie with no face / group photo as selfie | Respective rejection text (§5.3) |
| T5 | Send valid selfie, matches exist | Previews received (≤5 images), then result message with working gallery link |
| T6 | Send valid selfie, no matches | "Couldn't find matching photos" text |
| T7 | Tap Search Again / Change Event | Backend re-prompts / asks for QR scan |
| T8 | Scan a second event's QR, send selfie | Results come only from the second event |
| T9 | Kill backend, send message, restore backend | Gateway retries webhook; message processed once (same `message_id`) |
| T10 | Status webhooks enabled | sent/delivered/read states recorded for an outbound preview |
