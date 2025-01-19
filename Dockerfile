# Build TS service
FROM node:20-alpine AS ts-builder

WORKDIR /app/ts-service
COPY services/ts-worker/package*.json ./
RUN npm install

COPY services/ts-worker/tsconfig.json .
COPY services/ts-worker/src/ src/
RUN npm run build

# Build Go service
FROM golang:1.21.5-alpine AS go-builder

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
RUN apk add --no-cache git sqlite nodejs

# Create non-root user
RUN adduser -D -h /app gituser

WORKDIR /app

# Copy binaries and assets from build stages
COPY --from=go-builder /app/simplegit .
COPY --from=ts-builder /app/ts-service/dist ./ts-service/dist
COPY --from=ts-builder /app/ts-service/node_modules ./ts-service/node_modules
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

# Add startup script
COPY --chown=gituser:gituser <<EOF /app/start.sh
#!/bin/sh
cd /app/ts-service && node dist/server.js & 
cd /app && ./simplegit
EOF

RUN chmod +x /app/start.sh

EXPOSE 3000 2222

CMD ["/app/start.sh"]