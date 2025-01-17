# Build stage
FROM golang:1.21.5-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY . .

# Enable CGO and build
ENV CGO_ENABLED=1
RUN go build -o simplegit

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache git sqlite

# Create non-root user
RUN adduser -D -h /app gituser

WORKDIR /app

# Copy binary and static assets
COPY --from=builder /app/simplegit .
COPY static/ static/
COPY templates/ templates/

# Create necessary directories with correct permissions
RUN mkdir -p /app/repositories /app/data /app/ssh && \
    chown -R gituser:gituser /app && \
    chmod 755 /app/repositories /app/data /app/ssh

# Switch to non-root user
USER gituser

# Set up volumes for persistent data
VOLUME ["/app/repositories", "/app/data", "/app/ssh"]

EXPOSE 3000 2222

CMD ["./simplegit"]
