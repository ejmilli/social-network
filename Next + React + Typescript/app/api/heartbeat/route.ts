import { NextResponse } from 'next/server';

export async function GET() {
  return NextResponse.json({ message: 'Heartbeat endpoint - server is running' }, { status: 200 });
}
