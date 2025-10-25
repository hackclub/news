'use client';

import { useState, useMemo } from 'react';
import { EmailCard } from '@/components/EmailCard';
import { EmailFilter } from '@/components/EmailFilter';
import { Email, MailingList } from '@/lib/cms';

interface HomePageClientProps {
  emails: Email[];
  mailingLists: MailingList[];
  hasMore: boolean;
}

export function HomePageClient({ emails, mailingLists, hasMore }: HomePageClientProps) {
  const [filters, setFilters] = useState({
    includeLists: [] as string[],
    excludeLists: [] as string[],
  });

  const filteredEmails = useMemo(() => {
    return emails.filter(email => {
      const emailListSlug = email.mailing_list.slug;
      
      // If include lists are specified, only show emails from those lists
      if (filters.includeLists.length > 0) {
        if (!filters.includeLists.includes(emailListSlug)) {
          return false;
        }
      }
      
      // If exclude lists are specified, hide emails from those lists
      if (filters.excludeLists.length > 0) {
        if (filters.excludeLists.includes(emailListSlug)) {
          return false;
        }
      }
      
      return true;
    });
  }, [emails, filters]);

  return (
    <>
      <EmailFilter 
        mailingLists={mailingLists} 
        onFilterChange={setFilters}
      />
      
      <main>
        {filteredEmails.length === 0 ? (
          <div className="text-center py-16">
            <div className="inline-flex items-center justify-center w-16 h-16 bg-gray-100 rounded-full mb-4">
              <svg className="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.172 16.172a4 4 0 015.656 0M9 12h6m-6-4h6m2 5.291A7.962 7.962 0 0112 15c-2.34 0-4.29-1.009-5.824-2.709M15 6.291A7.962 7.962 0 0012 5c-2.34 0-4.29 1.009-5.824 2.709" />
              </svg>
            </div>
            <h3 className="text-xl font-semibold text-gray-900 mb-2">No emails found</h3>
            <p className="text-gray-500">
              {emails.length === 0 
                ? "Check back later for new updates!" 
                : "Try adjusting your filters to see more emails."
              }
            </p>
          </div>
        ) : (
          <div className="space-y-12">
            {filteredEmails.map((email, index) => (
              <EmailCard key={email.id} email={email} index={index} />
            ))}
          </div>
        )}
      </main>

      {hasMore && (
        <div className="mt-16 text-center">
          <div className="inline-flex items-center px-4 py-2 bg-gray-50 rounded-lg text-gray-600 text-sm">
            <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            Showing {filteredEmails.length} of {emails.length} emails • More available
          </div>
        </div>
      )}
    </>
  );
}
