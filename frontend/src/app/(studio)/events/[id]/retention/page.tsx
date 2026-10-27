"use client";
import { useEffect, useState, use } from "react"; import { api } from "@/lib/api";
import Icon from "@/components/icons";

export default function Retention({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const [ev, setEv] = useState<any>(null); const [msg, setMsg] = useState(""); const [busy, setBusy] = useState(false);
  useEffect(() => { api(`/api/v1/events/${id}/retention`).then(setEv).catch(() => {}); }, []);
  if (!ev) return <p className="muted">Loading expiry settings…</p>;
  async function save() {
    setBusy(true);
    await api(`/api/v1/events/${id}/retention`, { method: "PATCH", body: JSON.stringify({ GalleryTTLHours: +ev.GalleryTTLHours, WhatsAppSessionTTLHours: +ev.WhatsAppSessionTTLHours, SelfieRetentionMinutes: +ev.SelfieRetentionMinutes, AutoDeleteEnabled: !!ev.AutoDeleteEnabled }) });
    setBusy(false); setMsg("Saved");
  }
  const num = (k: string) => (v: any) => { setEv({ ...ev, [k]: v.target.value }); setMsg(""); };
  return (
    <div className="section" style={{ maxWidth: 640 }}>
      <h1>Expiry for {ev.Name}</h1>
      <p className="muted">Each clock runs on its own. Face data can expire while photos stay, and photos only vanish when you allow it.</p>
      <div className="panel" style={{ marginTop: 16 }}>
        <label>Guest gallery lifetime (hours)<input type="number" min={1} value={ev.GalleryTTLHours} onChange={num("GalleryTTLHours")} /></label>
        <label>WhatsApp session lifetime (hours)<input type="number" min={1} value={ev.WhatsAppSessionTTLHours} onChange={num("WhatsAppSessionTTLHours")} /></label>
        <label>Guest selfie lifetime (minutes)<input type="number" min={5} value={ev.SelfieRetentionMinutes} onChange={num("SelfieRetentionMinutes")} /></label>
        <label><input type="checkbox" checked={!!ev.AutoDeleteEnabled} onChange={e => setEv({ ...ev, AutoDeleteEnabled: e.target.checked })} /> Delete expired face data and photos automatically</label>
        <p className="muted small">Face vectors expire {ev.FaceDataExpiresAt ? new Date(ev.FaceDataExpiresAt).toLocaleString() : "—"} (collection deleted, photos kept). Photos expire {ev.PhotosExpireAt ? new Date(ev.PhotosExpireAt).toLocaleString() : "—"}.</p>
        <div className="row"><button onClick={save} disabled={busy}><Icon name="check" /> {busy ? "Saving…" : "Save settings"}</button>{msg && <span className="status ok">Saved</span>}</div>
      </div>
    </div>
  );
}
