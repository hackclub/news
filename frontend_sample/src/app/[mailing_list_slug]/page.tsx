import Link from "next/link";
import {
  getMailingListBySlug,
  getEmailsForMailingList,
  getAllMailingLists,
} from "@/lib/cms";
import { EmailCard } from "@/components/EmailCard";
import { notFound } from "next/navigation";
import { getClosestColor } from "@/lib/utils";
import Icon from "@hackclub/icons";

// Force static generation
export const dynamic = "force-static";
export const revalidate = false;

// Generate static params for all mailing lists
export async function generateStaticParams() {
  const mailingLists = await getAllMailingLists();

  return mailingLists.map((mailingList) => ({
    mailing_list_slug: mailingList.slug,
  }));
}

interface MailingListPageProps {
  params: Promise<{
    mailing_list_slug: string;
  }>;
}

export default async function MailingListPage({
  params,
}: MailingListPageProps) {
  const { mailing_list_slug } = await params;
  const mailingList = await getMailingListBySlug(mailing_list_slug);

  // Get emails first to see if any exist for this mailing list
  const emails = await getEmailsForMailingList(mailing_list_slug);

  if (emails.length === 0) {
    notFound();
  }

  const corrected_color = getClosestColor(
    mailingList ? mailingList.color : emails[0].mailing_list.color,
  );

  // If mailing list doesn't exist in mailing lists endpoint but emails exist,
  // create a fallback mailing list object from the first email's mailing list data
  const displayMailingList = mailingList || emails[0].mailing_list;

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-4xl mx-auto px-6 py-12">
        <header className="mb-16">
          <Link
            href="/"
            className="inline-flex items-center mb-8 transition-colors"
            style={{ color: corrected_color }}
          >
            <Icon glyph="view-back" className="h-4 w-4 mr-2" />
            Back to all emails
          </Link>

          <div className="mb-8">
            <div className="flex items-center space-x-4 mb-4">
              <div
                className="w-4 h-4 rounded-full"
                style={{ backgroundColor: corrected_color }}
              />
              <h1 className="text-5xl font-bold  tracking-tight text-primary">
                {displayMailingList.name}
              </h1>
            </div>

            <p className="text-xl text-gray-600 leading-relaxed mb-6">
              {displayMailingList.description}
            </p>

            <div className="flex items-center space-x-6 text-sm text-gray-500">
              <div className="flex items-center space-x-2">
                <svg
                  className="w-4 h-4"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
                  />
                </svg>
                <span>
                  {displayMailingList.subscriber_count?.toLocaleString() ||
                    "N/A"}{" "}
                  subscribers
                </span>
              </div>

              <div className="flex items-center space-x-2">
                <svg
                  className="w-4 h-4"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M3 8l7.89 4.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
                  />
                </svg>
                <span>{emails.length} emails</span>
              </div>

              {displayMailingList.last_sent_at && (
                <div className="flex items-center space-x-2">
                  <svg
                    className="w-4 h-4"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
                    />
                  </svg>
                  <span>
                    Last sent{" "}
                    {new Date(
                      displayMailingList.last_sent_at,
                    ).toLocaleDateString()}
                  </span>
                </div>
              )}
            </div>
          </div>
        </header>

        <main>
          {emails.length === 0 ? (
            <div className="text-center py-16">
              <div className="inline-flex items-center justify-center w-16 h-16 bg-gray-100 rounded-full mb-4">
                <svg
                  className="w-8 h-8 text-gray-400"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M3 8l7.89 4.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
                  />
                </svg>
              </div>
              <h3 className="text-xl font-semibold text-gray-900 mb-2">
                No emails found
              </h3>
              <p className="text-gray-500">
                This mailing list doesnt have any emails yet.
              </p>
            </div>
          ) : (
            <div className="space-y-12">
              {emails.map((email, index) => (
                <EmailCard
                  key={email.id}
                  email={email}
                  showMailingList={false}
                  index={index}
                />
              ))}
            </div>
          )}
        </main>
      </div>
    </div>
  );
}
