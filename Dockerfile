FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/a-h/templ/cmd/templ@v0.3.1020

COPY . .
RUN templ generate
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o portfolio-web ./cmd/portfolio-web

FROM alpine:3.21

ENV PORTFOLIO_ADDR=0.0.0.0:8000 \
    PORTFOLIO_DB_PATH=/app/.data/portfolio.db

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S -G app app

WORKDIR /app
RUN mkdir -p .data && chown -R app:app /app

COPY --from=builder /build/portfolio-web .
COPY --from=builder /build/internal/web/static ./internal/web/static

USER app

EXPOSE 8000

CMD ["./portfolio-web"]
