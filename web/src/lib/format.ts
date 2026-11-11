import type { Lang } from './i18n';

/** 1850000 -> "ETB 1,850,000" (en) / "1,850,000 ብር" (am). Santim shown only when present. */
export function formatPrice(value: number, lang: Lang): string {
  const hasSantim = Math.round(value * 100) % 100 !== 0;
  const n = new Intl.NumberFormat('en-US', {
    minimumFractionDigits: hasSantim ? 2 : 0,
    maximumFractionDigits: 2,
  }).format(value);
  return lang === 'am' ? `${n} ብር` : `ETB ${n}`;
}

const UNITS: [Intl.RelativeTimeFormatUnit, number][] = [
  ['year', 365 * 24 * 3600],
  ['month', 30 * 24 * 3600],
  ['day', 24 * 3600],
  ['hour', 3600],
  ['minute', 60],
];

/** "3 days ago" / "ከ3 ቀናት በፊት" */
export function timeAgo(iso: string, lang: Lang, now: Date = new Date()): string {
  const seconds = Math.round((new Date(iso).getTime() - now.getTime()) / 1000);
  const rtf = new Intl.RelativeTimeFormat(lang === 'am' ? 'am' : 'en', { numeric: 'auto' });
  for (const [unit, size] of UNITS) {
    if (Math.abs(seconds) >= size) return rtf.format(Math.round(seconds / size), unit);
  }
  return rtf.format(0, 'minute');
}

/** Reads the JWT payload for display only (name/phone in the header). Authorization is always done by the services. */
export function decodeToken(token: string | undefined): { sub: string; phone: string; exp: number } | null {
  if (!token) return null;
  const parts = token.split('.');
  if (parts.length !== 3) return null;
  try {
    const json = Buffer.from(parts[1].replace(/-/g, '+').replace(/_/g, '/'), 'base64').toString('utf8');
    const p = JSON.parse(json);
    if (typeof p.exp !== 'number' || p.exp * 1000 < Date.now()) return null;
    return { sub: String(p.sub), phone: String(p.phone ?? ''), exp: p.exp };
  } catch {
    return null;
  }
}

/** Builds a query string from an object, skipping empty values. */
export function qs(params: Record<string, string | number | undefined | null>): string {
  const u = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && String(v) !== '') u.set(k, String(v));
  }
  const s = u.toString();
  return s ? `?${s}` : '';
}
