# =============================
# Build TinyGo binary
# =============================
FROM golang:1.22-alpine AS Build

WORKDIR /src
COPY main.go .

RUN tinygo build \
  -o proxy \
  -target=linux-amd64 \
  -no-debug \
  main.go

# =============================
# Runtime
# =============================
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
  ca-certificates curl bash libsqlite3-0 \
  && rm -rf /var/lib/apt/lists/*

# Install OpenCode
ENV OPENCODE_INSTALL_DIR=/usr/local/bin
RUN curl -fsSL https://opencode.ai/install | bash

# Copy proxy
COPY --from=build /src/proxy /usr/local/bin/proxy

WORKDIR /app
COPY opencode.json /app/opencode.json
ENV OPENCODE_CONFIG=/app/opencode.json

EXPOSE 4096

CMD bash -lc "\
  opencode serve --hostname 127.0.0.1 --port 4097 & \
  exec proxy \
  "
