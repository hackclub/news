'use client';

import { useState, useEffect } from 'react';

export function ViewCount({ emailId, initialViews }: { emailId: string; initialViews: number }) {
  const [views, setViews] = useState<number>(initialViews);
  const [error, setError] = useState(false);
  const [isClient, setIsClient] = useState(false);

  useEffect(() => {
    setIsClient(true);
  }, []);

  useEffect(() => {
    if (!isClient) return;
    
    let eventSource: EventSource | null = null;
    
    try {
      eventSource = new EventSource(`http://localhost:8080/emails/${emailId}/views/stream`);
      
      eventSource.onopen = () => {
        console.log(`EventSource opened for email ${emailId}`);
      };
      
      eventSource.onmessage = (event) => {
        const newViews = parseInt(event.data, 10);
        setViews(newViews);
        console.log(`ViewCount updated for ${emailId}: ${newViews}`);
      };
      
      eventSource.onerror = (error) => {
        console.log(`EventSource error for ${emailId}:`, error);
        eventSource?.close();
        setError(true);
      };
    } catch (err) {
      console.log(`EventSource creation failed for ${emailId}:`, err);
      setError(true);
    }
    
    // Cleanup: close SSE connection on unmount
    return () => {
      eventSource?.close();
    };
  }, [emailId, isClient]);

  // Prevent hydration mismatch by not rendering until client-side
  if (!isClient) {
    return <span>{initialViews.toLocaleString()} views</span>;
  }

  if (error) {
    return <span>{initialViews.toLocaleString()} views</span>; // Fallback to initial
  }
  
  return <span>{views.toLocaleString()} views</span>;
}
