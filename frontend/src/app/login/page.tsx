"use client";
import { useState } from "react";
import { API } from "@/lib/api";
import { authClient, NEON_ENABLED } from "@/lib/auth/client";
import Icon from "@/components/icons";

export default function Login() {
  const [email, setEmail] = useState(""); const [password, setPassword] = useState("");
  const [err, setErr] = useState(""); const [busy, setBusy] = useState(false);
  async function go(e: any) {
    e.preventDefault(); setBusy(true); setErr("");
    try {
      if (NEON_ENABLED) {
        const { error } = await authClient.signIn.email({ email, password });
        if (error) throw new Error(error.message || "Sign in failed — check your email and password.");
        const r = await fetch("/api/backend-token"); const d = await r.json();
        if (!r.ok) throw new Error(d.error || "Could not create API session.");
        localStorage.setItem("ff_token", d.token);
      } else {
        const r = await fetch(API + "/api/v1/auth/login", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ email, password }) });
        const d = await r.json(); if (!r.ok) throw new Error(d.error);
        localStorage.setItem("ff_token", d.token);
      }
      location.href = "/dashboard";
    } catch (e: any) { setErr(e.message); setBusy(false); }
  }
  return (
    <div className="auth-shell">
      <form className="auth-card" onSubmit={go}>
        <div className="logo"><Icon name="camera" /></div>
        <h1>Welcome back</h1>
        <p className="muted center">Sign in to your studio.</p>
        {err && <div className="alert error">{err}</div>}
        <label>Email<input type="email" value={email} onChange={e => setEmail(e.target.value)} required placeholder="you@studio.com" /></label>
        <label>Password<input type="password" value={password} onChange={e => setPassword(e.target.value)} required placeholder="Your password" /></label>
        <button className="full" disabled={busy}>{busy ? "Signing in…" : "Sign in"}</button>
        <p className="muted center small">New to FaceFinder? <a href="/register">Create an account</a></p>
        {NEON_ENABLED && <p className="auth-note">Secured by Neon Auth</p>}
      </form>
    </div>
  );
}
