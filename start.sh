#!/bin/sh
set -eu

./jask &
backend_pid=$!

caddy run --config /etc/caddy/Caddyfile --adapter caddyfile &
caddy_pid=$!

stop_services() {
	trap - EXIT INT TERM
	kill "$backend_pid" "$caddy_pid" 2>/dev/null || true
	wait "$backend_pid" "$caddy_pid" 2>/dev/null || true
}

trap stop_services EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

while kill -0 "$backend_pid" 2>/dev/null && kill -0 "$caddy_pid" 2>/dev/null; do
	sleep 1
done

status=0
if ! kill -0 "$backend_pid" 2>/dev/null; then
	wait "$backend_pid" || status=$?
else
	wait "$caddy_pid" || status=$?
fi
exit "$status"
