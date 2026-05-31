FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o portfolio-web ./cmd/portfolio-web

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=builder /build/portfolio-web .
COPY --from=builder /build/internal/web/static ./internal/web/static

USER app

EXPOSE 8000

CMD ["./portfolio-web"]
