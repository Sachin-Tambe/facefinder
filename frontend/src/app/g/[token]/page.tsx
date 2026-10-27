"use client";
import { useEffect, useState, use } from "react"; import { API } from "@/lib/api";
import Icon from "@/components/icons";
import { Reveal } from "@/components/motion";

export default function Gallery({ params }: { params: Promise<{ token: string }> }) {
  const { token } = use(params);
  const [g, setG] = useState<any>(null); const [err, setErr] = useState(""); const [sel, setSel] = useState<any>(null);
  useEffect(() => { fetch(`${API}/api/v1/gallery/${token}`).then(async r => { const d = await r.json(); if (!r.ok) throw new Error(d.error || "expired"); setG(d) }).catch(() => setErr("This gallery has expired or the link is invalid.")) }, [token]);
  if (err) return (
    <div className="auth-shell"><div className="auth-card">
      <div className="logo"><Icon name="clock" /></div>
      <h1>This gallery has expired</h1>
      <p className="muted center">Private galleries are short-lived by design. Send a fresh selfie on WhatsApp to get a new link.</p>
    </div></div>
  );
  if (!g) return <div className="gallery-head"><p className="muted">Developing your photos…</p></div>;
  return (
    <div>
      <div className="gallery-head">
        <div className="event">{g.event_name}</div>
        <h1>We found {g.count} photo{g.count === 1 ? "" : "s"} of you</h1>
        <p className="muted">Tap any frame to view it{g.downloads_enabled ? " and save the original" : ""}. This link expires on its own — keep the ones you love.</p>
      </div>
      <div className="gallery-wrap">
        <div className="photos">
          {g.photos.map((p: any, i: number) => (
            <Reveal key={p.photo_id} delay={Math.min(i, 8) * 60}>
            <figure className="film" onClick={() => setSel(p)} style={{ cursor: "zoom-in" }}>
              <img src={p.web_url} alt={p.file_name} loading="lazy" />
              <div className="frame-no">Frame {String(i + 1).padStart(2, "0")}</div>
            </figure>
            </Reveal>
          ))}
        </div>
      </div>
      {sel && (
        <div className="lightbox" onClick={() => setSel(null)}>
          <img src={sel.web_url} alt={sel.file_name} style={{ maxWidth: "94vw", maxHeight: "78vh", borderRadius: 6 }} />
          <p className="row" style={{ marginTop: 16 }}>
            {g.downloads_enabled && sel.download_url && <a href={sel.download_url} download onClick={e => e.stopPropagation()}><button><Icon name="download" /> Save original</button></a>}
            <button className="secondary">Back to frames</button>
          </p>
        </div>
      )}
    </div>
  );
}
