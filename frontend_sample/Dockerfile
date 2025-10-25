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
