# Multi-stage build
FROM golang:1.25.1-alpine AS builder

# Install ca-certificates for HTTPS requests
RUN apk update && apk add --no-cache ca-certificates git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o webhook cmd/webhook/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates
RUN apk --no-cache add ca-certificates

# Create a non-root user
RUN addgroup -g 65532 webhook &&\
    adduser -D -u 65532 -G webhook webhook

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/webhook .

# Change ownership to non-root user
RUN chown -R webhook:webhook /root/
USER webhook

# Expose port
EXPOSE 8443

# Run the binary
ENTRYPOINT ["./webhook"]