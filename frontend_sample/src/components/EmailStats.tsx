'use client';

import { useState, useEffect } from 'react';

const CMS_BASE_URL = process.env.NEXT_PUBLIC_CMS_API_BASE_URL || 'http://localhost:8080';

export function EmailStats({ 
  emailId, 
  initialViews, 
  initialClicks 
}: { 
  emailId: string; 
  initialViews: number; 
  initialClicks: number;
}) {
  const [views, setViews] = useState<number>(initialViews);
  const [clicks, setClicks] = useState<number>(initialClicks);
  const [error, setError] = useState(false);
  const [isClient, setIsClient] = useState(false);

  useEffect(() => {
    setIsClient(true);
  }, []);

  useEffect(() => {
    if (!isClient) return;
    
    let eventSource: EventSource | null = null;
    
    try {
      eventSource = new EventSource(`${CMS_BASE_URL}/emails/${emailId}/stats/stream`);
      
      eventSource.onopen = () => {
        console.log(`EventSource opened for email ${emailId}`);
      };
      
      eventSource.onmessage = (event) => {
        try {
          const stats = JSON.parse(event.data);
          setViews(stats.views ?? initialViews);
          setClicks(stats.clicks ?? initialClicks);
          console.log(`EmailStats updated for ${emailId}:`, stats);
        } catch (parseError) {
          console.error(`Failed to parse stats for ${emailId}:`, parseError);
          setError(true);
        }
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
  }, [emailId, isClient, initialViews, initialClicks]);

  // Prevent hydration mismatch by not rendering until client-side
  if (!isClient) {
    return (
      <span>
        {initialViews.toLocaleString()} views • {initialClicks.toLocaleString()} clicks
      </span>
    );
  }

  if (error) {
    return (
      <span>
        {initialViews.toLocaleString()} views • {initialClicks.toLocaleString()} clicks
      </span>
    );
  }
  
  return (
    <span>
      {views.toLocaleString()} views • {clicks.toLocaleString()} clicks
    </span>
  );
}
