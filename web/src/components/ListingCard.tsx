import Link from 'next/link';
import { formatPrice, timeAgo } from '@/lib/format';
import { DICTS, type Lang } from '@/lib/i18n';
import type { Listing } from '@/lib/types';
import { categoryEmoji } from './icons';

export function ListingCard({ l, lang }: { l: Listing; lang: Lang }) {
  const t = DICTS[lang];
  const top = l.category.split('/')[0];
  return (
    <Link href={`/listing/${l.id}`} className="card">
      <div className="card-img">
        {l.image_urls[0] ? <img src={l.image_urls[0]} alt="" loading="lazy" /> : <span aria-hidden>{categoryEmoji(top)}</span>}
        {l.featured && <span className="badge featured">{t.featured}</span>}
      </div>
      <div className="card-body">
        <h3 className="card-title">{l.title}</h3>
        <p className="price">
          {formatPrice(l.price_etb, lang)}
          {l.negotiable && <span className="muted small"> · {t.negotiable}</span>}
        </p>
        <p className="muted small">
          {l.subcity ? `${l.subcity}, ` : ''}{l.city} · {timeAgo(l.created_at, lang)}
        </p>
      </div>
    </Link>
  );
}
