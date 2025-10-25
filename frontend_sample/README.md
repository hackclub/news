# Hack Club News Frontend

A Next.js application for displaying Hack Club email newsletters using a custom headless CMS.

## Environment Variables

Create a `.env.local` file in the root directory with the following variables:

```bash
# CMS API Configuration
CMS_API_BASE_URL=http://localhost:8080
NEXT_PUBLIC_CMS_API_BASE_URL=http://localhost:8080
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

Update the environment variables to point to your production CMS API:

```bash
CMS_API_BASE_URL=https://your-cms-api.com
NEXT_PUBLIC_CMS_API_BASE_URL=https://your-cms-api.com
```