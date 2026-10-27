"use client";
import { useEffect, useState } from "react"; import { api, requireAuth } from "@/lib/api";
import Icon from "@/components/icons";
import { Reveal } from "@/components/motion";

export default function Events() {
  const [evs, setEvs] = useState<any[]>([]); const [loaded, setLoaded] = useState(false);
  useEffect(() => { requireAuth(); api("/api/v1/events").then(d => { setEvs(d); setLoaded(true); }).catch(() => setLoaded(true)); }, []);
  return (
    <div className="section">
      <div className="section-head">
        <div><h1>Events</h1><p className="muted">{evs.length === 0 ? "Nothing scheduled yet." : `${evs.length} event${evs.length === 1 ? "" : "s"} on the books.`}</p></div>
        <a href="/events/new"><button><Icon name="plus" /> New event</button></a>
      </div>
      {loaded && evs.length === 0 && (
        <div className="empty">
          <Icon name="calendar" />
          <h3>No events yet</h3>
          <p className="muted small">Create your first event to get its QR code and face collection.</p>
          <a href="/events/new"><button>Create event</button></a>
        </div>
      )}
      <Reveal delay={60}>
        {evs.map(e => (
          <a className="event-row" key={e.ID} href={`/events/${e.ID}`}>
            <div className="grow">
              <h3>{e.Name}<span className="code">{e.EventCode}</span></h3>
              <span className={`status ${e.Status === "active" ? "ok" : ""}`}>{e.Status === "active" ? "Accepting guest selfies" : e.Status}</span>
            </div>
            <span className="go">→</span>
          </a>
        ))}
      </Reveal>
    </div>
  );
}
