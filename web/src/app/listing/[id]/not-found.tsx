import Link from 'next/link';
import { getLang } from '@/lib/api';
import { DICTS } from '@/lib/i18n';

export default async function NotFound() {
  const t = DICTS[await getLang()];
  return (
    <div className="empty">
      <p>{t.notFound}</p>
      <Link className="btn primary" href="/">{t.home}</Link>
    </div>
  );
}
