FROM debian:bookworm-slim

# =============================
# Dependencies
# =============================
RUN apt-get update \
  && apt-get install -y --no-install-recommends \
  ca-certificates \
  curl \
  bash \
  jq \
  libsqlite3-0 \
  caddy \
  && rm -rf /var/lib/apt/lists/*

# =============================
# Install OpenCode
# =============================
ENV OPENCODE_INSTALL_DIR=/usr/local/bin
RUN curl -fsSL https://opencode.ai/install | bash

# =============================
# App config
# =============================
WORKDIR /app
COPY opencode.json /app/opencode.json
ENV OPENCODE_CONFIG=/app/opencode.json

# =============================
# SSE filter (bash)
# =============================
RUN cat > /app/sse-filter.sh <<'EOF'
#!/usr/bin/env bash

SESSION_ID="$1"
OPENCODE_URL="http://127.0.0.1:4097/global/event"

echo "Content-Type: text/event-stream"
echo "Cache-Control: no-cache"
echo "Connection: keep-alive"
echo

curl -sN \
  -H "Authorization: Bearer ${OPENCODE_API_TOKEN}" \
  "$OPENCODE_URL" \
| sed -n 's/^data: //p' \
| jq --unbuffered -c '
    select(
      .type == "chat.message"
      and .sessionId?
      and .sessionId == env.SESSION_ID
    )
  ' \
| while read -r json; do
    echo "data: $json"
    echo
  done
EOF

RUN chmod +x /app/sse-filter.sh

# =============================
# Entrypoint
# =============================
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 4096
CMD ["/entrypoint.sh"]
