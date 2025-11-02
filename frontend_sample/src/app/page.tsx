import { getEmails, getAllMailingLists } from "@/lib/cms";
import { HomePageClient } from "@/components/HomePageClient";
import Image from "next/image";
import Icon from "@hackclub/icons";
import { STATIC_ROUTE_CONFIG } from "@/lib/isr-config";

// Use ISR: statically generate but revalidate every 5 minutes
export const revalidate = STATIC_ROUTE_CONFIG.revalidate;

export default async function Home() {
  const postsResponse = await getEmails(20, 0); // Start with smaller initial load
  const posts = postsResponse.items;
  const mailingLists = await getAllMailingLists();

  return (
    <div className="min-h-screen bg-background">
      <div className="relative h-32 bg-background mb-8">
        <a href="https://hackclub.com/">
          <Image
            src="https://assets.hackclub.com/flag-orpheus-top.svg"
            alt="Hack Club"
            width={256}
            height={128}
            style={{ position: "absolute", left: 30 }}
          />
        </a>
      </div>
      <div className="max-w-4xl mx-auto px-6 py-12">
        <header className="mb-16 text-start">
          <div className="mb-8 flex-col items-start justify-start">
            <div className="flex flex-row mb-4">
              <Icon
                glyph="announcement"
                className=" text-red h-auto w-12 mr-4"
              />
              <h1 className="text-5xl text-red font-bold">HACKCLUB NEWS!</h1>
            </div>
            <p className="text-xl text-muted">
              Dispatches from the Hack Club community
            </p>
          </div>

          <div className="flex items-start justify-start space-x-6 text-sm text-muted">
            <div className="flex items-center space-x-2 bg-dark">
              <Icon glyph="email" className="h-4 w-4" />
              <span className="text-snow">Loading posts...</span>
            </div>
            <div className="flex items-center space-x-2">
              <Icon glyph="rss" className="h-4 w-4" />
              <span className="text-snow">RSS</span>
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
