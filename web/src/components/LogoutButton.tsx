'use client';

import { useRouter } from 'next/navigation';

export function LogoutButton({ label }: { label: string }) {
  const router = useRouter();
  return (
    <button
      type="button"
      className="btn ghost"
      onClick={async () => {
        await fetch('/v1/auth/logout', { method: 'POST', credentials: 'same-origin' });
        router.push('/');
        router.refresh();
      }}
    >
      {label}
    </button>
  );
}
