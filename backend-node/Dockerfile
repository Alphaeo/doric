FROM node:18-alpine

WORKDIR /app

# Copy proto files first
COPY proto ./proto

# Copy backend-node files
COPY backend-node/package*.json ./
RUN npm install

COPY backend-node/server.js ./
COPY backend-node/.env* ./

EXPOSE 50051

CMD ["node", "server.js"]
