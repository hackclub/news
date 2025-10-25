'use client';

import { useEffect } from 'react';

export function ViewTracker({ emailId }: { emailId: string }) {
  useEffect(() => {
    // Only track the view - no SSE streaming
    async function trackView() {
      try {
        await fetch(`http://localhost:8080/emails/${emailId}/view`, {
          credentials: 'include',
        });
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
