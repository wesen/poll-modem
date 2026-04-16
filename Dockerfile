FROM golang:1.25-bookworm AS builder

RUN apt-get update && apt-get install -y gcc libc6-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o /poll-modem ./cmd/poll-modem

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=builder /poll-modem /usr/local/bin/poll-modem

RUN mkdir -p /data
VOLUME /data

ENTRYPOINT ["poll-modem"]
CMD ["collect"]
