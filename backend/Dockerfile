FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Copy go mod files first (cache layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Cross-compile for the target platform (no CGO, no QEMU needed)
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /server ./cmd/server

# Runtime image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /server .
COPY --from=builder /app/migrations ./migrations

# Create certs cache directory for Let's Encrypt autocert
RUN mkdir -p /app/certs

EXPOSE 8080 443 80

CMD ["./server"]
