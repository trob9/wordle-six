FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=1 go build -o wordle-six .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates wget su-exec && \
    addgroup -S appuser && adduser -S appuser -G appuser

WORKDIR /app
COPY --from=builder /app/wordle-six .

# Copy static frontend files
COPY index.html terms.html privacy.html manifest.json ./static/
COPY *.js ./static/
COPY icon.svg icon-192.png icon-512.png og-preview.png ./static/
COPY entrypoint.sh .

RUN chmod +x entrypoint.sh && mkdir -p /data && chown -R appuser:appuser /app /data

VOLUME /data
ENV PORT=8080
ENV DB_PATH=/data/wordle-six.db

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/ || exit 1

ENTRYPOINT ["./entrypoint.sh"]
