#!/usr/bin/env bash
# Dev-time parity harness: render the same routes from the Python app (oracle)
# and the Go app against identical database copies and diff the normalized HTML.
# Usage: bash scripts/parity_check.sh
set -u

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

SRC_DB="${PARITY_SRC_DB:-.data/portfolio.db}"
PY_DB="/tmp/pm_py.db"
GO_DB="/tmp/pm_go.db"
PY_PORT=8001
GO_PORT=8079
OUT="/tmp/parity_out"
mkdir -p "$OUT"

cp "$SRC_DB" "$PY_DB"
cp "$SRC_DB" "$GO_DB"

cleanup() {
  pkill -f "uvicorn portfolio_manager" 2>/dev/null
  [ -n "${GO_PID:-}" ] && kill -TERM "$GO_PID" 2>/dev/null
}
trap cleanup EXIT

# --- Python oracle ---
PORTFOLIO_DB_PATH="$PY_DB" uv run uvicorn portfolio_manager.web.app:app \
  --host 127.0.0.1 --port "$PY_PORT" --no-access-log >/tmp/parity_py.log 2>&1 &
for _ in $(seq 1 30); do
  curl -sf -o /dev/null "http://127.0.0.1:$PY_PORT/groups" && break
  sleep 0.5
done

GID="$(sqlite3 "$PY_DB" 'SELECT id FROM groups LIMIT 1;')"
echo "group id: $GID"

curl -s -o "$OUT/py_list.html"  "http://127.0.0.1:$PY_PORT/groups"
curl -s -H 'HX-Request: true' -o "$OUT/py_edit.html" "http://127.0.0.1:$PY_PORT/groups/$GID/edit"
curl -s -o "$OUT/py_row.html"   "http://127.0.0.1:$PY_PORT/groups/$GID"
pkill -f "uvicorn portfolio_manager" 2>/dev/null
sleep 1

# --- Go ---
PORTFOLIO_ADDR="127.0.0.1:$GO_PORT" PORTFOLIO_DB_PATH="$GO_DB" go run ./cmd/portfolio-web >/tmp/parity_go.log 2>&1 &
GO_PID=$!
for _ in $(seq 1 30); do
  curl -sf -o /dev/null "http://127.0.0.1:$GO_PORT/groups" && break
  sleep 0.5
done

curl -s -o "$OUT/go_list.html"  "http://127.0.0.1:$GO_PORT/groups"
curl -s -H 'HX-Request: true' -o "$OUT/go_edit.html" "http://127.0.0.1:$GO_PORT/groups/$GID/edit"
curl -s -o "$OUT/go_row.html"   "http://127.0.0.1:$GO_PORT/groups/$GID"
kill -TERM "$GO_PID" 2>/dev/null
GO_PID=""

# --- compare ---
python3 - "$OUT" <<'PY'
import re, sys
out = sys.argv[1]
def norm(p):
    s = open(p, encoding="utf-8").read()
    # Whitespace adjacent to tag boundaries is cosmetic (Jinja keeps source
    # indentation; templ trims it). Strip it so the comparison asserts
    # structural + attribute + text-content parity, not byte-for-byte layout.
    s = re.sub(r">\s+", ">", s)
    s = re.sub(r"\s+<", "<", s)
    s = re.sub(r"\s+", " ", s)
    s = re.sub(r"\s*=\s*", "=", s)
    s = s.replace(" >", ">").replace("/>", ">")
    # The HTML5 doctype is case-insensitive (Jinja emits DOCTYPE, templ doctype).
    s = re.sub(r"(?i)<!doctype html>", "<!doctype html>", s)
    # Entity equivalence: inside a double-quoted attribute, a literal ' (Jinja)
    # and &#39; (templ auto-escaping) parse to the same DOM string value.
    s = s.replace("&#39;", "'").replace("&#34;", '"').replace("&amp;", "&")
    return s.strip()
rc = 0
for name in ("list", "edit", "row"):
    a, b = norm(f"{out}/py_{name}.html"), norm(f"{out}/go_{name}.html")
    if a == b:
        print(f"{name}: MATCH ({len(a)} bytes)")
    else:
        rc = 1
        i = 0
        while i < min(len(a), len(b)) and a[i] == b[i]:
            i += 1
        print(f"{name}: DIFFER at {i} (py={len(a)} go={len(b)})")
        print(f"  PY: {a[max(0,i-60):i+100]!r}")
        print(f"  GO: {b[max(0,i-60):i+100]!r}")
sys.exit(rc)
PY
