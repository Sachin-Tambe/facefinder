"use client";
import { usePathname } from "next/navigation";
import Icon from "@/components/icons";

const NAV = [
  { href: "/dashboard", label: "Dashboard", icon: "home" },
  { href: "/events", label: "Events", icon: "calendar" },
  { href: "/events/new", label: "New event", icon: "plus" },
] as const;
export default function StudioLayout({ children }: { children: React.ReactNode }) {
  const path = usePathname();
  function logout() {
    localStorage.removeItem("ff_token"); localStorage.removeItem("ff_org");
    fetch("/api/auth/sign-out", { method: "POST" }).catch(() => {});
    location.href = "/login";
  }
  return (
    <div className="shell">
      <aside className="sidebar">
        <a className="brand" href="/dashboard"><span className="brand-mark"><Icon name="camera" /></span> FaceFinder</a>
        <nav>
          {NAV.map(n => (
            <a key={n.href} href={n.href} className={path === n.href || (n.href !== "/dashboard" && path?.startsWith(n.href)) ? "active" : ""}>
              <Icon name={n.icon} />{n.label}
            </a>
          ))}
        </nav>
        <div className="sidebar-foot">
          <p className="muted small">One selfie on WhatsApp finds every photo a guest is in.</p>
          <button className="btn-ghost full" onClick={logout}><Icon name="logout" /> Sign out</button>
        </div>
      </aside>
      <div className="content">
        <header className="mobilebar">
          <a className="brand" href="/dashboard"><span className="brand-mark"><Icon name="camera" /></span> FaceFinder</a>
          <button className="btn-ghost" onClick={logout}>Sign out</button>
        </header>
        <main className="main">{children}</main>
      </div>
    </div>
  );
}
