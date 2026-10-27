"use client";
import { createAuthClient } from "@neondatabase/auth/next";
export const authClient: any = createAuthClient();
export const NEON_ENABLED = process.env.NEXT_PUBLIC_NEON_AUTH_ENABLED === "true";
