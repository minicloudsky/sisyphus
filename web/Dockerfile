# ─────────────────────────────────────────────────────────────────
# Stage 1 — Build
# ─────────────────────────────────────────────────────────────────
FROM node:20-alpine AS builder

WORKDIR /app

# Copy manifests first to leverage layer cache —
# npm ci only reruns when package-lock.json changes
COPY package*.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --prefer-offline

# Copy source and build
COPY . .
RUN npm run build

# ─────────────────────────────────────────────────────────────────
# Stage 2 — Runtime (nginx)
# ─────────────────────────────────────────────────────────────────
FROM nginx:1.27-alpine AS runtime

LABEL org.opencontainers.image.title="sisyphus-frontend" \
      org.opencontainers.image.description="Sisyphus DBA Platform — Frontend" \
      org.opencontainers.image.licenses="Apache-2.0"

# Copy built SPA assets
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy nginx site config (overwrites default)
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Drop nginx worker process to non-root
RUN sed -i 's/^user .*/user nginx;/' /etc/nginx/nginx.conf || true

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost/index.html || exit 1

CMD ["nginx", "-g", "daemon off;"]
