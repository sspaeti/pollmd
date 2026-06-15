# Multi-stage build for Railway. Build context is the repo root; this
# Dockerfile is pointed to from railway.json's dockerfilePath.
#
# Runtime is debian:bookworm-slim (glibc) because duckdb-go-bindings/v2
# embeds a prebuilt libduckdb.so for linux-amd64 / linux-arm64 and extracts
# it to a temp dir at startup. Alpine (musl) would mismatch the libc and
# fail to dlopen.

# --- Build ---
FROM golang:1.24-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /out/survey ./cmd/survey

# --- Runtime ---
FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /var/db/survey

COPY --from=build /out/survey /usr/local/bin/survey

# Image-level defaults. Both listeners bind to 0.0.0.0 so Railway's proxy
# (HTTP edge + TCP Proxy) can reach them. We deliberately do NOT use $PORT
# magic — Railway sets $PORT for internal convention only, and inferring
# from it conflicts with multi-port apps like this one (HTTP + Quack).
#
# Override via Railway env vars only if you genuinely need different ports.
# Then update the matching Public Networking target port + TCP Proxy target
# port in the dashboard.
ENV SURVEY_HTTP_ADDR=0.0.0.0:8080 \
    SURVEY_QUACK_ADDR=0.0.0.0:9494 \
    SURVEY_DB_PATH=/var/db/survey/votes.duckdb

WORKDIR /var/db/survey

EXPOSE 8080 9494

ENTRYPOINT ["/usr/local/bin/survey"]
