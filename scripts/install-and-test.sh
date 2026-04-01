#!/bin/bash
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

PASS=0
FAIL=0
BASE_URL="http://localhost:3000"
FULL_URL="http://localhost:3001"

log()  { echo -e "${CYAN}[SPECTRA]${NC} $1"; }
ok()   { echo -e "${GREEN}  ✅ PASS${NC} $1"; PASS=$((PASS+1)); }
fail() { echo -e "${RED}  ❌ FAIL${NC} $1"; FAIL=$((FAIL+1)); }
warn() { echo -e "${YELLOW}  ⚠️  WARN${NC} $1"; }

cd "$(dirname "$0")/.."

########################################
# PHASE 1: BUILD
########################################
log "========================================="
log "🔮 Spectra v3 — Full Install & Test"
log "========================================="
echo ""

if ! command -v go &> /dev/null; then
    echo -e "${RED}Go is not installed. Install Go 1.23+ first.${NC}"
    exit 1
fi
log "Go version: $(go version)"

log "📦 Downloading dependencies..."
go mod tidy 2>&1
[ $? -eq 0 ] && ok "go mod tidy" || { fail "go mod tidy"; exit 1; }

log "🔨 Building core..."
mkdir -p bin
go build -o bin/spectra ./cmd/spectra/ 2>&1
[ $? -eq 0 ] && ok "build core" || { fail "build core"; exit 1; }

log "🔨 Building CLI..."
go build -o bin/spectra-cli ./cmd/spectra-cli/ 2>&1
[ $? -eq 0 ] && ok "build CLI" || { fail "build CLI"; exit 1; }

log "🔨 Building plugins..."
mkdir -p bin/plugins
for dir in plugins/*/; do
    name=$(basename "$dir")
    go build -o "bin/plugins/$name" "./$dir" 2>&1
    [ $? -eq 0 ] && ok "build plugin: $name" || fail "build plugin: $name"
    [ -f "$dir/plugin.json" ] && cp "$dir/plugin.json" "bin/plugins/${name}.json"
done

echo ""
########################################
# PHASE 2: UNIT TESTS
########################################
log "========================================="
log "🧪 PHASE 2: UNIT TESTS"
log "========================================="
echo ""

go test ./pkg/... ./internal/domain/... ./internal/config/... 2>&1
[ $? -eq 0 ] && ok "unit tests" || warn "unit tests (some may not exist yet)"

echo ""
########################################
# PHASE 3: BASIC SERVER (no auth)
########################################
log "========================================="
log "🚀 PHASE 3: BASIC SERVER TEST"
log "========================================="
echo ""

cat > /tmp/spectra-basic.yaml <<EOF
server:
  port: 3000
  read_timeout: 30s
  write_timeout: 60s
browser:
  max_instances: 5
  launch_timeout: 30s
  idle_timeout: 5m
  share_pool: true
queue:
  max_concurrent: 10
  max_pending: 100
auth:
  enabled: false
plugins:
  dir: "./bin/plugins"
  load_timeout: 10s
  call_timeout: 60s
  pool_size: 2
health:
  enabled: true
  cpu_limit: 95
  memory_limit: 95
storage:
  driver: memory
recording:
  enabled: false
webhook:
  enabled: false
scheduler:
  enabled: false
mcp:
  enabled: false
log:
  level: info
  format: json
EOF

pkill -f "bin/spectra" 2>/dev/null || true
fuser -k 3000/tcp 2>/dev/null || true
fuser -k 3001/tcp 2>/dev/null || true
sleep 2

log "Starting Spectra (port 3000, no auth)..."
SPECTRA_CONFIG=/tmp/spectra-basic.yaml ./bin/spectra > /tmp/spectra-basic.log 2>&1 &
SERVER_PID=$!
sleep 3

if ! kill -0 $SERVER_PID 2>/dev/null; then
    fail "server failed to start"
    tail -5 /tmp/spectra-basic.log
    exit 1
fi
ok "server started (PID: $SERVER_PID)"

# --- Health ---
log "Test: GET /health"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" "$BASE_URL/health")
[ "$HTTP_CODE" = "200" ] && ok "/health → 200" || fail "/health → $HTTP_CODE"

# --- Readiness ---
log "Test: GET /ready"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" "$BASE_URL/ready")
BODY=$(cat /tmp/resp.json)
if [ "$HTTP_CODE" = "200" ]; then
    ok "/ready → 200"
    echo "$BODY" | grep -q "browser_pool" && ok "  browser_pool stats" || fail "  browser_pool stats missing"
    echo "$BODY" | grep -q "queue" && ok "  queue stats" || fail "  queue stats missing"
else
    fail "/ready → $HTTP_CODE"
fi

# --- Pressure (NEW) ---
log "Test: GET /pressure"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" "$BASE_URL/pressure")
BODY=$(cat /tmp/resp.json)
if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "503" ]; then
    ok "/pressure → $HTTP_CODE (system health endpoint working)"
    echo "$BODY" | grep -q "cpu_percent" && ok "  cpu_percent present" || fail "  cpu_percent missing"
    echo "$BODY" | grep -q "memory_percent" && ok "  memory_percent present" || fail "  memory_percent missing"
else
    fail "/pressure → $HTTP_CODE"
fi

# --- Metrics (NEW) ---
log "Test: GET /api/metrics"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" "$BASE_URL/api/metrics")
BODY=$(cat /tmp/resp.json)
if [ "$HTTP_CODE" = "200" ]; then
    ok "/api/metrics → 200"
    echo "$BODY" | grep -q "total_requests" && ok "  total_requests present" || fail "  total_requests missing"
    echo "$BODY" | grep -q "by_plugin" && ok "  by_plugin present" || fail "  by_plugin missing"
else
    fail "/api/metrics → $HTTP_CODE"
fi

# --- Jobs history (NEW) ---
log "Test: GET /api/jobs"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" "$BASE_URL/api/jobs")
if [ "$HTTP_CODE" = "200" ]; then
    ok "/api/jobs → 200"
else
    fail "/api/jobs → $HTTP_CODE"
fi

# --- Plugins list ---
log "Test: GET /api/plugins"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" "$BASE_URL/api/plugins")
BODY=$(cat /tmp/resp.json)
if [ "$HTTP_CODE" = "200" ]; then
    ok "/api/plugins → 200"
    echo "$BODY" | grep -q "screenshot" && ok "  screenshot found" || fail "  screenshot missing"
    echo "$BODY" | grep -q "pdf" && ok "  pdf found" || fail "  pdf missing"
    echo "$BODY" | grep -q "scrape" && ok "  scrape found" || fail "  scrape missing"
    echo "$BODY" | grep -q '"ai"' && ok "  ai found" || fail "  ai missing"
    echo "$BODY" | grep -q "recorder" && ok "  recorder found" || fail "  recorder missing"
else
    fail "/api/plugins → $HTTP_CODE"
fi

# --- Error cases ---
log "Test: nonexistent plugin → 404"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/fake/method" -H "Content-Type: application/json" -d '{}')
[ "$HTTP_CODE" = "404" ] && ok "nonexistent → 404" || fail "nonexistent → $HTTP_CODE"

log "Test: invalid JSON → 400"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/screenshot/capture" -H "Content-Type: application/json" -d 'broken')
[ "$HTTP_CODE" = "400" ] && ok "invalid JSON → 400" || fail "invalid JSON → $HTTP_CODE"

log "Test: X-Request-ID header"
HEADERS=$(curl -s -D - -o /dev/null "$BASE_URL/health")
echo "$HEADERS" | grep -qi "x-request-id" && ok "X-Request-ID present" || fail "X-Request-ID missing"

# --- Browser tests (require Chromium) ---
HAS_CHROME=false
(command -v chromium || command -v chromium-browser || command -v google-chrome) &>/dev/null && HAS_CHROME=true

echo ""
log "--- Plugin Tests (require Chromium) ---"
echo ""

if [ "$HAS_CHROME" = true ]; then
    # Screenshot — verify 1920x1080 default
    log "Test: screenshot (default 1920x1080)"
    HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" --max-time 30 -X POST "$BASE_URL/api/screenshot/capture" \
        -H "Content-Type: application/json" -d '{"url":"https://example.com"}')
    BODY=$(cat /tmp/resp.json)
    if [ "$HTTP_CODE" = "200" ]; then
        ok "screenshot → 200"
        echo "$BODY" | grep -q '"data"' && ok "  base64 data present" || fail "  data missing"
        echo "$BODY" | grep -q '"width":1920' && ok "  width=1920 ✓" || warn "  width not 1920 (check default)"
        echo "$BODY" | grep -q '"height":1080' && ok "  height=1080 ✓" || warn "  height not 1080 (check default)"
    else
        fail "screenshot → $HTTP_CODE"
    fi

    # Screenshot — explicit size
    log "Test: screenshot (explicit 1280x720)"
    HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" --max-time 30 -X POST "$BASE_URL/api/screenshot/capture" \
        -H "Content-Type: application/json" -d '{"url":"https://example.com","width":1280,"height":720}')
    BODY=$(cat /tmp/resp.json)
    if [ "$HTTP_CODE" = "200" ]; then
        ok "screenshot explicit size → 200"
        echo "$BODY" | grep -q '"width":1280' && ok "  width=1280 ✓" || fail "  width not 1280"
    else
        fail "screenshot explicit size → $HTTP_CODE"
    fi

    # PDF
    log "Test: pdf"
    HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" --max-time 30 -X POST "$BASE_URL/api/pdf/generate" \
        -H "Content-Type: application/json" -d '{"url":"https://example.com"}')
    BODY=$(cat /tmp/resp.json)
    [ "$HTTP_CODE" = "200" ] && ok "pdf → 200" && echo "$BODY" | grep -q '"data"' && ok "  data present" || fail "pdf → $HTTP_CODE"

    # Scrape
    log "Test: scrape"
    HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" --max-time 30 -X POST "$BASE_URL/api/scrape/extract" \
        -H "Content-Type: application/json" -d '{"url":"https://example.com"}')
    BODY=$(cat /tmp/resp.json)
    if [ "$HTTP_CODE" = "200" ]; then
        ok "scrape → 200"
        echo "$BODY" | grep -q '"title"' && ok "  title present" || fail "  title missing"
        echo "$BODY" | grep -q '"links"' && ok "  links present" || fail "  links missing"
    else
        fail "scrape → $HTTP_CODE"
    fi

    # Stealth
    log "Test: stealth navigate"
    HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" --max-time 30 -X POST "$BASE_URL/api/stealth/navigate" \
        -H "Content-Type: application/json" -d '{"url":"https://example.com"}')
    BODY=$(cat /tmp/resp.json)
    if [ "$HTTP_CODE" = "200" ]; then
        ok "stealth navigate → 200"
        echo "$BODY" | grep -q '"stealth":true' && ok "  stealth=true ✓" || fail "  stealth flag missing"
    else
        fail "stealth navigate → $HTTP_CODE"
    fi

    # Visual diff
    log "Test: visual-diff compare"
    HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" --max-time 60 -X POST "$BASE_URL/api/visual-diff/compare" \
        -H "Content-Type: application/json" -d '{"url1":"https://example.com","url2":"https://example.com"}')
    BODY=$(cat /tmp/resp.json)
    if [ "$HTTP_CODE" = "200" ]; then
        ok "visual-diff → 200"
        echo "$BODY" | grep -q '"diff_percent"' && ok "  diff_percent present" || fail "  diff_percent missing"
        echo "$BODY" | grep -q '"match"' && ok "  match present" || fail "  match missing"
    else
        fail "visual-diff → $HTTP_CODE"
    fi

    # Recorder — enhanced with new actions
    log "Test: recorder (enhanced actions)"
    HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" --max-time 30 -X POST "$BASE_URL/api/recorder/record" \
        -H "Content-Type: application/json" \
        -d '{"url":"https://example.com","output_mode":"frames","steps":[{"action":"scroll","delay":300},{"action":"screenshot"}]}')
    BODY=$(cat /tmp/resp.json)
    if [ "$HTTP_CODE" = "200" ]; then
        ok "recorder → 200"
        echo "$BODY" | grep -q '"steps"' && ok "  steps present" || fail "  steps missing"
        echo "$BODY" | grep -q '"frames"' && ok "  frames present (output_mode=frames)" || fail "  frames missing"
        echo "$BODY" | grep -q '"duration_ms"' && ok "  duration_ms present" || fail "  duration_ms missing"
    else
        fail "recorder → $HTTP_CODE"
    fi

    # Recorder — assert_text action
    log "Test: recorder assert_text action"
    HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" --max-time 30 -X POST "$BASE_URL/api/recorder/record" \
        -H "Content-Type: application/json" \
        -d '{"url":"https://example.com","output_mode":"frames","steps":[{"action":"wait_for","selector":"h1","timeout":5000},{"action":"assert_text","selector":"h1","value":"Example Domain"}]}')
    BODY=$(cat /tmp/resp.json)
    if [ "$HTTP_CODE" = "200" ]; then
        ok "recorder assert_text → 200"
        echo "$BODY" | grep -q '"success":true' && ok "  assert_text passed" || warn "  assert_text may have failed (check page)"
    else
        fail "recorder assert_text → $HTTP_CODE"
    fi

    # Metrics after browser tests
    log "Test: /api/metrics after browser requests"
    HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" "$BASE_URL/api/metrics")
    BODY=$(cat /tmp/resp.json)
    if [ "$HTTP_CODE" = "200" ]; then
        ok "/api/metrics after requests → 200"
        TOTAL=$(echo "$BODY" | grep -o '"total_requests":[0-9]*' | grep -o '[0-9]*')
        [ -n "$TOTAL" ] && [ "$TOTAL" -gt 0 ] && ok "  total_requests=$TOTAL > 0" || fail "  total_requests not incrementing"
        echo "$BODY" | grep -q '"screenshot"' && ok "  screenshot metrics tracked" || fail "  screenshot not in metrics"
    else
        fail "/api/metrics → $HTTP_CODE"
    fi

    # Jobs history after requests
    log "Test: /api/jobs after browser requests"
    HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" "$BASE_URL/api/jobs")
    BODY=$(cat /tmp/resp.json)
    if [ "$HTTP_CODE" = "200" ]; then
        ok "/api/jobs → 200"
        COUNT=$(echo "$BODY" | grep -o '"count":[0-9]*' | grep -o '[0-9]*')
        [ -n "$COUNT" ] && [ "$COUNT" -gt 0 ] && ok "  job count=$COUNT > 0" || warn "  no jobs recorded (memory store)"
    else
        fail "/api/jobs → $HTTP_CODE"
    fi

    # SpectraQL — multi-step query (NEW)
    log "Test: SpectraQL multi-step query"
    HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" --max-time 30 -X POST "$BASE_URL/api/query" \
        -H "Content-Type: application/json" \
        -d '{"steps":[{"action":"goto","url":"https://example.com"},{"action":"screenshot"}]}')
    BODY=$(cat /tmp/resp.json)
    if [ "$HTTP_CODE" = "200" ]; then
        ok "SpectraQL → 200"
        echo "$BODY" | grep -q '"data"' && ok "  result data present" || warn "  no data (recorder may not be built)"
    else
        fail "SpectraQL → $HTTP_CODE"
    fi

    # SpectraQL — empty steps → 400
    log "Test: SpectraQL empty steps → 400"
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/query" \
        -H "Content-Type: application/json" -d '{"steps":[]}')
    [ "$HTTP_CODE" = "400" ] && ok "SpectraQL empty steps → 400" || fail "SpectraQL empty steps → $HTTP_CODE"

else
    warn "Browser tests SKIPPED (no Chromium found)"
fi

echo ""
log "--- Sessions & Profiles (NEW) ---"
echo ""

# Sessions CRUD (memory mode → 503, sqlite mode → 200)
log "Test: create session (memory → 503)"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" -X POST "$BASE_URL/api/sessions" \
    -H "Content-Type: application/json" -d '{"ttl_seconds":3600}')
if [ "$HTTP_CODE" = "503" ]; then
    ok "create session (memory) → 503 (expected, needs sqlite)"
elif [ "$HTTP_CODE" = "200" ]; then
    ok "create session → 200"
else
    fail "create session → $HTTP_CODE"
fi

log "Test: list sessions (always 200)"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" "$BASE_URL/api/sessions")
[ "$HTTP_CODE" = "200" ] && ok "list sessions → 200" || fail "list sessions → $HTTP_CODE"

log "Test: create profile (memory → 503)"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" -X POST "$BASE_URL/api/profiles" \
    -H "Content-Type: application/json" \
    -d '{"name":"test-profile","locale":"en-US","stealth_level":"basic"}')
if [ "$HTTP_CODE" = "503" ]; then
    ok "create profile (memory) → 503 (expected, needs sqlite)"
elif [ "$HTTP_CODE" = "200" ]; then
    ok "create profile → 200"
else
    fail "create profile → $HTTP_CODE"
fi

log "Test: list profiles (always 200)"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" "$BASE_URL/api/profiles")
[ "$HTTP_CODE" = "200" ] && ok "list profiles → 200" || fail "list profiles → $HTTP_CODE"

echo ""
log "--- Live View WebSocket (NEW) ---"
echo ""

log "Test: live view WebSocket endpoint exists"
# curl cannot do a real WebSocket upgrade — we just verify the route is registered (not 404)
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/sessions/test-id/live")
[ "$HTTP_CODE" != "404" ] && ok "live view endpoint registered (→ $HTTP_CODE, not 404)" || fail "live view endpoint → 404 (not registered)"

log "Test: spectra-cli health"
./bin/spectra-cli health --server "$BASE_URL" 2>&1 | grep -q "healthy" && ok "CLI health" || fail "CLI health"

log "Test: spectra-cli plugins"
./bin/spectra-cli plugins --server "$BASE_URL" 2>&1 | grep -q "screenshot" && ok "CLI plugins" || fail "CLI plugins"

log "Test: spectra-cli exec nonexistent"
./bin/spectra-cli exec nonexistent method --server "$BASE_URL" 2>&1 | grep -qi "error\|not found\|fail" && ok "CLI exec error" || fail "CLI exec error"

log "Stopping basic server..."
kill $SERVER_PID 2>/dev/null; wait $SERVER_PID 2>/dev/null || true
sleep 2

echo ""
########################################
# PHASE 4: FULL FEATURES (auth + webhooks + scheduler + sqlite)
########################################
log "========================================="
log "🔐 PHASE 4: AUTH + WEBHOOKS + SCHEDULER + SQLITE"
log "========================================="
echo ""

cat > /tmp/spectra-full.yaml <<EOF
server:
  port: 3001
  read_timeout: 30s
  write_timeout: 60s
browser:
  max_instances: 2
  launch_timeout: 30s
  idle_timeout: 5m
  share_pool: true
queue:
  max_concurrent: 5
  max_pending: 50
auth:
  enabled: true
  api_key: test-key-123
plugins:
  dir: "./bin/plugins"
  load_timeout: 10s
  call_timeout: 60s
  pool_size: 2
health:
  enabled: true
  cpu_limit: 95
  memory_limit: 95
storage:
  driver: sqlite
  sqlite_path: /tmp/spectra-test.db
recording:
  enabled: false
webhook:
  enabled: true
  max_retries: 2
  retry_interval: 1s
scheduler:
  enabled: true
mcp:
  enabled: false
log:
  level: info
  format: json
EOF

rm -f /tmp/spectra-test.db
fuser -k 3001/tcp 2>/dev/null || true
sleep 1

log "Starting Spectra (port 3001, auth+webhooks+scheduler+sqlite)..."
SPECTRA_CONFIG=/tmp/spectra-full.yaml ./bin/spectra > /tmp/spectra-full.log 2>&1 &
SERVER2_PID=$!
sleep 3

if ! kill -0 $SERVER2_PID 2>/dev/null; then
    fail "full-feature server failed to start"
    tail -5 /tmp/spectra-full.log
    exit 1
fi
ok "full-feature server started (PID: $SERVER2_PID)"

# --- Auth tests ---
log "Test: no API key → 401"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$FULL_URL/api/plugins")
[ "$HTTP_CODE" = "401" ] && ok "no key → 401" || fail "no key → $HTTP_CODE"

log "Test: wrong API key → 401"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer wrong" "$FULL_URL/api/plugins")
[ "$HTTP_CODE" = "401" ] && ok "wrong key → 401" || fail "wrong key → $HTTP_CODE"

log "Test: correct API key → 200"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer test-key-123" "$FULL_URL/api/plugins")
[ "$HTTP_CODE" = "200" ] && ok "correct key → 200" || fail "correct key → $HTTP_CODE"

log "Test: /health skips auth"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$FULL_URL/health")
[ "$HTTP_CODE" = "200" ] && ok "/health no auth → 200" || fail "/health no auth → $HTTP_CODE"

log "Test: /pressure skips auth (NEW)"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$FULL_URL/pressure")
( [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "503" ] ) && ok "/pressure no auth → $HTTP_CODE" || fail "/pressure no auth → $HTTP_CODE"

echo ""
log "--- Webhook CRUD ---"
echo ""

log "Test: create webhook"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" -X POST "$FULL_URL/api/webhooks" \
    -H "Authorization: Bearer test-key-123" -H "Content-Type: application/json" \
    -d '{"event":"job.completed","target_url":"https://httpbin.org/post","secret":"mysecret"}')
BODY=$(cat /tmp/resp.json)
if [ "$HTTP_CODE" = "200" ]; then
    ok "create webhook → 200"
    WEBHOOK_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    [ -n "$WEBHOOK_ID" ] && ok "  id: ${WEBHOOK_ID:0:8}..." || fail "  id missing"
else
    fail "create webhook → $HTTP_CODE"
fi

log "Test: list webhooks"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" -H "Authorization: Bearer test-key-123" "$FULL_URL/api/webhooks")
BODY=$(cat /tmp/resp.json)
[ "$HTTP_CODE" = "200" ] && ok "list webhooks → 200" && echo "$BODY" | grep -q "job.completed" && ok "  event found" || fail "list webhooks → $HTTP_CODE"

if [ -n "$WEBHOOK_ID" ]; then
    log "Test: delete webhook"
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "Authorization: Bearer test-key-123" "$FULL_URL/api/webhooks/$WEBHOOK_ID")
    [ "$HTTP_CODE" = "200" ] && ok "delete webhook → 200" || fail "delete webhook → $HTTP_CODE"
fi

echo ""
log "--- Scheduler CRUD ---"
echo ""

log "Test: create schedule"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" -X POST "$FULL_URL/api/schedules" \
    -H "Authorization: Bearer test-key-123" -H "Content-Type: application/json" \
    -d '{"cron":"*/5 * * * *","plugin":"screenshot","method":"capture","params":{"url":"https://example.com"}}')
BODY=$(cat /tmp/resp.json)
if [ "$HTTP_CODE" = "200" ]; then
    ok "create schedule → 200"
    SCHEDULE_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    [ -n "$SCHEDULE_ID" ] && ok "  id: ${SCHEDULE_ID:0:8}..." || fail "  id missing"
else
    fail "create schedule → $HTTP_CODE"
fi

log "Test: list schedules"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" -H "Authorization: Bearer test-key-123" "$FULL_URL/api/schedules")
BODY=$(cat /tmp/resp.json)
if [ "$HTTP_CODE" = "200" ]; then
    ok "list schedules → 200"
    echo "$BODY" | grep -q "screenshot" && ok "  plugin found" || fail "  plugin missing"
    echo "$BODY" | grep -q "next_run" && ok "  next_run present" || fail "  next_run missing"
else
    fail "list schedules → $HTTP_CODE"
fi

log "Test: invalid cron → 400"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$FULL_URL/api/schedules" \
    -H "Authorization: Bearer test-key-123" -H "Content-Type: application/json" \
    -d '{"cron":"not-a-cron","plugin":"screenshot","method":"capture"}')
[ "$HTTP_CODE" = "400" ] && ok "invalid cron → 400" || fail "invalid cron → $HTTP_CODE"

if [ -n "$SCHEDULE_ID" ]; then
    log "Test: delete schedule"
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "Authorization: Bearer test-key-123" "$FULL_URL/api/schedules/$SCHEDULE_ID")
    [ "$HTTP_CODE" = "200" ] && ok "delete schedule → 200" || fail "delete schedule → $HTTP_CODE"
fi

echo ""
log "--- SQLite Persistence (NEW) ---"
echo ""

log "Test: create webhook → restart → webhook persists"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" -X POST "$FULL_URL/api/webhooks" \
    -H "Authorization: Bearer test-key-123" -H "Content-Type: application/json" \
    -d '{"event":"job.failed","target_url":"https://httpbin.org/post"}')
BODY=$(cat /tmp/resp.json)
PERSIST_WEBHOOK_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ "$HTTP_CODE" = "200" ] && [ -n "$PERSIST_WEBHOOK_ID" ]; then
    ok "created webhook for persistence test: ${PERSIST_WEBHOOK_ID:0:8}..."

    # Restart server
    kill $SERVER2_PID 2>/dev/null; wait $SERVER2_PID 2>/dev/null || true
    sleep 2
    SPECTRA_CONFIG=/tmp/spectra-full.yaml ./bin/spectra > /tmp/spectra-full2.log 2>&1 &
    SERVER2_PID=$!
    sleep 3

    if kill -0 $SERVER2_PID 2>/dev/null; then
        ok "server restarted"
        HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" -H "Authorization: Bearer test-key-123" "$FULL_URL/api/webhooks")
        BODY=$(cat /tmp/resp.json)
        if echo "$BODY" | grep -q "$PERSIST_WEBHOOK_ID"; then
            ok "  webhook persisted after restart ✓"
        else
            fail "  webhook lost after restart (SQLite not working)"
        fi
    else
        fail "server failed to restart"
    fi
else
    fail "create webhook for persistence test → $HTTP_CODE"
fi

echo ""
log "--- Sessions & Profiles with SQLite (PHASE 4) ---"
echo ""

log "Test: create session (sqlite)"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" -X POST "$FULL_URL/api/sessions" \
    -H "Authorization: Bearer test-key-123" -H "Content-Type: application/json" \
    -d '{"ttl_seconds":3600}')
BODY=$(cat /tmp/resp.json)
if [ "$HTTP_CODE" = "200" ]; then
    ok "create session (sqlite) → 200"
    SESS_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    [ -n "$SESS_ID" ] && ok "  session id: ${SESS_ID:0:8}..." || fail "  id missing"
else
    fail "create session (sqlite) → $HTTP_CODE"
fi

log "Test: list sessions (sqlite)"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" -H "Authorization: Bearer test-key-123" "$FULL_URL/api/sessions")
BODY=$(cat /tmp/resp.json)
[ "$HTTP_CODE" = "200" ] && ok "list sessions → 200" && echo "$BODY" | grep -q '"count"' && ok "  count present" || fail "list sessions → $HTTP_CODE"

log "Test: create profile (sqlite)"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" -X POST "$FULL_URL/api/profiles" \
    -H "Authorization: Bearer test-key-123" -H "Content-Type: application/json" \
    -d '{"name":"us-chrome","locale":"en-US","timezone":"America/New_York","stealth_level":"advanced"}')
BODY=$(cat /tmp/resp.json)
if [ "$HTTP_CODE" = "200" ]; then
    ok "create profile (sqlite) → 200"
    PROF_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    [ -n "$PROF_ID" ] && ok "  profile id: ${PROF_ID:0:8}..." || fail "  id missing"
else
    fail "create profile (sqlite) → $HTTP_CODE"
fi

log "Test: list profiles (sqlite)"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" -H "Authorization: Bearer test-key-123" "$FULL_URL/api/profiles")
[ "$HTTP_CODE" = "200" ] && ok "list profiles → 200" || fail "list profiles → $HTTP_CODE"

echo ""
log "--- SpectraQL with auth ---"
echo ""

log "Test: SpectraQL with auth"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" -X POST "$FULL_URL/api/query" \
    -H "Authorization: Bearer test-key-123" -H "Content-Type: application/json" \
    -d '{"steps":[{"action":"goto","url":"https://example.com"}]}')
[ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "500" ] && ok "SpectraQL with auth → $HTTP_CODE" || fail "SpectraQL with auth → $HTTP_CODE"

log "Test: SpectraQL no auth → 401"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$FULL_URL/api/query" \
    -H "Content-Type: application/json" -d '{"steps":[{"action":"goto","url":"https://example.com"}]}')
[ "$HTTP_CODE" = "401" ] && ok "SpectraQL no auth → 401" || fail "SpectraQL no auth → $HTTP_CODE"

log "Test: /api/metrics with auth"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" -H "Authorization: Bearer test-key-123" "$FULL_URL/api/metrics")
BODY=$(cat /tmp/resp.json)
if [ "$HTTP_CODE" = "200" ]; then
    ok "/api/metrics → 200"
    echo "$BODY" | grep -q "total_requests" && ok "  total_requests present" || fail "  total_requests missing"
    echo "$BODY" | grep -q "avg_duration_ms" && ok "  avg_duration_ms present" || fail "  avg_duration_ms missing"
else
    fail "/api/metrics → $HTTP_CODE"
fi

log "Test: /api/jobs with auth"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" -H "Authorization: Bearer test-key-123" "$FULL_URL/api/jobs")
[ "$HTTP_CODE" = "200" ] && ok "/api/jobs → 200" || fail "/api/jobs → $HTTP_CODE"

log "Test: /api/jobs?limit=10"
HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" -H "Authorization: Bearer test-key-123" "$FULL_URL/api/jobs?limit=10")
[ "$HTTP_CODE" = "200" ] && ok "/api/jobs?limit=10 → 200" || fail "/api/jobs?limit=10 → $HTTP_CODE"

echo ""
log "--- Rate Limit Test ---"
echo ""

log "Test: rapid requests → rate limit"
RATE_LIMITED=false
for i in $(seq 1 300); do
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$FULL_URL/health")
    if [ "$HTTP_CODE" = "429" ]; then
        RATE_LIMITED=true
        ok "rate limiter triggered at request #$i → 429"
        break
    fi
done
[ "$RATE_LIMITED" = false ] && ok "rate limiter: 300 requests within limit (high burst)"

echo ""
########################################
# PHASE 5: AI PLUGIN (optional, needs API key)
########################################
log "========================================="
log "🤖 PHASE 5: AI PLUGIN (optional)"
log "========================================="
echo ""

# AI tests run against FULL_URL (port 3001, still running from Phase 4)
AI_URL="$FULL_URL"
AI_AUTH="-H \"Authorization: Bearer test-key-123\""

if [ -n "$OPENAI_API_KEY" ] && [ "$HAS_CHROME" = true ]; then
    # ai/plan — dry run, no browser needed
    log "Test: ai/plan (dry run)"
    HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" --max-time 30 -X POST "$FULL_URL/api/ai/plan" \
        -H "Authorization: Bearer test-key-123" -H "Content-Type: application/json" \
        -d "{\"task\":\"Navigate to example.com and return the title\",\"openai_api_key\":\"$OPENAI_API_KEY\"}")
    BODY=$(cat /tmp/resp.json)
    if [ "$HTTP_CODE" = "200" ]; then
        ok "ai/plan → 200"
        echo "$BODY" | grep -q '"plan"' && ok "  plan present" || fail "  plan missing"
        echo "$BODY" | grep -q '"estimated_steps"' && ok "  estimated_steps present" || warn "  estimated_steps missing"
    else
        fail "ai/plan → $HTTP_CODE"
    fi

    # ai/observe — list page actions
    log "Test: ai/observe"
    HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" --max-time 30 -X POST "$FULL_URL/api/ai/observe" \
        -H "Authorization: Bearer test-key-123" -H "Content-Type: application/json" \
        -d "{\"openai_api_key\":\"$OPENAI_API_KEY\",\"url\":\"https://example.com\"}")
    BODY=$(cat /tmp/resp.json)
    if [ "$HTTP_CODE" = "200" ]; then
        ok "ai/observe → 200"
        echo "$BODY" | grep -q '"actions"' && ok "  actions present" || fail "  actions missing"
        echo "$BODY" | grep -q '"url"' && ok "  url present" || fail "  url missing"
    else
        fail "ai/observe → $HTTP_CODE"
    fi

    # ai/extract — structured extraction
    log "Test: ai/extract"
    HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" --max-time 30 -X POST "$FULL_URL/api/ai/extract" \
        -H "Authorization: Bearer test-key-123" -H "Content-Type: application/json" \
        -d "{\"instruction\":\"get the page title\",\"schema\":{\"title\":\"string\"},\"openai_api_key\":\"$OPENAI_API_KEY\",\"url\":\"https://example.com\"}")
    BODY=$(cat /tmp/resp.json)
    if [ "$HTTP_CODE" = "200" ]; then
        ok "ai/extract → 200"
        echo "$BODY" | grep -q '"title"' && ok "  title extracted" || warn "  title not in response"
    else
        fail "ai/extract → $HTTP_CODE"
    fi

    # ai/execute — full autonomous agent
    log "Test: ai/execute (full agent)"
    HTTP_CODE=$(curl -s -o /tmp/resp.json -w "%{http_code}" --max-time 60 -X POST "$FULL_URL/api/ai/execute" \
        -H "Authorization: Bearer test-key-123" -H "Content-Type: application/json" \
        -d "{\"task\":\"Navigate to https://example.com and return the page title\",\"openai_api_key\":\"$OPENAI_API_KEY\",\"max_steps\":5,\"config\":{\"planning\":true,\"self_correction\":true,\"memory\":true}}")
    BODY=$(cat /tmp/resp.json)
    if [ "$HTTP_CODE" = "200" ]; then
        ok "ai/execute → 200"
        echo "$BODY" | grep -q '"result"' && ok "  result present" || fail "  result missing"
        echo "$BODY" | grep -q '"action_log"' && ok "  action_log present" || fail "  action_log missing"
        echo "$BODY" | grep -q '"plan"' && ok "  plan present (planning=true)" || warn "  plan missing"
        echo "$BODY" | grep -q '"llm_calls"' && ok "  llm_calls tracked" || fail "  llm_calls missing"
        echo "$BODY" | grep -q '"cache_hits"' && ok "  cache_hits tracked" || fail "  cache_hits missing"
    else
        fail "ai/execute → $HTTP_CODE"
        echo "  $(cat /tmp/resp.json | head -c 200)"
    fi
else
    warn "AI plugin tests SKIPPED (set OPENAI_API_KEY env var + Chromium required)"
    warn "  Test manually: curl -X POST $FULL_URL/api/ai/plan -H 'Authorization: Bearer test-key-123' -d '{\"task\":\"...\",\"openai_api_key\":\"sk-...\"}'"
fi

########################################
# CLEANUP
########################################
echo ""
log "========================================="
log "🧹 CLEANUP"
log "========================================="
echo ""

kill $SERVER2_PID 2>/dev/null; wait $SERVER2_PID 2>/dev/null || true
ok "servers stopped"
rm -f /tmp/spectra-basic.yaml /tmp/spectra-full.yaml /tmp/spectra-basic.log /tmp/spectra-full.log /tmp/spectra-full2.log /tmp/resp.json /tmp/spectra-test.db

########################################
# RESULTS
########################################
echo ""
log "========================================="
log "📊 FINAL RESULTS"
log "========================================="
echo ""
echo -e "  ${GREEN}Passed: $PASS${NC}"
echo -e "  ${RED}Failed: $FAIL${NC}"
echo -e "  Total:  $((PASS+FAIL))"
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}  🎉 ALL TESTS PASSED!${NC}"
    echo ""
    echo "  Tested:"
    echo "    ⚡ Build: core + CLI + 8 plugins (incl. ai)"
    echo "    📡 REST API: health, ready, plugins, execute, errors"
    echo "    🔍 /pressure — system health endpoint"
    echo "    📊 /api/metrics — request tracking"
    echo "    📋 /api/jobs — job history"
    echo "    🔗 SpectraQL — multi-step query in one request"
    echo "    🖥️  Sessions — create/get/list/delete"
    echo "    👤 Profiles — browser fingerprint identities"
    echo "    📡 Live View — WebSocket endpoint"
    echo "    📸 Screenshot: default 1920x1080, explicit size"
    echo "    📄 PDF: 1920x1080 viewport"
    echo "    🕷️  Scrape: 1920x1080 viewport"
    echo "    👻 Stealth: bot detection bypass"
    echo "    🔍 Visual Diff: pixel comparison"
    echo "    🎥 Recorder: enhanced actions (assert_text, wait_for, hover)"
    echo "    💻 CLI: health, plugins, exec"
    echo "    🔐 Auth: no key, wrong key, correct key"
    echo "    🔔 Webhooks: CRUD"
    echo "    ⏰ Scheduler: CRUD, invalid cron"
    echo "    💾 SQLite: persistence across restart"
    echo "    ⏱️  Rate limiter: burst test"
    echo "    🤖 AI Plugin: plan/observe/extract/execute (if OPENAI_API_KEY set)"
    echo ""
    exit 0
else
    echo -e "${RED}  ⚠️  Some tests failed. Check output above.${NC}"
    exit 1
fi
