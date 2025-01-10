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
WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache git

# Copy binary and static assets
COPY --from=builder /app/simplegit .
COPY static/ static/
COPY templates/ templates/
COPY config.json .

# Create repositories directory
RUN mkdir -p repositories

# Run as non-root user
RUN adduser -D -h /app gituser && \
    chown -R gituser:gituser /app
USER gituser

EXPOSE 3000
CMD ["./simplegit"]