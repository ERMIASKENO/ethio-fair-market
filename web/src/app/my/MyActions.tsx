'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import type { Dict } from '@/lib/i18n';

export function MyActions({ id, status, t }: { id: string; status: string; t: Dict }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const act = async (method: string, path: string) => {
    setBusy(true);
    const res = await fetch(path, { method, credentials: 'same-origin' });
    setBusy(false);
    if (res.ok) router.refresh();
    else alert(t.error);
  };
  return (
    <div className="row-actions">
      {status === 'active' && <button className="btn ghost" disabled={busy} onClick={() => act('POST', `/v1/listings/${id}/sold`)}>{t.markSold}</button>}
      <button className="btn danger" disabled={busy} onClick={() => { if (confirm(`${t.delete}?`)) void act('DELETE', `/v1/listings/${id}`); }}>{t.delete}</button>
    </div>
  );
}
