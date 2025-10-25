# Hack Club News Frontend

A Next.js application for displaying Hack Club email newsletters using a custom headless CMS.

## Environment Variables

Create a `.env.local` file in the root directory with the following variables:

```bash
# CMS API Configuration
CMS_API_BASE_URL=https://news-api.zachlatta.com
NEXT_PUBLIC_CMS_API_BASE_URL=https://news-api.zachlatta.com
```

### Variable Descriptions

- `CMS_API_BASE_URL`: Used by server-side components (cms.ts) for API calls
- `NEXT_PUBLIC_CMS_API_BASE_URL`: Used by client-side components (ViewTracker, ViewCount) for API calls and SSE streams

## Development

```bash
npm install
npm run dev
```

The application will be available at `http://localhost:3000`.

## Production Deployment

### Docker Deployment

The Dockerfile is configured with production environment variables. To deploy:

```bash
docker build -t hack-club-news .
docker run -p 3000:3000 hack-club-news
```

### Other Platforms

For other hosting platforms (Vercel, Netlify, etc.), set these environment variables:

```bash
CMS_API_BASE_URL=https://news-api.zachlatta.com
NEXT_PUBLIC_CMS_API_BASE_URL=https://news-api.zachlatta.com
```

### Custom API URL

To use a different CMS API, update the environment variables:

```bash
CMS_API_BASE_URL=https://your-cms-api.com
NEXT_PUBLIC_CMS_API_BASE_URL=https://your-cms-api.com
```