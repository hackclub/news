import { getEmails, getAllMailingLists } from '@/lib/cms';
import { HomePageClient } from '@/components/HomePageClient';

// Force static generation
export const dynamic = 'force-static';
export const revalidate = false;

export default async function Home() {
  const postsResponse = await getEmails(20, 0); // Start with smaller initial load
  const posts = postsResponse.items;
  const mailingLists = await getAllMailingLists();

  return (
    <div className="min-h-screen bg-white">
      <div className="max-w-4xl mx-auto px-6 py-12">
        <header className="mb-16 text-center">
          <div className="mb-8">
            <h1 className="text-5xl font-bold text-gray-900 mb-4 tracking-tight">
              News
            </h1>
            <p className="text-xl text-gray-600 max-w-2xl mx-auto leading-relaxed">
              Dispatches from the Hack Club community
            </p>
          </div>
          
          <div className="flex items-center justify-center space-x-6 text-sm text-gray-500">
            <div className="flex items-center space-x-2">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 4.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
              <span>Loading posts...</span>
            </div>
            <div className="flex items-center space-x-2">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
              </svg>
              <span>RSS</span>
            </div>
          </div>
        </header>

        <HomePageClient 
          initialPosts={posts}
          initialNextOffset={postsResponse.next_offset}
          mailingLists={mailingLists}
        />
      </div>
    </div>
  );
}