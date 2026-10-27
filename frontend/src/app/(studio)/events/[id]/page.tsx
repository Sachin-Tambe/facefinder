"use client";
import { useEffect, useRef, useState, use } from "react";
import { api, API, token, orgId, requireAuth } from "@/lib/api";
import Icon from "@/components/icons";
import { Reveal } from "@/components/motion";

const label: Record<string, string> = { uploaded: "Uploading", queued: "Queued", processing: "Indexing faces", ready: "Ready", no_face: "No face found", failed: "Failed" };
const tone: Record<string, string> = { ready: "ok", failed: "bad", processing: "working", queued: "working", uploaded: "working" };

export default function EventDetail({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const [data, setData] = useState<any>(null); const [photos, setPhotos] = useState<any[]>([]);
  const [total, setTotal] = useState(0); const [qr, setQr] = useState(""); const [busy, setBusy] = useState("");
  const [over, setOver] = useState(false);
  const input = useRef<HTMLInputElement>(null);
  async function load() {
    try {
      setData(await api(`/api/v1/events/${id}`));
      const p = await api(`/api/v1/events/${id}/photos?limit=60`);
      setPhotos(p.photos || []); setTotal(p.total || 0);
    } catch {}
  }
  useEffect(() => { requireAuth(); load(); const t = setInterval(load, 5000); return () => clearInterval(t); }, []);
  async function regen() {
    const d = await api(`/api/v1/events/${id}/qr/regenerate`, { method: "POST" }); setQr(d.whatsapp_link);
    const r = await fetch(`${API}/api/v1/events/${id}/qr/regenerate?format=png`, { method: "POST", headers: { Authorization: "Bearer " + token(), "X-Org-Id": orgId() } });
    if (r.ok) { const u = URL.createObjectURL(await r.blob()); const i = document.getElementById("qrimg") as HTMLImageElement | null; if (i) { i.src = u; i.style.display = "block"; } }
  }
  async function upload(list: FileList | File[] | null) {
    if (!list) return; const files = Array.from(list).filter(f => f.type.startsWith("image/"));
    if (!files.length) return; setBusy(`Uploading 0 of ${files.length}…`);
    let done = 0;
    for (const f of files) {
      try {
        const p = await api(`/api/v1/events/${id}/uploads/presign`, { method: "POST", body: JSON.stringify({ FileName: f.name, Mime: f.type || "image/jpeg", Size: f.size }) });
        await fetch(p.upload_url, { method: "PUT", headers: { "Content-Type": f.type || "image/jpeg" }, body: f });
        await api(`/api/v1/events/${id}/uploads/complete`, { method: "POST", body: JSON.stringify({ PhotoID: p.photo_id, Key: p.key }) });
      } catch (e) { console.error(e); }
      done++; setBusy(`Uploading ${done} of ${files.length}…`);
    }
    setBusy(""); load();
  }
  async function del() { if (!confirm("Delete this event, its face collection, and all its photos? This cannot be undone.")) return; await api(`/api/v1/events/${id}`, { method: "DELETE" }); location.href = "/events"; }
  if (!data) return <p className="muted">Loading the event…</p>;
  const ev = data.event, st = data.stats || {};
  const finished = (st.Ready || 0) + (st.NoFace || 0) + (st.Failed || 0);
  const pct = st.TotalPhotos ? Math.round((finished / st.TotalPhotos) * 100) : 0;
  return (
    <div>
      <Reveal className="section">
        <h1>{ev.Name}</h1>
        <p className="muted">Code {ev.EventCode} · {ev.Status === "active" ? "Accepting guest selfies" : ev.Status} · Face data expires {ev.FaceDataExpiresAt ? new Date(ev.FaceDataExpiresAt).toLocaleDateString() : "—"}</p>
        <div className="ledger">
          <div><b className={pct === 100 && st.TotalPhotos ? "" : "hot"}>{pct}%</b><span>Indexed</span></div>
          <div><b>{st.TotalPhotos || 0}</b><span>Photos</span></div>
          <div><b>{st.IndexedFaces || 0}</b><span>Faces indexed</span></div>
          <div><b>{st.Failed || 0}</b><span>Failed</span></div>
          <div><b>{st.Searches || 0}</b><span>Guest searches</span></div>
        </div>
        {(st.TotalPhotos || 0) > 0 && <div className="progress"><i style={{ width: pct + "%" }} /></div>}
        <p className="muted small">{finished} of {st.TotalPhotos || 0} photos finished · {st.SuccessfulSearches || 0} searches found photos</p>
      </Reveal>

      <Reveal className="section" delay={90}>
        <div className="section-head"><h2>Upload photos</h2>{(st.Failed || 0) > 0 && <button className="secondary" onClick={() => api(`/api/v1/events/${id}/retry-failed`, { method: "POST" }).then(load)}><Icon name="refresh" /> Retry {st.Failed} failed</button>}</div>
        <input ref={input} type="file" accept="image/*" multiple style={{ display: "none" }} onChange={e => upload(e.target.files)} />
        <div className={`dropzone${over ? " over" : ""}`}
          onClick={() => input.current?.click()}
          onDragOver={e => { e.preventDefault(); setOver(true); }}
          onDragLeave={() => setOver(false)}
          onDrop={e => { e.preventDefault(); setOver(false); upload(e.dataTransfer.files); }}>
          <Icon name="upload" style={{ width: 26, height: 26, color: "#e8a33d" }} />
          <p><b>Drop photos here, or click to browse</b></p>
          <p className="muted small">{busy || "Straight to secure storage, then faces are indexed in the background. Thousands at a time."}</p>
        </div>
      </Reveal>

      <Reveal className="section" delay={180}>
        <div className="section-head"><h2>Contact sheet{total > 0 && ` · ${total}`}</h2></div>
        {total === 0 && (
          <div className="empty"><Icon name="image" /><h3>Nothing uploaded yet</h3>
          <p className="muted small">Upload your first batch above and watch faces get indexed here.</p></div>
        )}
        <div className="photos">
          {photos.map(({ photo: p, thumb_url }: any) => (
            <figure key={p.ID} className="film">
              {thumb_url ? <img src={thumb_url} alt={p.FileName} loading="lazy" /> : <div style={{ aspectRatio: 1, background: "#000" }} />}
              <figcaption className="frame-meta">
                <span className={`status ${tone[p.Status] || ""}`}>{label[p.Status] || p.Status}</span>
                <span>{p.FaceCount} face{p.FaceCount === 1 ? "" : "s"}</span>
              </figcaption>
            </figure>
          ))}
        </div>
      </Reveal>

      <Reveal className="section" delay={270}>
        <div className="section-head"><h2>Guest access</h2></div>
        <div className="panel">
          <p className="muted">One QR code per event. Guests scan it, WhatsApp opens with the event filled in, and one business number serves every event.</p>
          <div className="row">
            <button onClick={regen}><Icon name="qr" /> {qr ? "Regenerate QR code" : "Generate QR code"}</button>
            <a href={`/events/${id}/retention`}><button className="secondary"><Icon name="clock" /> Expiry settings</button></a>
            <button className="danger" onClick={del}>Delete event</button>
          </div>
          {qr && (
            <div className="qr-block">
              <img id="qrimg" alt="Event QR code" width={200} height={200} style={{ display: "none" }} />
              <div>
                <p className="linkline"><a href={qr}>{qr}</a></p>
                <p className="muted small">Print this code and place it at the venue. Regenerating invalidates the old one.</p>
              </div>
            </div>
          )}
          <p className="muted small" style={{ marginTop: 12 }}>Galleries last {ev.GalleryTTLHours}h · guest sessions {ev.WhatsAppSessionTTLHours}h · selfies deleted after {ev.SelfieRetentionMinutes} min</p>
        </div>
      </Reveal>
    </div>
  );
}
