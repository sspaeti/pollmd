# Multi-stage build for Railway. Build context is the repo root; this
# Dockerfile is pointed to from deploy/railway/railway.json (or set by hand
# in the Railway service settings).
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

COPY cmd/    ./cmd/
COPY internal/ ./internal/

RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /out/survey ./cmd/survey

# --- Runtime ---
FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN useradd --system --uid 1001 --home-dir /var/db/survey --create-home survey \
    && mkdir -p /var/db/survey \
    && chown -R survey:survey /var/db/survey

COPY --from=build /out/survey /usr/local/bin/survey

USER survey
WORKDIR /var/db/survey

# Cosmetic; Railway routes via $PORT and TCP Proxy mappings, not EXPOSE.
# Only EXPOSE 8080 to discourage platforms from choosing 9494 as $PORT.
EXPOSE 8080

# Bind HTTP to Railway's assigned $PORT when present. Falls back to
# SURVEY_HTTP_ADDR (or the app default) outside Railway/local.
ENTRYPOINT ["/bin/sh", "-c", "\
  # Bind HTTP to Railway-assigned $PORT when present\n\
  if [ -n \"$PORT\" ]; then export SURVEY_HTTP_ADDR=\"0.0.0.0:$PORT\"; fi;\n\
  # If HTTP and Quack would collide (rarely when PORT=9494), move Quack to 9495.\n\
  qp=\"${SURVEY_QUACK_ADDR##*:}\";\n\
  if [ -z \"$qp\" ] || [ \"$SURVEY_QUACK_ADDR\" = \"\" ]; then qp=9494; fi;\n\
  if [ -n \"$PORT\" ] && [ \"$PORT\" = \"$qp\" ]; then export SURVEY_QUACK_ADDR=\"0.0.0.0:9495\"; fi;\n\
  exec /usr/local/bin/survey\n\
"]
