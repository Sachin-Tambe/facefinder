"use client";
import { useEffect, useState } from "react"; import { api, requireAuth } from "@/lib/api";
import Icon from "@/components/icons";
import { CountUp, Reveal } from "@/components/motion";

export default function Dashboard() {
  const [stats, setStats] = useState<any>({}); const [orgs, setOrgs] = useState<any[]>([]); const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    requireAuth();
    (async () => {
      const o = await api("/api/v1/organizations"); setOrgs(o);
      if (o[0] && !localStorage.getItem("ff_org")) localStorage.setItem("ff_org", o[0].ID || o[0].id);
      if (o[0]) setStats(await api("/api/v1/organizations/stats"));
    })().catch(() => {});
  }, []);
  async function createOrg() {
    if (!name.trim()) return; setBusy(true);
    try { const o = await api("/api/v1/organizations", { method: "POST", body: JSON.stringify({ Name: name }) }); localStorage.setItem("ff_org", o.ID || o.id); location.reload(); }
    catch { setBusy(false); }
  }
  if (!orgs.length) return (
    <Reveal className="section">
      <h1>Set up your studio</h1>
      <p className="muted">Your studio holds every event, photo, and guest search in one isolated space.</p>
      <div className="panel" style={{ maxWidth: 480 }}>
        <label>Studio name<input placeholder="Alex Photography" value={name} onChange={e => setName(e.target.value)} /></label>
        <button onClick={createOrg} disabled={busy || !name.trim()}>{busy ? "Creating…" : "Create studio"}</button>
      </div>
    </Reveal>
  );
  const s = (k: string) => stats[k] ?? 0;
  return (
    <div>
      <Reveal className="section">
        <div className="section-head">
          <div><h1>{orgs[0]?.Name || orgs[0]?.name || "Studio"}</h1>
          <p className="muted">Here is how the season is shaping up.</p></div>
          <a href="/events/new"><button><Icon name="plus" /> New event</button></a>
        </div>
        <div className="ledger">
          <div><b className="hot"><CountUp value={s("active_events")} /></b><span>Active events</span></div>
          <div><b><CountUp value={s("total_photos")} /></b><span>Photos uploaded</span></div>
          <div><b><CountUp value={s("indexed_faces")} /></b><span>Faces indexed</span></div>
          <div><b><CountUp value={s("guest_searches")} /></b><span>Guest searches</span></div>
        </div>
      </Reveal>
      <Reveal className="section" delay={120}>
        <div className="section-head"><h2>Getting going</h2></div>
        <div className="panel">
          <p>Create an event, upload the shoot, print the QR code. Guests handle the rest with one selfie on WhatsApp.</p>
          <div className="row">
            <a href="/events/new"><button><Icon name="plus" /> Create event</button></a>
            <a href="/events"><button className="secondary">Browse events <Icon name="arrow" /></button></a>
          </div>
        </div>
      </Reveal>
    </div>
  );
}
