#!/usr/bin/env bash
# End-to-end demo for the appsmock build-tag.
#
# Verifies that with -tags appsmock and LARK_CLI_APPS_MOCK set:
#   1) `lark-cli apps +list` is routed to the local mock server
#      (mock log has a matching entry, response contains canned data)
#   2) a non-apps command (`lark-cli contact +get-user`) is NOT routed
#      to the mock (mock log stays empty)
#
# Run from the lark-cli repo root:
#   bash demo/e2e.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

BIN_DIR="$(mktemp -d -t appsmock-e2e.XXXXXX)"
trap 'rm -rf "$BIN_DIR"; [[ -n "${MOCK_PID:-}" ]] && kill "$MOCK_PID" 2>/dev/null || true' EXIT

PORT="${APPSMOCK_PORT:-7878}"
MOCK_URL="http://127.0.0.1:${PORT}"
MOCK_LOG="$BIN_DIR/mockserver.log"

echo "[1/5] building lark-cli with -tags appsmock"
go build -tags appsmock -o "$BIN_DIR/lark-cli-appsmock" .

echo "[2/5] building mock server"
go build -o "$BIN_DIR/mockserver" ./demo/cmd/mockserver

echo "[3/5] starting mock server on $MOCK_URL"
"$BIN_DIR/mockserver" -addr "127.0.0.1:${PORT}" -log "$MOCK_LOG" >/dev/null 2>"$BIN_DIR/mock.stderr" &
MOCK_PID=$!
# Wait for the server to actually listen before issuing requests.
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if curl -sf "$MOCK_URL/healthz" >/dev/null 2>&1; then break; fi
  sleep 0.1
done

echo "[4/5] running positive case: apps +list"
out="$(LARK_CLI_APPS_MOCK="$MOCK_URL" "$BIN_DIR/lark-cli-appsmock" apps +list --format json 2>&1 | tee "$BIN_DIR/positive.out")"
if ! grep -q "mock_app_aaa" <<<"$out"; then
  echo "FAIL: positive case did not see canned mock data" >&2
  echo "$out" >&2
  exit 1
fi
if ! grep -q '"path":"/open-apis/spark/v1/apps"' "$MOCK_LOG"; then
  echo "FAIL: mock log missing /open-apis/spark/v1/apps entry" >&2
  cat "$MOCK_LOG" >&2 || true
  exit 1
fi
if ! grep -q '"appsmock_orig":"open.feishu.cn"' "$MOCK_LOG"; then
  echo "FAIL: interceptor origin header not seen (URL may not have been rewritten)" >&2
  exit 1
fi
echo "  positive case OK"

echo "[5/5] running negative case: contact +get-user must NOT hit mock"
# Truncate mock log so we can assert "no new lines".
: >"$MOCK_LOG"
LARK_CLI_APPS_MOCK="$MOCK_URL" "$BIN_DIR/lark-cli-appsmock" contact +get-user --user-id u_nonexistent --user-id-type open_id 2>"$BIN_DIR/negative.err" >/dev/null || true
# A line containing `/open-apis/contact` in the mock log would indicate the
# interceptor over-reached. Empty log = correct pass-through (the real call
# either hit open.feishu.cn or failed earlier in the credential pipeline —
# both are acceptable for this assertion).
if grep -q '/open-apis/' "$MOCK_LOG"; then
  echo "FAIL: non-apps request was routed to mock:" >&2
  cat "$MOCK_LOG" >&2
  exit 1
fi
echo "  negative case OK (mock log empty)"

echo
echo "ALL E2E CHECKS PASSED"
