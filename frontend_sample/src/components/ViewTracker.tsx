'use client';

import { useEffect, useRef } from 'react';

const CMS_BASE_URL = process.env.NEXT_PUBLIC_CMS_API_BASE_URL || 'http://localhost:8080';

export function ViewTracker({ emailId }: { emailId: string }) {
  const hasTracked = useRef(false);

  useEffect(() => {
    // Only track the view once per component mount
    if (hasTracked.current) return;
    
    async function trackView() {
      try {
        await fetch(`${CMS_BASE_URL}/emails/${emailId}/view`, {
          credentials: 'include',
        });
        hasTracked.current = true;
      } catch (err) {
        // Silently fail - tracking is not critical
        console.warn('Failed to track view:', err);
      }
    }
    
    trackView();
  }, [emailId]);

  // This component doesn't render anything - it just tracks views
  return null;
}
