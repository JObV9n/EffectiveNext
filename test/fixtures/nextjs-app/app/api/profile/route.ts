import { NextResponse } from 'next/server';
import { getProfile } from '@/lib/profile';

export async function GET() {
  const profile = await getProfile();
  return NextResponse.json(profile);
}

export async function PUT(request: Request) {
  const body = await request.json();
  // Update profile logic here
  return NextResponse.json({ success: true });
}
