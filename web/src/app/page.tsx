import Link from 'next/link';
import { ListingCard } from '@/components/ListingCard';
import { SearchBar } from '@/components/SearchBar';
import { categoryEmoji } from '@/components/icons';
import { getCategoryTree, getLang, searchListings } from '@/lib/api';
import { DICTS, pick } from '@/lib/i18n';

export const dynamic = 'force-dynamic'; // data comes from the services at request time (cached by fetch)

export default async function HomePage() {
  const lang = await getLang();
  const t = DICTS[lang];
  const [tree, latest] = await Promise.all([
    getCategoryTree().catch(() => []),
    searchListings({ page_size: '12' }).catch(() => null),
  ]);

  return (
    <>
      <section className="hero">
        <h1>{t.tagline}</h1>
        <SearchBar t={t} />
      </section>

      <section>
        <h2>{t.categories}</h2>
        <div className="cat-grid">
          {tree.map((c) => (
            <Link key={c.slug} href={`/search?category=${encodeURIComponent(c.slug)}`} className="cat">
              <span className="cat-icon" aria-hidden>{categoryEmoji(c.slug)}</span>
              <span>{pick(c.name, lang)}</span>
            </Link>
          ))}
        </div>
      </section>

      <section>
        <h2>{t.latest}</h2>
        {latest === null ? (
          <p className="alert">{t.error}</p>
        ) : (
          <div className="grid">
            {latest.items.map((l) => <ListingCard key={l.id} l={l} lang={lang} />)}
          </div>
        )}
      </section>
    </>
  );
}
