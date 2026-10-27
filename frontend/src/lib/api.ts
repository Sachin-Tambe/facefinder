export const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
export function token() { return typeof window === "undefined" ? "" : localStorage.getItem("ff_token") || ""; }
export function orgId() { return typeof window === "undefined" ? "" : localStorage.getItem("ff_org") || ""; }
export async function api(path: string, opts: RequestInit = {}) {
  const r = await fetch(API + path, { ...opts, headers: { "Content-Type": "application/json", Authorization: "Bearer " + token(), "X-Org-Id": orgId(), ...(opts.headers || {}) } });
  const data = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error((data as any).error || r.statusText);
  return data;
}
export function requireAuth() { if (typeof window !== "undefined" && !token()) location.href = "/login"; }
