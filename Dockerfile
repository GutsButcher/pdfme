# ============================================
# Builder Stage
# ============================================
FROM node:18-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm install --production && \
    npm cache clean --force

# ============================================
# Runtime Stage
# ============================================
FROM node:18-alpine

WORKDIR /app

# Copy package files
COPY package*.json ./

# Copy node_modules from builder
COPY --from=builder /app/node_modules ./node_modules

# Copy application code
COPY src ./src
COPY public ./public

# Create templates directory (will be mounted as volume)
RUN mkdir -p /app/templates

# Expose port
EXPOSE 3000

# Start the application
CMD ["node", "src/index.js"]
