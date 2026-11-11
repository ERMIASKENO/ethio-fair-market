'use client';

import { useRouter } from 'next/navigation';
import { LANG_COOKIE, type Lang } from '@/lib/i18n';

export function LangToggle({ lang }: { lang: Lang }) {
  const router = useRouter();
  const next: Lang = lang === 'en' ? 'am' : 'en';
  return (
    <button
      type="button"
      className="btn ghost"
      aria-label="Change language"
      onClick={() => {
        document.cookie = `${LANG_COOKIE}=${next}; path=/; max-age=31536000; samesite=lax`;
        router.refresh();
      }}
    >
      {next === 'am' ? 'አማርኛ' : 'English'}
    </button>
  );
}
