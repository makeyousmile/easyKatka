FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/app ./cmd/easykatka

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /out/app /app/app
COPY account_id /app/account_id
ENV TELEGRAM_BOT_TOKEN=""
ENV TELEGRAM_NOTIFY_CHAT_ID=""
CMD ["/app/app"]
