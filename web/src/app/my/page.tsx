import type { Metadata } from 'next';
import Link from 'next/link';
import { redirect } from 'next/navigation';
import { getLang, getSession, myListings } from '@/lib/api';
import { formatPrice, timeAgo } from '@/lib/format';
import { DICTS } from '@/lib/i18n';
import { MyActions } from './MyActions';

export const dynamic = 'force-dynamic';
export const metadata: Metadata = { title: 'My ads' };

export default async function MyPage() {
  const session = await getSession();
  if (!session) redirect('/login?next=/my');
  const lang = await getLang();
  const t = DICTS[lang];
  const { items } = await myListings(session.token);

  return (
    <div>
      <h1>{t.myAds}</h1>
      <p className="muted">{session.phone}</p>
      {items.length === 0 ? (
        <div className="empty"><p>{t.noAdsYet}</p><Link className="btn primary" href="/post">{t.postAd}</Link></div>
      ) : (
        <table className="table">
          <thead><tr><th>{t.title}</th><th>{t.price}</th><th>{t.status}</th><th>{t.posted}</th><th /></tr></thead>
          <tbody>
            {items.map((l) => (
              <tr key={l.id}>
                <td><Link href={`/listing/${l.id}`}>{l.title}</Link></td>
                <td>{formatPrice(l.price_etb, lang)}</td>
                <td><span className={`badge ${l.status}`}>{t[l.status as 'active' | 'sold' | 'expired'] ?? l.status}</span></td>
                <td className="muted small">{timeAgo(l.created_at, lang)}</td>
                <td><MyActions id={l.id} status={l.status} t={t} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
