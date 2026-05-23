FROM oven/bun:1-alpine AS builder
WORKDIR /app
COPY package.json bun.lock* ./
RUN bun install --frozen-lockfile
COPY tsconfig.json ./
COPY src ./src
COPY scripts ./scripts
RUN bun build \
    --compile \
    --target=bun-linux-x64-musl \
    --define CK_VERSION='"docker"' \
    --define CK_COMMIT='"docker"' \
    --define process.env.DEV='"false"' \
    --minify \
    --sourcemap=none \
    --outfile=ck-cli \
    ./src/main.tsx

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/ck-cli /usr/local/bin/ck-cli
ENTRYPOINT ["/usr/local/bin/ck-cli"]
