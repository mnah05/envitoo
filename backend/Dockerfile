# syntax=docker/dockerfile:1.7   # enables modern features like cache mounts

# ---- Stage 1: Build the Go binary ----
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache go.mod + go.sum separately so Docker reuses the layer
# (avoids re-downloading modules on every source change)
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the full source and compile
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o /server ./cmd/server
# Flags explained:
#   CGO_ENABLED=0       — fully static binary (no libc dependency)
#   -trimpath           — strips absolute file paths from the binary
#   -ldflags="-w -s"    — strips DWARF debug info + symbol table (smaller binary)

# ---- Stage 2: Minimal runtime image ----
FROM alpine:3.21

# ca-certificates  — required for HTTPS calls (Resend, Clerk, Supabase)
# tzdata           — timezone support for logs / DB timestamps
# adduser          — run as non-root for security
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

# Copy only the compiled binary from the builder stage
COPY --from=builder /server /server

USER app

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz || exit 1

CMD ["/server"]
