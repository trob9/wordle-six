FROM golang:1.27.4-alpine3.24 AS builder

# git is needed by `go mod download` to fetch the private module github.com/trob9/pkg.
RUN apk add --no-cache gcc musl-dev git

ENV GOPRIVATE=github.com/trob9/* GONOSUMDB=github.com/trob9/*

WORKDIR /app
COPY go.mod go.sum ./
# GITHUB_TOKEN is a build arg supplied at build time (never committed). It is used
# only in this builder stage, through git env vars on a single RUN, so it is never
# written to a file or copied into the final image.
ARG GITHUB_TOKEN
RUN test -n "$GITHUB_TOKEN" || { echo "GITHUB_TOKEN build arg is required (private module trob9/pkg)"; exit 1; }; \
    GIT_CONFIG_COUNT=1 \
    GIT_CONFIG_KEY_0="url.https://x-access-token:${GITHUB_TOKEN}@github.com/.insteadOf" \
    GIT_CONFIG_VALUE_0="https://github.com/" \
    go mod download

COPY *.go ./
COPY internal/ ./internal/
RUN CGO_ENABLED=1 go build -o wordle-six .

FROM alpine:3.24

RUN apk add --no-cache ca-certificates wget su-exec && \
    addgroup -S appuser && adduser -S appuser -G appuser

WORKDIR /app
COPY --from=builder /app/wordle-six .

COPY static/ ./static/
COPY entrypoint.sh .

RUN chmod +x entrypoint.sh && mkdir -p /data && chown -R appuser:appuser /app /data

VOLUME /data
ENV PORT=8080
ENV DB_PATH=/data/wordle-six.db

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/ || exit 1

ENTRYPOINT ["./entrypoint.sh"]
