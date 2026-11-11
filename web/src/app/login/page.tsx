import type { Metadata } from 'next';
import { getLang } from '@/lib/api';
import { DICTS } from '@/lib/i18n';
import { LoginForm } from './LoginForm';

export const metadata: Metadata = { title: 'Log in' };
type SP = Promise<Record<string, string | string[] | undefined>>;

export default async function LoginPage({ searchParams }: { searchParams: SP }) {
  const t = DICTS[await getLang()];
  const next = (await searchParams).next;
  // only allow local redirects (no open redirect to other sites)
  const target = typeof next === 'string' && next.startsWith('/') && !next.startsWith('//') ? next : '/';
  return (
    <div className="narrow">
      <h1>{t.login}</h1>
      <LoginForm t={t} next={target} />
    </div>
  );
}
