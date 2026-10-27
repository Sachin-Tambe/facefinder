import { NextResponse } from "next/server";
import { auth } from "@/lib/auth/server";

// Trades a verified Neon Auth session (cookie, server-side) for a Go API JWT.
// The shared BETTER_AUTH_SECRET never reaches the browser.
export async function GET() {
  if (!auth) return NextResponse.json({ error: "Neon Auth not configured" }, { status: 503 });
  const { data: session } = await auth.getSession();
  const user = session?.user;
  if (!user?.email) return NextResponse.json({ error: "not signed in" }, { status: 401 });
  const api = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
  const r = await fetch(api + "/api/v1/auth/exchange", {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-Auth-Bridge-Secret": process.env.BETTER_AUTH_SECRET || "" },
    body: JSON.stringify({ Email: user.email, Name: user.name || "" }),
  });
  const data = await r.json().catch(() => ({}));
  if (!r.ok) return NextResponse.json({ error: data.error || "exchange failed" }, { status: 502 });
  return NextResponse.json(data);
}
