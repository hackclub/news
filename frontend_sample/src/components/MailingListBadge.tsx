import Link from 'next/link';
import { Email, MailingList } from '@/lib/cms';

interface MailingListBadgeProps {
  mailingList: MailingList;
  className?: string;
}

export function MailingListBadge({ mailingList, className = '' }: MailingListBadgeProps) {
  return (
    <Link
      href={`/${mailingList.slug}`}
      className={`inline-flex items-center px-3 py-1 rounded-md text-sm font-medium text-white hover:opacity-90 transition-opacity ${className}`}
      style={{ backgroundColor: mailingList.color }}
    >
      {mailingList.name}
    </Link>
  );
}
