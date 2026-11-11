import type { Metadata } from 'next';
import Link from 'next/link';
import { notFound } from 'next/navigation';
import { categoryEmoji } from '@/components/icons';
import { getCategory, getLang, getListing } from '@/lib/api';
import { formatPrice, timeAgo } from '@/lib/format';
import { DICTS, pick } from '@/lib/i18n';

export const dynamic = 'force-dynamic';
type Params = Promise<{ id: string }>;

export async function generateMetadata({ params }: { params: Params }): Promise<Metadata> {
  const l = await getListing((await params).id).catch(() => null);
  return l ? { title: l.title, description: l.description.slice(0, 150) } : { title: 'Not found' };
}

export default async function ListingPage({ params }: { params: Params }) {
  const { id } = await params;
  const lang = await getLang();
  const t = DICTS[lang];
  const l = await getListing(id);
  if (!l) notFound();
  const category = await getCategory(l.category).catch(() => null);
  const labels = new Map(category?.attributes.map((a) => [a.key, a]) ?? []);
  const attrs = Object.entries(l.attributes ?? {});

  return (
    <article className="detail">
      <nav className="crumbs muted small">
        <Link href="/">{t.home}</Link>
        {category?.breadcrumbs.map((b) => (
          <span key={b.slug}> › <Link href={`/search?category=${encodeURIComponent(b.slug)}`}>{pick(b.name, lang)}</Link></span>
        ))}
      </nav>

      <div className="detail-grid">
        <div>
          <div className="detail-img">
            {l.image_urls[0] ? <img src={l.image_urls[0]} alt={l.title} /> : <span aria-hidden>{categoryEmoji(l.category)}</span>}
          </div>
          <h1>{l.title}</h1>
          <p className="price big">
            {formatPrice(l.price_etb, lang)} {l.negotiable && <span className="badge">{t.negotiable}</span>}
          </p>
          <p className="muted">
            📍 {l.subcity ? `${l.subcity}, ` : ''}{l.city} · {t.posted} {timeAgo(l.created_at, lang)} · {l.view_count} {t.views}
          </p>
          {l.status !== 'active' && <p className="alert">{t[l.status as 'sold' | 'expired'] ?? l.status}</p>}

          {attrs.length > 0 && (
            <section>
              <h2>{t.details}</h2>
              <dl className="attrs">
                {attrs.map(([k, v]) => {
                  const a = labels.get(k);
                  return (
                    <div key={k}>
                      <dt>{a ? pick(a.label, lang) : k}</dt>
                      <dd>{typeof v === 'boolean' ? (v ? '✓' : '✗') : String(v)}{a?.unit ? ` ${a.unit}` : ''}</dd>
                    </div>
                  );
                })}
              </dl>
            </section>
          )}
          {l.description && (
            <section>
              <h2>{t.description}</h2>
              <p className="prewrap">{l.description}</p>
            </section>
          )}
        </div>

        <aside className="contact">
          <h2>{t.contactSeller}</h2>
          {l.contact_phone ? (
            <a className="btn primary block" href={`tel:${l.contact_phone}`}>📞 {t.callSeller} {l.contact_phone}</a>
          ) : (
            <p className="muted">{t.noPhone}</p>
          )}
        </aside>
      </div>
    </article>
  );
}
