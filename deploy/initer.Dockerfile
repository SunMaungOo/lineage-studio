FROM golang:alpine3.24 AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY backend/lineage-studio/go.mod .

RUN go mod download

COPY backend/lineage-studio/. .

RUN CGO_ENABLED=0 GOOS=linux go build -o /build/initer ./cmd/initer

FROM alpine:3.23.5 AS runner

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

RUN addgroup -S appgroup && adduser -S appuser -G appgroup -u 10001

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /build/initer .

ENTRYPOINT ["./initer"]

CMD ["web"]