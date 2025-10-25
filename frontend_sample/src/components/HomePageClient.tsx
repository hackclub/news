'use client';

import { useState, useMemo, useEffect, useCallback, useRef } from 'react';
import { EmailCard } from '@/components/EmailCard';
import { EmailFilter } from '@/components/EmailFilter';
import { Email, MailingList } from '@/lib/cms';

interface HomePageClientProps {
  initialPosts: Email[];
  initialNextOffset?: number;
  mailingLists: MailingList[];
}

export function HomePageClient({ initialPosts, initialNextOffset, mailingLists }: HomePageClientProps) {
  const [posts, setPosts] = useState<Email[]>(initialPosts);
  const [nextOffset, setNextOffset] = useState<number | undefined>(initialNextOffset);
  const [loading, setLoading] = useState(false);
  const [hasMore, setHasMore] = useState(!!initialNextOffset);
  const [filters, setFilters] = useState({
    includeLists: [] as string[],
    excludeLists: [] as string[],
  });

  const observerRef = useRef<IntersectionObserver | null>(null);
  const loadMoreRef = useRef<HTMLDivElement | null>(null);

  const filteredPosts = useMemo(() => {
    return posts.filter(post => {
      const postListSlug = post.mailing_list.slug;
      
      // If include lists are specified, only show posts from those lists
      if (filters.includeLists.length > 0) {
        if (!filters.includeLists.includes(postListSlug)) {
          return false;
        }
      }
      
      // If exclude lists are specified, hide posts from those lists
      if (filters.excludeLists.length > 0) {
        if (filters.excludeLists.includes(postListSlug)) {
          return false;
        }
      }
      
      return true;
    });
  }, [posts, filters]);

  const loadMorePosts = useCallback(async () => {
    if (loading || !nextOffset) return;

    setLoading(true);
    try {
      const response = await fetch(`/api/emails?limit=20&offset=${nextOffset}`);
      if (!response.ok) {
        throw new Error('Failed to fetch posts');
      }
      const data = await response.json();
      
      setPosts(prev => [...prev, ...data.items]);
      setNextOffset(data.next_offset);
      setHasMore(!!data.next_offset);
    } catch (error) {
      console.error('Error loading more posts:', error);
    } finally {
      setLoading(false);
    }
  }, [loading, nextOffset]);

  useEffect(() => {
    if (observerRef.current) {
      observerRef.current.disconnect();
    }

    observerRef.current = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loading) {
          loadMorePosts();
        }
      },
      { threshold: 0.1 }
    );

    if (loadMoreRef.current) {
      observerRef.current.observe(loadMoreRef.current);
    }

    return () => {
      if (observerRef.current) {
        observerRef.current.disconnect();
      }
    };
  }, [hasMore, loading, loadMorePosts]);

  return (
    <>
      <EmailFilter 
        mailingLists={mailingLists} 
        onFilterChange={setFilters}
      />
      
      <main>
        {filteredPosts.length === 0 ? (
          <div className="text-center py-16">
            <div className="inline-flex items-center justify-center w-16 h-16 bg-gray-100 rounded-full mb-4">
              <svg className="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.172 16.172a4 4 0 015.656 0M9 12h6m-6-4h6m2 5.291A7.962 7.962 0 0112 15c-2.34 0-4.29-1.009-5.824-2.709M15 6.291A7.962 7.962 0 0012 5c-2.34 0-4.29 1.009-5.824 2.709" />
              </svg>
            </div>
            <h3 className="text-xl font-semibold text-gray-900 mb-2">No posts found</h3>
            <p className="text-gray-500">
              {posts.length === 0 
                ? "Check back later for new updates!" 
                : "Try adjusting your filters to see more posts."
              }
            </p>
          </div>
        ) : (
          <div className="space-y-12">
            {filteredPosts.map((post, index) => (
              <EmailCard key={post.id} email={post} index={index} />
            ))}
          </div>
        )}
      </main>

      {/* Loading indicator and intersection observer target */}
      <div ref={loadMoreRef} className="mt-16 text-center">
        {loading ? (
          <div className="inline-flex items-center px-4 py-2 bg-gray-50 rounded-lg text-gray-600 text-sm">
            <svg className="animate-spin -ml-1 mr-3 h-4 w-4 text-gray-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            Loading more posts...
          </div>
        ) : hasMore ? (
          <div className="inline-flex items-center px-4 py-2 bg-gray-50 rounded-lg text-gray-600 text-sm">
            <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            Showing {filteredPosts.length} of {posts.length} posts • Scroll for more
          </div>
        ) : (
          <div className="inline-flex items-center px-4 py-2 bg-gray-50 rounded-lg text-gray-600 text-sm">
            <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
            All {posts.length} posts loaded
          </div>
        )}
      </div>
    </>
  );
}
