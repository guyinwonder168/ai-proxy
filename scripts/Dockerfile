# Stage 1: Build the binary
FROM golang:1.24-alpine AS builder

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Set the working directory within the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the binary statically
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/ai-proxy main.go

# Stage 2: Use a smaller base image
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S ai-proxy &&\
    adduser -u 1001 -S ai-proxy -G ai-proxy

# Create config directory
RUN mkdir -p /config && chown ai-proxy:ai-proxy /config

# Copy the binary from the builder stage
COPY --from=builder /app/ai-proxy /app/ai-proxy

# Copy the entrypoint script
COPY --from=builder /app/scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

# Set proper permissions for the entrypoint script
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# Switch to non-root user
USER ai-proxy

# Set working directory
WORKDIR /app

# Expose default port
EXPOSE 80

# Define volumes for configuration files
VOLUME ["/config"]

# Set default port environment variable
ENV PORT="80"

# Set labels and metadata
LABEL name="ai-proxy" \
      version="1.0" \
      description="AI Proxy Server for multiple LLM providers" \
      maintainer="AI Proxy Team"

# Set the entrypoint
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]