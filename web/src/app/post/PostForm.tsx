'use client';

import { useRouter } from 'next/navigation';
import { useEffect, useMemo, useState } from 'react';
import { DICTS, pick, type Lang } from '@/lib/i18n';
import type { Attribute, Category, CategoryTreeNode, Region } from '@/lib/types';

export function PostForm({ lang, tree, regions }: { lang: Lang; tree: CategoryTreeNode[]; regions: Region[] }) {
  const t = DICTS[lang];
  const router = useRouter();
  const [category, setCategory] = useState('');
  const [attributes, setAttributes] = useState<Attribute[]>([]);
  const [attrValues, setAttrValues] = useState<Record<string, string | boolean>>({});
  const [region, setRegion] = useState('');
  const [city, setCity] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);

  const cities = useMemo(() => regions.find((r) => r.code === region)?.cities ?? [], [regions, region]);
  const subcities = useMemo(() => cities.find((c) => c.name.en === city)?.subcities ?? [], [cities, city]);

  // load the attribute schema of the chosen category (e.g. cars: make, year, mileage)
  useEffect(() => {
    setAttrValues({});
    if (!category) return setAttributes([]);
    const path = category.split('/').map(encodeURIComponent).join('/');
    fetch(`/v1/categories/${path}`)
      .then((r) => (r.ok ? r.json() : null))
      .then((c: Category | null) => setAttributes(c?.attributes ?? []))
      .catch(() => setAttributes([]));
  }, [category]);

  async function submit(form: HTMLFormElement) {
    const f = new FormData(form);
    const attrs: Record<string, unknown> = {};
    for (const a of attributes) {
      const v = attrValues[a.key];
      if (v === undefined || v === '') continue;
      attrs[a.key] = a.type === 'number' ? Number(v) : v;
    }
    const body = {
      title: String(f.get('title') ?? ''),
      description: String(f.get('description') ?? ''),
      price_etb: Number(f.get('price_etb')),
      negotiable: f.get('negotiable') === 'on',
      category,
      region,
      city,
      subcity: String(f.get('subcity') ?? ''),
      condition: String(f.get('condition') ?? 'used'),
      attributes: attrs,
      show_phone: f.get('show_phone') === 'on',
    };
    setBusy(true);
    setError(null);
    setFieldErrors({});
    try {
      const res = await fetch('/v1/listings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'same-origin',
        body: JSON.stringify(body),
      });
      const data = await res.json().catch(() => ({}));
      if (res.status === 401) return router.push('/login?next=/post');
      if (!res.ok) {
        setFieldErrors(data?.error?.fields ?? {});
        throw new Error(data?.error?.message ?? t.error);
      }
      router.push(`/listing/${data.id}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : t.error);
    } finally {
      setBusy(false);
    }
  }

  const fe = (k: string) => fieldErrors[k] && <span className="field-error">{fieldErrors[k]}</span>;

  return (
    <form className="stack" onSubmit={(e) => { e.preventDefault(); void submit(e.currentTarget); }}>
      <label>{t.category}
        <select required value={category} onChange={(e) => setCategory(e.target.value)}>
          <option value="">{t.chooseCategory}</option>
          {tree.map((c) => (
            <optgroup key={c.slug} label={pick(c.name, lang)}>
              {c.children.map((s) => <option key={s.slug} value={s.slug}>{pick(s.name, lang)}</option>)}
            </optgroup>
          ))}
        </select>{fe('category')}
      </label>

      <label>{t.title}<input name="title" required minLength={5} maxLength={120} />{fe('title')}</label>

      <div className="row">
        <label>{t.price}<input name="price_etb" type="number" min={0} step="0.01" required />{fe('price_etb')}</label>
        <label className="check"><input type="checkbox" name="negotiable" /> {t.negotiable}</label>
      </div>

      <label>{t.condition}
        <select name="condition" defaultValue="used">
          <option value="used">{t.used}</option>
          <option value="new">{t.new}</option>
          <option value="not_applicable">{t.not_applicable}</option>
        </select>
      </label>

      {attributes.map((a) => (
        <label key={a.key} className={a.type === 'boolean' ? 'check' : undefined}>
          {a.type === 'boolean' ? (
            <><input type="checkbox" checked={Boolean(attrValues[a.key])} onChange={(e) => setAttrValues({ ...attrValues, [a.key]: e.target.checked })} /> {pick(a.label, lang)}</>
          ) : (
            <>
              {pick(a.label, lang)}{a.unit ? ` (${a.unit})` : ''}{!a.required && <span className="muted small"> – {t.optional}</span>}
              {a.type === 'select' ? (
                <select required={a.required} value={String(attrValues[a.key] ?? '')} onChange={(e) => setAttrValues({ ...attrValues, [a.key]: e.target.value })}>
                  <option value="" />
                  {a.options?.map((o) => <option key={o} value={o}>{o}</option>)}
                </select>
              ) : (
                <input type={a.type === 'number' ? 'number' : 'text'} min={a.min} max={a.max} required={a.required}
                  value={String(attrValues[a.key] ?? '')} onChange={(e) => setAttrValues({ ...attrValues, [a.key]: e.target.value })} />
              )}
            </>
          )}
        </label>
      ))}

      <div className="row">
        <label>{t.region}
          <select required value={region} onChange={(e) => { setRegion(e.target.value); setCity(''); }}>
            <option value="">{t.chooseRegion}</option>
            {regions.map((r) => <option key={r.code} value={r.code}>{pick(r.name, lang)}</option>)}
          </select>
        </label>
        <label>{t.city}
          <select required value={city} onChange={(e) => setCity(e.target.value)} disabled={!region}>
            <option value="">{t.chooseCity}</option>
            {cities.map((c) => <option key={c.slug} value={c.name.en}>{pick(c.name, lang)}</option>)}
          </select>{fe('city')}
        </label>
      </div>
      {subcities.length > 0 && (
        <label>{t.subcity}
          <select name="subcity" defaultValue="">
            <option value="" />
            {subcities.map((s) => <option key={s.name.en} value={s.name.en}>{pick(s.name, lang)}</option>)}
          </select>
        </label>
      )}

      <label>{t.description}<textarea name="description" rows={6} maxLength={5000} />{fe('description')}</label>
      <label className="check"><input type="checkbox" name="show_phone" defaultChecked /> {t.showPhone}</label>

      {error && <p className="alert">{error}</p>}
      <button className="btn primary" disabled={busy}>{t.publish}</button>
    </form>
  );
}
