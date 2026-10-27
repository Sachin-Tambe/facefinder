import type { CSSProperties } from "react";
import Icon from "@/components/icons";
import { Reveal } from "@/components/motion";

const d = (ms: number) => ({ "--d": `${ms}ms` }) as CSSProperties;
const k = (n: number) => ({ "--k": n }) as CSSProperties;

const frames: Array<[string, string, string]> = [
  ["A-01", "Couple portrait", "/landing/couple-portrait.jpg"],
  ["A-02", "Sangeet night", "/landing/sangeet.jpg"],
  ["A-03", "Vows", "/landing/vows.jpg"],
  ["A-04", "Reception", "/landing/reception.jpg"],
  ["A-05", "Afterparty", "/landing/afterparty.jpg"],
  ["A-06", "Table setting", "/landing/tablesetting.jpg"],
  ["A-07", "Rings", "/landing/rings.jpg"],
  ["A-08", "Send-off", "/landing/sendoff.jpg"],
];

type Bubble = { from: "in" | "out"; kind: "text" | "photo" | "typing" | "link"; body?: string; src?: string; link?: string };
const chat: Bubble[] = [
  { from: "out", kind: "text", body: "Hi! I was at the Rao wedding yesterday" },
  { from: "in", kind: "text", body: "Hello! Send one clear selfie and I'll find every photo you're in." },
  { from: "out", kind: "photo", src: "/landing/selfie.jpg" },
  { from: "in", kind: "typing" },
  { from: "in", kind: "text", body: "Found you in 14 photos." },
  { from: "in", kind: "link", link: "facefinder.link/g/8f3k2q", body: "Your private gallery · expires in 48 hours" },
  { from: "out", kind: "text", body: "Got them all. Thank you!" },
];

const steps: Array<[string, string, string]> = [
  ["01", "Upload the event", "Drop in a thousand photos or ten thousand. Indexing runs in the background while you keep shooting."],
  ["02", "Print one QR code", "Each event gets its own code. One WhatsApp number serves every event you run."],
  ["03", "Guests send a selfie", "They scan, say hello, and send one clear selfie. Matches arrive as previews with a private gallery link."],
];

export default function Home() {
  return (
    <div>
      <header className="topbar hero-anim" style={d(0)}>
        <a className="brand" href="/"><span className="brand-mark"><Icon name="camera" /></span> FaceFinder</a>
        <div className="row">
          <a href="/login"><button className="btn-ghost">Sign in</button></a>
          <a href="/register"><button>Create your first event</button></a>
        </div>
      </header>
      <section className="hero">
        <h1 className="hero-anim" style={d(60)}>Every guest finds their photos with one selfie</h1>
        <p className="lede hero-anim" style={d(150)}>Upload your event photos and we index every face. Guests scan your QR code, send a selfie on WhatsApp, and get back every photo they appear in — plus a private gallery link.</p>
        <div className="hero-cta hero-anim" style={d(230)}>
          <a href="/register"><button className="btn-lg">Create your first event</button></a>
          <a href="/login"><button className="btn-ghost btn-lg">Sign in</button></a>
        </div>
        <div className="marquee hero-anim" style={d(320)} aria-hidden>
          <div className="marquee-track">
            {[0, 1].map(copy => (
              <div className="marquee-group" key={copy}>
                {frames.map(([no, label, src]) => (
                  <div className="film" key={no}>
                    <img src={src} alt="" loading={copy === 0 ? "eager" : "lazy"} />
                    <div className="frame-no">{no} · {label}</div>
                  </div>
                ))}
              </div>
            ))}
          </div>
        </div>
      </section>
      <section className="demo">
        <Reveal>
          <div className="step">Live demo</div>
          <h2>Watch a guest get their photos</h2>
          <p className="muted">No app to install, no gallery to scroll. They scan the QR, say hello, send a selfie — matches land in the chat with a private gallery link.</p>
          <p className="muted small"><span className="loop-note">Replaying a real guest flow on loop</span></p>
        </Reveal>
        <Reveal delay={140}>
          <div className="phone" aria-hidden>
            <div className="phone-top">
              <span className="wa-dot"><Icon name="chat" /></span>
              <div><b>FaceFinder Events</b><span>online</span></div>
            </div>
            <div className="chat">
              {chat.map((b, i) => (
                <div key={i} className={`bubble ${b.from}${b.kind === "link" ? " link" : ""}${b.kind === "photo" ? " photo" : ""}`} style={k(i)}>
                  {b.kind === "text" && b.body}
                  {b.kind === "photo" && b.src && <img src={b.src} alt="" />}
                  {b.kind === "typing" && <span className="typing"><i /><i /><i /></span>}
                  {b.kind === "link" && <><span className="fake-link">{b.link}</span><small>{b.body}</small></>}
                </div>
              ))}
            </div>
          </div>
        </Reveal>
      </section>
      <section className="how">
        {steps.map(([step, title, text], i) => (
          <Reveal key={step} delay={i * 110}>
            <div className="step">Step {step}</div><h3>{title}</h3><p className="muted small">{text}</p>
          </Reveal>
        ))}
      </section>
      <Reveal>
        <footer className="footer">
          <span className="brand" style={{ fontSize: 15 }}><span className="brand-mark"><Icon name="camera" /></span> FaceFinder</span>
          <p className="muted small">Face search only — never biometric identity. Each event is isolated, selfies are deleted within minutes, and every gallery link expires on its own.</p>
          <a href="/login" className="muted small">Sign in to your studio</a>
          <span className="muted small">Demo photos via Unsplash</span>
        </footer>
      </Reveal>
    </div>
  );
}
