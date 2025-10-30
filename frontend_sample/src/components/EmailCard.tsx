"use client";
import Link from "next/link";
import { Email } from "@/lib/cms";
import { MailingListBadge } from "./MailingListBadge";
import { EmailStats } from "./EmailStats";
import { getClosestColor } from "@/lib/utils";

interface EmailCardProps {
  email: Email;
  showMailingList?: boolean;
  index?: number;
}

export function EmailCard({ email, showMailingList = true }: EmailCardProps) {
  const formatDate = (dateString: string | null | undefined) => {
    if (!dateString) return "Date not available";
    try {
      return new Date(dateString).toLocaleDateString("en-US", {
        year: "numeric",
        month: "long",
        day: "numeric",
      });
    } catch {
      return "Date not available";
    }
  };

  const corrected_list_color = getClosestColor(email.mailing_list.color);

  return (
    <article className="group bg-dark p-4 rounded-2xl">
      <div className="mb-4">
        <div className="text-sm text-muted mb-2">
          {formatDate(email.sent_at)}
        </div>

        <h2 className="text-2xl font-semibold text-primary mb-3 transition-colors leading-tight">
          <Link
            href={`/${email.mailing_list.slug}/${email.slug}`}
            style={{ color: "inherit" }}
            onMouseEnter={(e) =>
              (e.currentTarget.style.color = corrected_list_color)
            }
            onMouseLeave={(e) => (e.currentTarget.style.color = "")}
          >
            {email.subject}
          </Link>
        </h2>

        {email.internal_title && (
          <p className="text-lg text-smoke mb-4 font-medium">
            {email.internal_title}
          </p>
        )}
      </div>

      <div className="prose prose-gray max-w-none mb-6">
        <p className="text-smoke leading-relaxed">{email.excerpt}</p>
      </div>

      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          {showMailingList && (
            <MailingListBadge mailingList={email.mailing_list} />
          )}

          <div className="flex items-center space-x-2 text-sm text-gray-500">
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
                d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
              />
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
              />
            </svg>
            <EmailStats
              emailId={email.id}
              initialViews={email.stats.views}
              initialClicks={email.stats.clicks}
            />
          </div>
        </div>

        <Link
          href={`/${email.mailing_list.slug}/${email.slug}`}
          className="font-medium text-sm transition-colors"
          style={{ color: corrected_list_color }}
        >
          Read more →
        </Link>
      </div>
    </article>
  );
}
