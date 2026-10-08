#!/usr/bin/env bash
# Run from any directory. No database is discovered from application settings.
# Either provide LOGSERVICE_TEST_POSTGRES for a disposable database, or provide
# LOGSERVICE_PG_BIN to create a temporary loopback-only PostgreSQL cluster.
# Optional: LOGSERVICE_PG_SHARE (initdb -L), LOGSERVICE_PG_PORT (default 55433).
set -euo pipefail
cd "$(dirname "$0")/.."

if [[ -n "${LOGSERVICE_TEST_POSTGRES:-}" ]]; then
  export DBSOURCE_TEST_POSTGRES="${DBSOURCE_TEST_POSTGRES:-$LOGSERVICE_TEST_POSTGRES}"
  exec go test -race "$@" ./logservice ./dbsource/...
fi

: "${LOGSERVICE_PG_BIN:?Set LOGSERVICE_PG_BIN to PostgreSQL bin directory, or LOGSERVICE_TEST_POSTGRES to a disposable database URL}"
port="${LOGSERVICE_PG_PORT:-55433}"
[[ "$port" =~ ^[0-9]+$ ]] || { echo 'LOGSERVICE_PG_PORT must be numeric' >&2; exit 2; }
fixture="$(mktemp -d "${TMPDIR:-/tmp}/commons-logservice-pg.XXXXXXXX")"
cleanup() {
  status=$?
  "$LOGSERVICE_PG_BIN/pg_ctl" -D "$fixture/data" -m immediate -w stop >/dev/null 2>&1 || true
  if (( status != 0 )); then
    echo 'PostgreSQL fixture log:' >&2
    tail -n 60 "$fixture/server.log" >&2 2>/dev/null || true
  fi
  rm -rf "$fixture"
  exit "$status"
}
trap cleanup EXIT
share=()
if [[ -n "${LOGSERVICE_PG_SHARE:-}" ]]; then share=(-L "$LOGSERVICE_PG_SHARE"); fi
"$LOGSERVICE_PG_BIN/initdb" -D "$fixture/data" -U logservice_test --auth=trust --no-locale --encoding=UTF8 "${share[@]}" >"$fixture/initdb.log"
"$LOGSERVICE_PG_BIN/pg_ctl" -D "$fixture/data" -l "$fixture/server.log" -o "-p $port -h 127.0.0.1 -F -c unix_socket_directories=''" -w start
export LOGSERVICE_TEST_POSTGRES="postgres://logservice_test@127.0.0.1:$port/postgres?sslmode=disable"
export DBSOURCE_TEST_POSTGRES="$LOGSERVICE_TEST_POSTGRES"
"$LOGSERVICE_PG_BIN/psql" "$LOGSERVICE_TEST_POSTGRES" -c 'SELECT version();'
go test -race "$@" ./logservice ./dbsource/...
