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

# Create a script to wait for CMS health check
RUN echo '#!/bin/sh\n\
set -e\n\
echo "Waiting for CMS to be ready..."\n\
max_attempts=60\n\
attempt=1\n\
while [ $attempt -le $max_attempts ]; do\n\
  echo "Attempt $attempt/$max_attempts: Checking CMS health..."\n\
  if wget --spider --timeout=5 --tries=1 "$CMS_API_BASE_URL/healthz" 2>/dev/null; then\n\
    echo "CMS is ready!"\n\
    exit 0\n\
  fi\n\
  echo "CMS not ready yet, waiting 1 second..."\n\
  sleep 1\n\
  attempt=$((attempt + 1))\n\
done\n\
echo "ERROR: CMS did not become ready within 60 seconds"\n\
exit 1' > /wait-for-cms.sh && chmod +x /wait-for-cms.sh

# Wait for CMS to be ready before building
RUN /wait-for-cms.sh

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
