import type { Dict } from '@/lib/i18n';

/** Plain HTML form (GET /search) – works without JavaScript, important on slow phones. */
export function SearchBar({ t, q = '', category = '' }: { t: Dict; q?: string; category?: string }) {
  return (
    <form action="/search" method="get" className="searchbar" role="search">
      <input name="q" defaultValue={q} placeholder={t.searchPlaceholder} aria-label={t.search} maxLength={100} />
      {category && <input type="hidden" name="category" value={category} />}
      <button type="submit" className="btn primary">{t.search}</button>
    </form>
  );
}
