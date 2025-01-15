FROM node:20-alpine

WORKDIR /app

# Install dependencies
COPY package*.json ./
RUN npm install

# Copy source code
COPY tsconfig.json .
COPY src/ src/

# Build TypeScript
RUN npm run build

EXPOSE 3001
CMD ["npm", "start"]

