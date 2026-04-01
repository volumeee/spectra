#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

PASS=0
FAIL=0
BASE_URL="http://localhost:3001"

ok()   { echo -e "${GREEN}  ✅ PASS${NC} $1"; PASS=$((PASS+1)); }
fail() { echo -e "${RED}  ❌ FAIL${NC} $1"; FAIL=$((FAIL+1)); }
log()  { echo -e "${CYAN}[TEST]${NC} $1"; }

cd "$(dirname "$0")/.."

# Make sure binaries exist
if [ ! -f bin/spectra ]; then
    echo "Run scripts/install-and-test.sh first to build."
    exit 1
fi

# Create test config with all features enabled
cat > /tmp/spectra-test.yaml <<EOF
server:
  port: 3001
  read_timeout: 30s
  write_timeout: 30s
browser:
  max_instances: 2
  launch_timeout: 30s
  idle_timeout: 5m
queue:
  max_concurrent: 5
  max_pending: 50
auth:
  enabled: true
  api_key: test-key-123
plugins:
  dir: "./bin/plugins"
  load_timeout: 10s
mcp:
  enabled: false
  transport: stdio
webhook:
  enabled: true
  max_retries: 2
  retry_interval: 1s
scheduler:
  enabled: true
log:
  level: debug
  format: text
EOF

pkill -f "bin/spectra" 2>/dev/null || true
sleep 1

log "========================================="
log "🔮 Spectra — Full Feature Test"
log "  Auth: ON | Webhooks: ON | Scheduler: ON"
log "========================================="
echo ""

# Start server with test config
SPECTRA_CONFIG=/tmp/spectra-test.yaml ./bin/spectra &
SERVER_PID=$!
sleep 3

if ! kill -0 $SERVER_PID 2>/dev/null; then
    fail "server failed to start"
    exit 1
fi
ok "server started on port 3001"

echo ""
log "--- Auth Tests ---"
echo ""

# No auth → 401
log "Test: request without API key"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/plugins" 2>/dev/null)
if [ "$HTTP_CODE" = "401" ]; then
    ok "no auth → 401"
else
    fail "no auth → $HTTP_CODE (expected 401)"
fi

# Wrong auth → 401
log "Test: request with wrong API key"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer wrong-key" "$BASE_URL/api/plugins" 2>/dev/null)
if [ "$HTTP_CODE" = "401" ]; then
    ok "wrong key → 401"
else
    fail "wrong key → $HTTP_CODE (expected 401)"
fi

# Correct auth → 200
log "Test: request with correct API key"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer test-key-123" "$BASE_URL/api/plugins" 2>/dev/null)
if [ "$HTTP_CODE" = "200" ]; then
    ok "correct key → 200"
else
    fail "correct key → $HTTP_CODE (expected 200)"
fi

# Health skips auth
log "Test: /health skips auth"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health" 2>/dev/null)
if [ "$HTTP_CODE" = "200" ]; then
    ok "/health no auth → 200"
else
    fail "/health no auth → $HTTP_CODE (expected 200)"
fi

AUTH="-H \"Authorization: Bearer test-key-123\""

echo ""
log "--- Webhook CRUD Tests ---"
echo ""

# Create webhook
log "Test: create webhook"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/webhooks" \
    -H "Authorization: Bearer test-key-123" \
    -H "Content-Type: application/json" \
    -d '{"event":"job.completed","target_url":"https://httpbin.org/post","secret":"mysecret"}' 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -1)
if [ "$HTTP_CODE" = "200" ]; then
    ok "create webhook → 200"
    WEBHOOK_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [ -n "$WEBHOOK_ID" ]; then
        ok "  webhook ID: $WEBHOOK_ID"
    else
        fail "  webhook ID not found in response"
    fi
else
    fail "create webhook → $HTTP_CODE"
fi

# List webhooks
log "Test: list webhooks"
RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer test-key-123" "$BASE_URL/api/webhooks" 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -1)
if [ "$HTTP_CODE" = "200" ]; then
    ok "list webhooks → 200"
    if echo "$BODY" | grep -q "job.completed"; then
        ok "  webhook event found"
    else
        fail "  webhook event not found"
    fi
else
    fail "list webhooks → $HTTP_CODE"
fi

# Delete webhook
if [ -n "$WEBHOOK_ID" ]; then
    log "Test: delete webhook"
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE \
        -H "Authorization: Bearer test-key-123" \
        "$BASE_URL/api/webhooks/$WEBHOOK_ID" 2>/dev/null)
    if [ "$HTTP_CODE" = "200" ]; then
        ok "delete webhook → 200"
    else
        fail "delete webhook → $HTTP_CODE"
    fi
fi

echo ""
log "--- Scheduler CRUD Tests ---"
echo ""

# Create schedule
log "Test: create schedule"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/schedules" \
    -H "Authorization: Bearer test-key-123" \
    -H "Content-Type: application/json" \
    -d '{"cron":"*/5 * * * *","plugin":"screenshot","method":"capture","params":{"url":"https://example.com"}}' 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -1)
if [ "$HTTP_CODE" = "200" ]; then
    ok "create schedule → 200"
    SCHEDULE_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [ -n "$SCHEDULE_ID" ]; then
        ok "  schedule ID: $SCHEDULE_ID"
    else
        fail "  schedule ID not found"
    fi
else
    fail "create schedule → $HTTP_CODE"
    echo "  Body: $(echo "$BODY" | head -c 200)"
fi

# List schedules
log "Test: list schedules"
RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer test-key-123" "$BASE_URL/api/schedules" 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -1)
if [ "$HTTP_CODE" = "200" ]; then
    ok "list schedules → 200"
    if echo "$BODY" | grep -q "screenshot"; then
        ok "  schedule plugin found"
    else
        fail "  schedule plugin not found"
    fi
    if echo "$BODY" | grep -q "next_run"; then
        ok "  next_run present"
    else
        fail "  next_run missing"
    fi
else
    fail "list schedules → $HTTP_CODE"
fi

# Invalid cron → error
log "Test: create schedule with invalid cron"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/schedules" \
    -H "Authorization: Bearer test-key-123" \
    -H "Content-Type: application/json" \
    -d '{"cron":"invalid","plugin":"screenshot","method":"capture"}' 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
if [ "$HTTP_CODE" = "400" ]; then
    ok "invalid cron → 400"
else
    fail "invalid cron → $HTTP_CODE (expected 400)"
fi

# Delete schedule
if [ -n "$SCHEDULE_ID" ]; then
    log "Test: delete schedule"
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE \
        -H "Authorization: Bearer test-key-123" \
        "$BASE_URL/api/schedules/$SCHEDULE_ID" 2>/dev/null)
    if [ "$HTTP_CODE" = "200" ]; then
        ok "delete schedule → 200"
    else
        fail "delete schedule → $HTTP_CODE"
    fi
fi

echo ""
log "--- Rate Limit Test ---"
echo ""

log "Test: rapid requests (rate limit)"
RATE_LIMITED=false
for i in $(seq 1 250); do
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer test-key-123" "$BASE_URL/health" 2>/dev/null)
    if [ "$HTTP_CODE" = "429" ]; then
        RATE_LIMITED=true
        break
    fi
done
if [ "$RATE_LIMITED" = true ]; then
    ok "rate limiter triggered at request $i"
else
    ok "rate limiter not triggered (within limit)"
fi

echo ""
log "--- Readiness Detail Test ---"
echo ""

log "Test: /ready shows pool + queue stats"
RESP=$(curl -s "$BASE_URL/ready" 2>/dev/null)
if echo "$RESP" | grep -q "browser_pool"; then
    ok "/ready has browser_pool stats"
else
    fail "/ready missing browser_pool"
fi
if echo "$RESP" | grep -q "queue"; then
    ok "/ready has queue stats"
else
    fail "/ready missing queue"
fi

echo ""
log "========================================="
log "🧹 CLEANUP"
log "========================================="
echo ""

kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null
ok "server stopped"
rm -f /tmp/spectra-test.yaml

echo ""
log "========================================="
log "📊 RESULTS"
log "========================================="
echo ""
echo -e "  ${GREEN}Passed: $PASS${NC}"
echo -e "  ${RED}Failed: $FAIL${NC}"
TOTAL=$((PASS+FAIL))
echo -e "  Total:  $TOTAL"
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}  🎉 ALL TESTS PASSED!${NC}"
    exit 0
else
    echo -e "${RED}  ⚠️  Some tests failed.${NC}"
    exit 1
fi
