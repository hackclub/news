import Link from 'next/link';
import { getEmailBySlug, getEmails } from '@/lib/cms';
import { MailingListBadge } from '@/components/MailingListBadge';
import { notFound } from 'next/navigation';
import { Metadata } from 'next';

// Force static generation
export const dynamic = 'force-static';
export const revalidate = false;

// Generate static params for all emails
export async function generateStaticParams() {
  const emailsResponse = await getEmails(1000, 0); // Get all emails for static generation
  const emails = emailsResponse.items;
  
  return emails.map((email) => ({
    mailing_list_slug: email.mailing_list.slug,
    email_slug: email.slug,
  }));
}

interface EmailDetailPageProps {
  params: Promise<{
    mailing_list_slug: string;
    email_slug: string;
  }>;
}

export async function generateMetadata({ params }: EmailDetailPageProps): Promise<Metadata> {
  const { email_slug } = await params;
  const email = await getEmailBySlug(email_slug);

  if (!email) {
    return {
      title: 'Email Not Found',
    };
  }

  return {
    title: email.subject,
    description: email.excerpt,
    openGraph: {
      title: email.subject,
      description: email.excerpt,
      type: 'article',
      publishedTime: email.sent_at || undefined,
      authors: ['Hack Club'],
      tags: [email.mailing_list.name],
    },
  };
}

export default async function EmailDetailPage({ params }: EmailDetailPageProps) {
  const { mailing_list_slug, email_slug } = await params;
  const email = await getEmailBySlug(email_slug);
  
  if (!email) {
    notFound();
  }

  const formatDate = (dateString: string | null | undefined) => {
    if (!dateString) {
      return 'Date not available';
    }
    try {
      return new Date(dateString).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch (error) {
      return 'Date not available';
    }
  };

  const formatStats = (rate: number) => {
    if (isNaN(rate) || rate === null || rate === undefined) {
      return '0.0%';
    }
    return `${(rate * 100).toFixed(1)}%`;
  };

  return (
    <div className="min-h-screen bg-white">
      <div className="max-w-4xl mx-auto px-6 py-12">
        <header className="mb-16">
          <Link
            href={`/${mailing_list_slug}`}
            className="inline-flex items-center text-blue-600 hover:text-blue-800 mb-8 transition-colors"
          >
            <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
            Back to {email.mailing_list.name}
          </Link>

          <div className="mb-8">
            <div className="text-sm text-gray-500 mb-4">
              {formatDate(email.sent_at)}
            </div>
            
            <h1 className="text-5xl font-bold text-gray-900 mb-6 tracking-tight leading-tight">
              {email.subject}
            </h1>

            <div className="flex items-center justify-between">
              <MailingListBadge mailingList={email.mailing_list} />

              <div className="flex items-center space-x-6 text-sm text-gray-500">
                <div className="flex items-center space-x-2">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                  </svg>
                  <span>{email.stats.opens.toLocaleString()} views</span>
                </div>

                <div className="flex items-center space-x-2">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122" />
                  </svg>
                  <span>{email.stats.clicks.toLocaleString()} link clicks</span>
                </div>
              </div>
            </div>
          </div>
        </header>
        
        <main className="prose prose-lg prose-gray max-w-none">
          <div dangerouslySetInnerHTML={{ __html: email.html }} />
        </main>
      </div>
    </div>
  );
}
