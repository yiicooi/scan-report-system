FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H appuser

COPY --from=builder /out/server ./server
COPY config ./config

EXPOSE 8080

USER appuser

CMD ["./server"]
