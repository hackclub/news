FROM node:20-alpine AS base

RUN corepack enable && corepack prepare pnpm@10.18.3 --activate

FROM base AS deps
WORKDIR /app

COPY package.json package-lock.json* pnpm-lock.yaml* ./
RUN pnpm install

FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

ENV NEXT_TELEMETRY_DISABLED=1
ENV CMS_API_BASE_URL=https://news-api.zachlatta.com
ENV NEXT_PUBLIC_CMS_API_BASE_URL=https://news-api.zachlatta.com

# Wait for CMS to be ready before building
RUN set -e && \
    echo "Waiting for CMS to be ready..." && \
    max_attempts=60 && \
    attempt=1 && \
    while [ $attempt -le $max_attempts ]; do \
      echo "Attempt $attempt/$max_attempts: Checking CMS health..." && \
      if wget --spider --timeout=5 --tries=1 "$CMS_API_BASE_URL/healthz" 2>/dev/null; then \
        echo "CMS is ready!" && \
        break; \
      fi && \
      echo "CMS not ready yet, waiting 1 second..." && \
      sleep 1 && \
      attempt=$((attempt + 1)); \
    done && \
    if [ $attempt -gt $max_attempts ]; then \
      echo "ERROR: CMS did not become ready within 60 seconds" && \
      exit 1; \
    fi

RUN pnpm run build

FROM base AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV CMS_API_BASE_URL=https://news-api.zachlatta.com
ENV NEXT_PUBLIC_CMS_API_BASE_URL=https://news-api.zachlatta.com

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

COPY --from=builder /app/public ./public

RUN mkdir .next
RUN chown nextjs:nodejs .next

COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]
