# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# docs/ is committed; no swag generation needed in build (keeps image small).
# CI verifies docs/ is not stale (`swag init` + `git diff --exit-code docs/`).
# If docs missing, uncomment next line and add swag install:
# RUN go install github.com/swaggo/swag/cmd/swag@latest && swag init -g cmd/server/main.go -o docs
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/server ./cmd/server/main.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -g '' appuser
WORKDIR /app
COPY --from=builder /app/server /app/server
EXPOSE 3278
USER appuser
CMD ["/app/server"]
