// Single-stroke SVG icon set. No emoji in the interface.
import type { ReactNode, SVGProps } from "react";
const P: Record<string, ReactNode> = {
  home: <><path d="M3 10.5 12 3l9 7.5" /><path d="M5 9.5V21h5v-6h4v6h5V9.5" /></>,
  calendar: <><rect x="3" y="5" width="18" height="16" rx="2" /><path d="M3 10h18M8 3v4M16 3v4" /></>,
  plus: <><path d="M12 5v14M5 12h14" /></>,
  upload: <><path d="M12 16V4m0 0 4 4m-4-4-4 4" /><path d="M4 17v3h16v-3" /></>,
  qr: <><rect x="3" y="3" width="7" height="7" rx="1" /><rect x="14" y="3" width="7" height="7" rx="1" /><rect x="3" y="14" width="7" height="7" rx="1" /><path d="M14 14h3v3h-3zM20 14h1M14 20h1M18 18h3v3h-3z" /></>,
  clock: <><circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" /></>,
  check: <><path d="m4 12.5 5 5L20 6.5" /></>,
  alert: <><path d="M12 3 2.5 20h19L12 3Z" /><path d="M12 9v5m0 3v.5" /></>,
  x: <><path d="M6 6l12 12M18 6 6 18" /></>,
  download: <><path d="M12 4v12m0 0 4-4m-4 4-4-4" /><path d="M4 17v3h16v-3" /></>,
  image: <><rect x="3" y="4" width="18" height="16" rx="2" /><circle cx="9" cy="10" r="2" /><path d="m4 18 5-5 3 3 3-3 5 5" /></>,
  arrow: <><path d="M4 12h16m0 0-5-5m5 5-5 5" /></>,
  camera: <><path d="M4 8h3l2-2h6l2 2h3v11H4V8Z" /><circle cx="12" cy="13" r="3.5" /></>,
  logout: <><path d="M14 4H6v16h8" /><path d="M10 12h11m0 0-3-3m3 3-3 3" /></>,
  chat: <><path d="M4 5h16v11H9l-5 4V5Z" /></>,
  refresh: <><path d="M20 12a8 8 0 1 1-2.3-5.6" /><path d="M20 3v4h-4" /></>,
};
export default function Icon({ name, ...rest }: { name: keyof typeof P } & SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round" aria-hidden {...rest}>
      {P[name]}
    </svg>
  );
}
