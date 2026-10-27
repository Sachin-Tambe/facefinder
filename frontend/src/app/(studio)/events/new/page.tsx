"use client";
import { useState } from "react"; import { api } from "@/lib/api";
import Icon from "@/components/icons";

export default function NewEvent() {
  const [name, setName] = useState(""); const [description, setDescription] = useState("");
  const [created, setCreated] = useState<any>(null); const [err, setErr] = useState(""); const [busy, setBusy] = useState(false);
  async function go(e: any) {
    e.preventDefault(); setBusy(true); setErr("");
    try { setCreated(await api("/api/v1/events", { method: "POST", body: JSON.stringify({ Name: name, Description: description }) })); }
    catch (e: any) { setErr(e.message); setBusy(false); }
  }
  if (created) return (
    <div className="section" style={{ maxWidth: 640 }}>
      <h1>{created.Name} is live</h1>
      <p className="muted">Its face collection is ready and it has its own guest code. The entry token below is shown once — save it with the QR code.</p>
      <div className="panel">
        <p>Event code <b>{created.EventCode}</b></p>
        <p className="linkline">WhatsApp entry <a href={created.WhatsAppLink}>{created.WhatsAppLink}</a></p>
        <div className="row">
          <a href={`/events/${created.ID}`}><button>Open the event <Icon name="arrow" /></button></a>
          <a href="/events"><button className="secondary">Back to events</button></a>
        </div>
      </div>
    </div>
  );
  return (
    <div className="section" style={{ maxWidth: 640 }}>
      <h1>New event</h1>
      <p className="muted">Each event gets an isolated face collection, its own QR code, and independent expiry.</p>
      <form className="panel" onSubmit={go} style={{ marginTop: 16 }}>
        {err && <div className="alert error">{err}</div>}
        <label>Event name<input value={name} onChange={e => setName(e.target.value)} required placeholder="Rahul Wedding" /></label>
        <label>Description<textarea value={description} onChange={e => setDescription(e.target.value)} placeholder="Venue, date, anything the team should know" /></label>
        <button disabled={busy}>{busy ? "Setting up the collection…" : "Create event"}</button>
      </form>
    </div>
  );
}
