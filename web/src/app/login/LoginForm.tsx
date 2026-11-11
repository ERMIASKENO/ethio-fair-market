'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import type { Dict } from '@/lib/i18n';

async function post(url: string, body: unknown) {
  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'same-origin',
    body: JSON.stringify(body),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data?.error?.message ?? `HTTP ${res.status}`);
  return data;
}

export function LoginForm({ t, next }: { t: Dict; next: string }) {
  const router = useRouter();
  const [phone, setPhone] = useState('');
  const [code, setCode] = useState('');
  const [step, setStep] = useState<'phone' | 'code'>('phone');
  const [devCode, setDevCode] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function run(fn: () => Promise<void>) {
    setBusy(true);
    setError(null);
    try {
      await fn();
    } catch (e) {
      setError(e instanceof Error ? e.message : t.error);
    } finally {
      setBusy(false);
    }
  }

  return step === 'phone' ? (
    <form
      className="stack"
      onSubmit={(e) => {
        e.preventDefault();
        run(async () => {
          const r = await post('/v1/auth/otp/request', { phone });
          setDevCode(r.dev_code ?? null);
          setStep('code');
        });
      }}
    >
      <label>{t.phone}
        <input type="tel" inputMode="tel" autoComplete="tel" required value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="09xx xxx xxx" />
      </label>
      <p className="muted small">{t.phoneHint}</p>
      {error && <p className="alert">{error}</p>}
      <button className="btn primary" disabled={busy}>{t.sendCode}</button>
    </form>
  ) : (
    <form
      className="stack"
      onSubmit={(e) => {
        e.preventDefault();
        run(async () => {
          await post('/v1/auth/otp/verify', { phone, code });
          router.push(next);
          router.refresh();
        });
      }}
    >
      {devCode && <p className="notice">{t.devCode} <strong>{devCode}</strong></p>}
      <label>{t.code}
        <input inputMode="numeric" autoComplete="one-time-code" pattern="[0-9]{6}" maxLength={6} required value={code} onChange={(e) => setCode(e.target.value)} />
      </label>
      {error && <p className="alert">{error}</p>}
      <button className="btn primary" disabled={busy}>{t.verify}</button>
    </form>
  );
}
