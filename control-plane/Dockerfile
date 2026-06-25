FROM golang:1.23-alpine AS builder

# Add ca-certificates and timezone data
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 1000 appuser

WORKDIR /app

# Cache dependencies first
COPY control-plane/go.mod control-plane/go.sum ./
RUN go mod download

# Build with reproducible, minimal footprint
COPY control-plane/ ./
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -o /control-plane ./cmd/control-plane

# Minimal runtime image
FROM scratch

# Copy certificates, user info, and timezone data from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy compiled binary
COPY --from=builder /control-plane /app/control-plane

USER appuser
EXPOSE 8080

ENTRYPOINT ["/app/control-plane"]
