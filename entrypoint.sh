#!/usr/bin/env bash
set -e

PORT="${PORT:-4096}"

cat > /etc/caddy/Caddyfile <<EOF
{
  admin off
}

:${PORT} {

  handle /healthz {
    respond "ok" 200
  }

  # Per-session SSE
  handle_path /sse/* {
    root * /app
    cgi /sse/* /app/sse-filter.sh
  }

  @authorized {
    header Authorization "Bearer {env.OPENCODE_API_TOKEN}"
  }

  handle @authorized {
    reverse_proxy 127.0.0.1:4097
  }

  handle {
    respond "Unauthorized" 401
  }
}
EOF

echo "[INFO] Starting OpenCode..."
opencode serve --hostname 127.0.0.1 --port 4097 &

echo "[INFO] Starting Caddy..."
exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
