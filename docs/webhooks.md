# 🔔 Webhooks

Get notified when events happen in Spectra. Requires `webhook.enabled: true` in config.

## Events

| Event | Trigger |
|-------|---------|
| `job.completed` | A job finishes successfully |
| `job.failed` | A job fails |
| `plugin.crashed` | A plugin process crashes |

## Enable

```yaml
webhook:
  enabled: true
  max_retries: 3
  retry_interval: 5s
```

## CRUD

```bash
# Create
curl -X POST http://localhost:3000/api/webhooks \
  -H "Content-Type: application/json" \
  -d '{"event":"job.completed","target_url":"https://your-server.com/webhook","secret":"your-hmac-secret"}'

# List
curl http://localhost:3000/api/webhooks

# Delete
curl -X DELETE http://localhost:3000/api/webhooks/{id}
```

## Payload

```json
{
  "event": "job.completed",
  "timestamp": "2026-04-01T00:00:00Z",
  "data": {
    "job_id": "uuid",
    "plugin": "screenshot",
    "method": "capture",
    "result": { ... }
  }
}
```

## HMAC Verification

If you set a `secret`, Spectra signs the payload with HMAC-SHA256 in the `X-Spectra-Signature` header.

```python
import hmac, hashlib

def verify(body: bytes, signature: str, secret: str) -> bool:
    expected = hmac.new(secret.encode(), body, hashlib.sha256).hexdigest()
    return hmac.compare_digest(expected, signature)
```

## Retry

Failed deliveries retry with linear backoff: `retry_interval * attempt_number`.

## Persistence

With `storage.driver: sqlite`, webhook subscriptions persist across server restarts.
With `storage.driver: memory` (default), subscriptions are lost on restart.
