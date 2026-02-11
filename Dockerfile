FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=1 go build -o wordle-six .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/wordle-six .

# Copy static frontend files
COPY index.html manifest.json ./static/
COPY *.js ./static/
COPY icon.svg icon-192.png icon-512.png og-preview.png ./static/

VOLUME /data
ENV PORT=8080
ENV DB_PATH=/data/wordle-six.db

EXPOSE 8080
CMD ["./wordle-six"]
