# ---- Build stage ----
FROM golang:1.25-alpine AS builder

# gcc and musl-dev are required for mattn/go-sqlite3 (CGO)
RUN apk add --no-cache gcc musl-dev

WORKDIR /src

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with CGO enabled for mattn/go-sqlite3.
# Static linking so the binary works on a minimal runtime image.
RUN CGO_ENABLED=1 go build \
    -ldflags="-s -w -linkmode external -extldflags '-static'" \
    -o /dutyround ./cmd/server

# ---- Runtime stage ----
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app && adduser -S app -G app

WORKDIR /app

# Copy the compiled binary
COPY --from=builder /dutyround .

# Copy runtime assets
COPY --from=builder /src/static ./static
COPY --from=builder /src/internal/templates ./internal/templates
COPY --from=builder /src/migrations ./migrations

# Writable directory for the SQLite database
RUN mkdir -p /data && chown app:app /data
ENV DB_PATH=/data/dutyround.db

USER app

EXPOSE 8080

ENTRYPOINT ["./dutyround"]
