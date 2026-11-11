import type { Metadata } from 'next';
import Link from 'next/link';
import { ListingCard } from '@/components/ListingCard';
import { SearchBar } from '@/components/SearchBar';
import { getCategoryTree, getLang, getRegions, searchListings } from '@/lib/api';
import { qs } from '@/lib/format';
import { DICTS, pick } from '@/lib/i18n';

export const dynamic = 'force-dynamic';
export const metadata: Metadata = { title: 'Search' };

type SP = Promise<Record<string, string | string[] | undefined>>;
const KEYS = ['q', 'category', 'region', 'city', 'min_price', 'max_price', 'condition', 'sort', 'page'] as const;

export default async function SearchPage({ searchParams }: { searchParams: SP }) {
  const raw = await searchParams;
  const p: Record<string, string> = {};
  for (const k of KEYS) {
    const v = raw[k];
    if (typeof v === 'string' && v.trim() !== '') p[k] = v.trim();
  }
  const lang = await getLang();
  const t = DICTS[lang];
  const [tree, regions, result] = await Promise.all([
    getCategoryTree().catch(() => []),
    getRegions().catch(() => []),
    searchListings({ ...p, page_size: '24' }).catch(() => null),
  ]);
  const page = Number(p.page ?? 1);
  const pages = result ? Math.max(1, Math.ceil(result.total / result.page_size)) : 1;

  return (
    <div className="search-layout">
      <aside className="filters">
        <form action="/search" method="get" className="stack">
          {p.q && <input type="hidden" name="q" value={p.q} />}
          <label>{t.category}
            <select name="category" defaultValue={p.category ?? ''}>
              <option value="">{t.allCategories}</option>
              {tree.map((c) => (
                <optgroup key={c.slug} label={pick(c.name, lang)}>
                  <option value={c.slug}>{pick(c.name, lang)} – {t.any}</option>
                  {c.children.map((s) => <option key={s.slug} value={s.slug}>{pick(s.name, lang)}</option>)}
                </optgroup>
              ))}
            </select>
          </label>
          <label>{t.region}
            <select name="region" defaultValue={p.region ?? ''}>
              <option value="">{t.allRegions}</option>
              {regions.map((r) => <option key={r.code} value={r.code}>{pick(r.name, lang)}</option>)}
            </select>
          </label>
          <label>{t.city}<input name="city" defaultValue={p.city ?? ''} maxLength={60} /></label>
          <div className="row">
            <label>{t.minPrice}<input name="min_price" type="number" min={0} defaultValue={p.min_price ?? ''} /></label>
            <label>{t.maxPrice}<input name="max_price" type="number" min={0} defaultValue={p.max_price ?? ''} /></label>
          </div>
          <label>{t.condition}
            <select name="condition" defaultValue={p.condition ?? ''}>
              <option value="">{t.any}</option>
              <option value="new">{t.new}</option>
              <option value="used">{t.used}</option>
            </select>
          </label>
          <label>{t.sort}
            <select name="sort" defaultValue={p.sort ?? ''}>
              <option value="">{t.newest}</option>
              <option value="price_asc">{t.price_asc}</option>
              <option value="price_desc">{t.price_desc}</option>
              {p.q && <option value="relevance">{t.relevance}</option>}
            </select>
          </label>
          <button className="btn primary" type="submit">{t.apply}</button>
        </form>
      </aside>

      <section>
        <SearchBar t={t} q={p.q} category={p.category} />
        {result === null ? (
          <p className="alert">{t.error}</p>
        ) : (
          <>
            <p className="muted">{result.total} {t.results}</p>
            {result.items.length === 0 ? (
              <p className="empty">{t.noResults}</p>
            ) : (
              <div className="grid">{result.items.map((l) => <ListingCard key={l.id} l={l} lang={lang} />)}</div>
            )}
            {pages > 1 && (
              <nav className="pager">
                {page > 1 && <Link className="btn ghost" href={`/search${qs({ ...p, page: page - 1 })}`}>{t.previous}</Link>}
                <span className="muted">{page} / {pages}</span>
                {page < pages && <Link className="btn ghost" href={`/search${qs({ ...p, page: page + 1 })}`}>{t.next}</Link>}
              </nav>
            )}
          </>
        )}
      </section>
    </div>
  );
}
