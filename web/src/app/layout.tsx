import type { Metadata, Viewport } from 'next';
import Link from 'next/link';
import { LangToggle } from '@/components/LangToggle';
import { LogoutButton } from '@/components/LogoutButton';
import { getLang, getSession } from '@/lib/api';
import { DICTS } from '@/lib/i18n';
import './globals.css';

// DEMO_MODE=true (set in the GitOps repo): visible banner + keep search engines away
const DEMO = process.env.DEMO_MODE === 'true';

export const metadata: Metadata = {
  title: { default: 'Ethio Fair Market', template: '%s · Ethio Fair Market' },
  description: 'Buy and sell anything in Ethiopia – cars, houses, phones, jobs and more.',
  ...(DEMO ? { robots: { index: false, follow: false } } : {}),
};

export const viewport: Viewport = { width: 'device-width', initialScale: 1, themeColor: '#078930' };

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const lang = await getLang();
  const session = await getSession();
  const t = DICTS[lang];
  return (
    <html lang={lang}>
      <body>
        {DEMO && <div className="demo-banner" role="note">⚠️ {t.demoBanner}</div>}
        <header className="topbar">
          <div className="container topbar-inner">
            <Link href="/" className="logo" aria-label={t.home}>
              <span className="logo-mark">EFM</span>
              <span className="logo-text">{t.brand}</span>
            </Link>
            <nav className="nav">
              <LangToggle lang={lang} />
              {session ? (
                <>
                  <Link href="/my" className="btn ghost">{t.myAds}</Link>
                  <LogoutButton label={t.logout} />
                </>
              ) : (
                <Link href="/login" className="btn ghost">{t.login}</Link>
              )}
              <Link href="/post" className="btn primary">{t.postAd}</Link>
            </nav>
          </div>
        </header>
        <main className="container main">{children}</main>
        <footer className="footer">
          <div className="container muted small">{t.footer}</div>
        </footer>
      </body>
    </html>
  );
}
