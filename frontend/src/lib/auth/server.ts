import { createNeonAuth } from "@neondatabase/auth/next/server";

// Neon Auth (Better Auth hosted by Neon). Null when not configured -> legacy Go auth is used.
export const auth: any = process.env.NEON_AUTH_BASE_URL
  ? createNeonAuth({ baseUrl: process.env.NEON_AUTH_BASE_URL, cookies: { secret: process.env.NEON_AUTH_COOKIE_SECRET || "dev-cookie-secret-at-least-32-chars!!" } })
  : null;
export const neonAuthEnabled = !!auth;
