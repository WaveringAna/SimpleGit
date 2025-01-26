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

# Install build dependencies for both SQLite and SSL
RUN apk add --no-cache \
    gcc \
    musl-dev \
    sqlite-dev \
    openssl-dev

WORKDIR /app
COPY . .

# Enable CGO and build with specific flags for SQLite support
ENV CGO_ENABLED=1
ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE"
RUN go build -o simplegit

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    git \
    sqlite \
    nodejs \
    openssl \
    ca-certificates \
    libssl3 \
    openssh

WORKDIR /app

# Copy binaries and assets from build stages
COPY --from=go-builder /app/simplegit .
COPY --from=ts-builder /app/ts-service/dist ./ts-service/dist
COPY --from=ts-builder /app/ts-service/node_modules ./ts-service/node_modules
COPY static/ static/
COPY templates/ templates/

RUN mkdir -p /app/repositories /app/data /app/ssh

# Add startup script
COPY  <<EOF /app/start.sh
#!/bin/sh
cd /app/ts-service && node dist/server.js & 
cd /app && ./simplegit
EOF

RUN chmod +x /app/start.sh

ENV TS_SERVICE_PORT=3001
EXPOSE 3000 2222 3001

CMD ["/app/start.sh"]
