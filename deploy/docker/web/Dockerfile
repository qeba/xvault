# Build stage
FROM node:20-alpine AS builder

WORKDIR /src

# Copy package files from web directory
COPY web/package*.json ./
RUN npm ci

# Copy source files from web directory
COPY web/ ./

# Build the application
RUN npm run build

# Production stage
FROM nginx:alpine AS production

# Copy built files from builder
COPY --from=builder /src/dist /usr/share/nginx/html

# Copy nginx config
COPY deploy/docker/web/nginx.conf /etc/nginx/conf.d/default.conf

# Expose port
EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
