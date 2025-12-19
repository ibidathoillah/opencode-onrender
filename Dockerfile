FROM debian:bookworm-slim

RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates curl bash git libsqlite3-0 caddy \
  && rm -rf /var/lib/apt/lists/*

ENV OPENCODE_INSTALL_DIR=/usr/local/bin

RUN curl -fsSL https://opencode.ai/install | bash

WORKDIR /app
COPY opencode.json /app/opencode.json
ENV OPENCODE_CONFIG=/app/opencode.json

EXPOSE 4096

CMD ["bash","-lc","PORT=\"${PORT:-4096}\"; cat > /etc/caddy/Caddyfile <<EOF\n{\n  admin off\n}\n\n:${PORT} {\n  @health {\n    path /healthz\n  }\n\n  handle @health {\n    respond \"ok\" 200\n  }\n\n  @authorized {\n    header Authorization \"Bearer {env.OPENCODE_API_TOKEN}\"\n  }\n\n  handle @authorized {\n    reverse_proxy 127.0.0.1:4097\n  }\n\n  handle {\n    respond \"Unauthorized\" 401\n  }\n}\nEOF\nopencode serve --hostname 127.0.0.1 --port 4097 & exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile"]
