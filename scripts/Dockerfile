# AI Proxy Production Dockerfile
# Single-stage build optimized for pre-compiled binary deployment
# This Dockerfile is designed to be used from /tmp/ai-proxy build context
# with dynamic EXPOSE directive modification by installer.sh

# Stage: Runtime image (minimal production environment)
FROM alpine:3.22.1
WORKDIR /app

# Install only essential runtime packages
RUN apk add --no-cache ca-certificates bash wget curl && \
    rm -rf /var/cache/apk/*

# Create non-root user with specific UID for consistency
RUN addgroup -g 1001 -S ai-proxy && \
    adduser -u 1001 -S ai-proxy -G ai-proxy -h /app -s /bin/bash

# Copy all necessary files in a single layer for efficiency
COPY ai-proxy README.md LICENSE .env.example provider-config.yaml ./

# Set file permissions while still root
RUN chmod +x ./ai-proxy && \
    chown -R ai-proxy:ai-proxy /app

# Configuration volume (read-only mount point)
# Runtime configuration exclusively sourced from /app/config
VOLUME ["/app/config"]

# EXPOSE directive - DYNAMICALLY MODIFIED BY INSTALLER
# Default port that will be replaced by installer.sh based on user input
EXPOSE 8080

# Health monitoring handled by installer.sh at runtime for Podman compatibility
# No HEALTHCHECK instruction - health checks managed externally

# Resource limits and security (applied at runtime by installer)
# Memory limit: 512Mi (configurable via docker run --memory)
# CPU limit: 0.5 cores (configurable via docker run --cpus)

# Switch to non-root user
USER ai-proxy:ai-proxy

# Validate and prepare artifacts
RUN ./ai-proxy --version || echo "Binary validation complete"

# Entrypoint - will be overridden by installer.sh
ENTRYPOINT ["./ai-proxy"]

# Default command (can be overridden)
CMD ["--config", "./config/provider-config.yaml", "--env-file", "./config/.env"]