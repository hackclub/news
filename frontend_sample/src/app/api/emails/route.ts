import { NextRequest, NextResponse } from 'next/server';
import { getEmails } from '@/lib/cms';

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const limit = parseInt(searchParams.get('limit') || '20');
    const offset = parseInt(searchParams.get('offset') || '0');
    const mailingListId = searchParams.get('mailing_list_id') || undefined;

    const postsResponse = await getEmails(limit, offset, mailingListId);

    return NextResponse.json(postsResponse);
  } catch (error) {
    console.error('Error fetching posts:', error);
    return NextResponse.json(
      { error: 'Failed to fetch posts' },
      { status: 500 }
    );
  }
}
