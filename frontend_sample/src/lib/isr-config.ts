/**
 * ISR (Incremental Static Regeneration) Configuration
 * 
 * Centralized configuration for all ISR settings across the application.
 * This allows us to change revalidation intervals in one place.
 */

// Revalidation interval in seconds (5 minutes)
export const REVALIDATE_INTERVAL = 300;

// Fetch cache configuration for Next.js fetch calls
export const FETCH_CACHE_CONFIG = {
  next: { revalidate: REVALIDATE_INTERVAL },
} as const;

// Route segment config for dynamic routes
export const DYNAMIC_ROUTE_CONFIG = {
  revalidate: REVALIDATE_INTERVAL,
  dynamicParams: true as const,
} as const;

// Route segment config for static routes
export const STATIC_ROUTE_CONFIG = {
  revalidate: REVALIDATE_INTERVAL,
} as const;

