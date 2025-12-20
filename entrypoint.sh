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

  # BLOCK GLOBAL EVENT (PUBLIC)
  handle /global/event {
    respond "Not Found" 404
  }

  # Per-session SSE (CGI)
  handle_path /sse/* {
    cgi {
      script /app/sse-filter.sh
    }
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

opencode serve --hostname 127.0.0.1 --port 4097 &
exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
