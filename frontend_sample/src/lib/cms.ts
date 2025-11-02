// CMS API client for Hack Club Email CMS
import { FETCH_CACHE_CONFIG } from './isr-config';

const CMS_BASE_URL = process.env.CMS_API_BASE_URL || 'http://localhost:8080';

export interface MailingList {
  id: string;
  slug: string;
  name: string;
  description: string;
  color: string;
  is_public: boolean;
  subscriber_count: number;
  last_updated_at: string;
  last_sent_at: string | null;
  sent_email_count: number;
}

export interface EmailStats {
  sent_count: number;
  opens: number;
  clicks: number;
  views: number;
  unsubscribes: number;
  hard_bounces: number;
  soft_bounces: number;
  open_rate: number;
  click_rate: number;
}

export interface Email {
  id: string;
  slug: string;
  subject: string;
  internal_title: string | null;
  emoji: string | null;
  sent_at: string | null;
  mailing_list_id: string;
  mailing_list: MailingList;
  stats: EmailStats;
  html: string;
  markdown: string;
  content_json: any;
  preview_text: string;
  excerpt: string;
}

export interface MailingListWithEmails {
  mailing_list: MailingList;
  emails: Email[];
}

export interface ApiResponse<T> {
  items: T[];
  next_offset?: number;
}

// Fetch emails with optional filtering by mailing list
export async function getEmails(
  limit: number = 50,
  offset: number = 0,
  mailing_list_id?: string
): Promise<ApiResponse<Email>> {
  const params = new URLSearchParams({
    limit: limit.toString(),
    offset: offset.toString(),
  });
  
  if (mailing_list_id) {
    params.append('mailing_list_id', mailing_list_id);
  }

  const response = await fetch(`${CMS_BASE_URL}/emails?${params}`, FETCH_CACHE_CONFIG);

  if (!response.ok) {
    throw new Error(`Failed to fetch emails: ${response.statusText}`);
  }

  return response.json();
}

// Fetch mailing lists
export async function getMailingLists(
  limit: number = 50,
  offset: number = 0
): Promise<ApiResponse<MailingList>> {
  const params = new URLSearchParams({
    limit: limit.toString(),
    offset: offset.toString(),
  });

  const response = await fetch(`${CMS_BASE_URL}/mailing_lists?${params}`, FETCH_CACHE_CONFIG);

  if (!response.ok) {
    throw new Error(`Failed to fetch mailing lists: ${response.statusText}`);
  }

  return response.json();
}

// Get latest email from each mailing list
export async function getMailingListsWithEmails(
  group_all: boolean = false,
  limit_per_list: number = 1
): Promise<MailingListWithEmails[]> {
  const params = new URLSearchParams({
    group_all: group_all.toString(),
    limit_per_list: limit_per_list.toString(),
  });

  const response = await fetch(`${CMS_BASE_URL}/mailing_lists/emails?${params}`, FETCH_CACHE_CONFIG);

  if (!response.ok) {
    throw new Error(`Failed to fetch mailing lists with emails: ${response.statusText}`);
  }

  return response.json();
}

// Find email by slug (searches through all emails)
export async function getEmailBySlug(slug: string, skipCache: boolean = false): Promise<Email | null> {
  try {
    // Search through multiple pages of emails to find the one with matching slug
    let offset = 0;
    const limit = 50;
    
    while (true) {
      const params = new URLSearchParams({
        limit: limit.toString(),
        offset: offset.toString(),
      });

      const fetchOptions: RequestInit = skipCache 
        ? { cache: 'no-store' } // Bypass cache for dynamic pages
        : FETCH_CACHE_CONFIG;

      const response = await fetch(`${CMS_BASE_URL}/emails?${params}`, fetchOptions);

      if (!response.ok) {
        throw new Error(`Failed to fetch emails: ${response.statusText}`);
      }

      const data = await response.json();
      const email = data.items.find((email: Email) => email.slug === slug);
      
      if (email) {
        return email;
      }
      
      // If no more emails or no next_offset, we've searched everything
      if (!data.next_offset || data.items.length === 0) {
        break;
      }
      
      offset = data.next_offset;
    }
    
    return null;
  } catch (error) {
    console.error('Error finding email by slug:', error);
    return null;
  }
}

// Find mailing list by slug
export async function getMailingListBySlug(slug: string): Promise<MailingList | null> {
  try {
    const response = await getMailingLists(200, 0); // Get more lists to search through
    const mailingList = response.items.find(list => list.slug === slug);
    return mailingList || null;
  } catch (error) {
    console.error('Error finding mailing list by slug:', error);
    return null;
  }
}

// Get all mailing lists (including those only found in emails)
export async function getAllMailingLists(): Promise<MailingList[]> {
  try {
    // First get mailing lists from the dedicated endpoint
    const mailingListsResponse = await getMailingLists(200, 0);
    const mailingListsFromEndpoint = mailingListsResponse.items;
    
    // Then get emails to find any additional mailing lists
    const emailsResponse = await getEmails(200, 0);
    const emails = emailsResponse.items;
    
    // Extract unique mailing lists from emails
    const mailingListsFromEmails = emails.reduce((acc: MailingList[], email) => {
      const existingList = acc.find(list => list.slug === email.mailing_list.slug);
      if (!existingList) {
        acc.push(email.mailing_list);
      }
      return acc;
    }, []);
    
    // Merge both sources, prioritizing the endpoint data
    const allMailingLists = [...mailingListsFromEndpoint];
    
    // Add mailing lists from emails that aren't in the endpoint
    mailingListsFromEmails.forEach(emailList => {
      const existsInEndpoint = mailingListsFromEndpoint.some(endpointList => 
        endpointList.slug === emailList.slug
      );
      if (!existsInEndpoint) {
        allMailingLists.push(emailList);
      }
    });
    
    return allMailingLists;
  } catch (error) {
    console.error('Error getting all mailing lists:', error);
    return [];
  }
}

// Get emails for a specific mailing list
export async function getEmailsForMailingList(slug: string): Promise<Email[]> {
  const mailingList = await getMailingListBySlug(slug);
  if (!mailingList) {
    // If mailing list doesn't exist, try to get emails by mailing_list_id from email data
    try {
      const emailsResponse = await getEmails(200, 0);
      const emails = emailsResponse.items.filter(email => email.mailing_list.slug === slug);
      return emails;
    } catch (error) {
      console.error('Error getting emails for mailing list:', error);
      return [];
    }
  }

  const response = await getEmails(200, 0, mailingList.id);
  return response.items;
}
