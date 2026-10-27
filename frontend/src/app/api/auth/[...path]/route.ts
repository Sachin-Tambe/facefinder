import { NextResponse } from "next/server";
import { auth } from "@/lib/auth/server";

function unavailable() { return NextResponse.json({ error: "Neon Auth not configured" }, { status: 503 }); }
export async function GET(req: Request, ctx: any) { if (!auth) return unavailable(); return auth.handler().GET(req, ctx); }
export async function POST(req: Request, ctx: any) { if (!auth) return unavailable(); return auth.handler().POST(req, ctx); }
