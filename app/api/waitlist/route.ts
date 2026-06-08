import { NextRequest, NextResponse } from "next/server";

// Server-side only — no NEXT_PUBLIC_ prefix needed
const API_URL = process.env.API_URL ?? "http://localhost:8090";

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();

    // Proxy to Go API — keeps API_URL server-side only if needed.
    const res = await fetch(`${API_URL}/api/v1/waitlist`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email: body.email, source: "landing_page" }),
      // 5 second timeout
      signal: AbortSignal.timeout(5000),
    });

    const data = await res.json().catch(() => ({}));

    if (!res.ok) {
      return NextResponse.json(
        { error: data.error ?? "Something went wrong" },
        { status: res.status }
      );
    }

    return NextResponse.json({ message: "you're on the list" }, { status: 201 });
  } catch (err) {
    if (err instanceof Error && err.name === "TimeoutError") {
      return NextResponse.json({ error: "Request timed out" }, { status: 504 });
    }
    return NextResponse.json({ error: "Internal server error" }, { status: 500 });
  }
}
