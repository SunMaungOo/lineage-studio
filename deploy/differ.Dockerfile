FROM golang:alpine3.24 AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY backend/lineage-studio/go.mod .

RUN go mod download

COPY backend/lineage-studio/. .

RUN CGO_ENABLED=0 GOOS=linux go build -o /build/differ ./cmd/differ

FROM alpine:3.23.5 AS runner

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

RUN apk add --no-cache git

RUN addgroup -S appgroup && adduser -S appuser -G appgroup -u 10001

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /build/differ .

ENTRYPOINT ["./differ"]