import assert from 'node:assert/strict';
import { test } from 'node:test';
import { decodeToken, formatPrice, qs, timeAgo } from '../src/lib/format.ts';
import { DICTS, normalizeLang, pick } from '../src/lib/i18n.ts';

test('formats birr prices in English and Amharic', () => {
  assert.equal(formatPrice(1850000, 'en'), 'ETB 1,850,000');
  assert.equal(formatPrice(1850000, 'am'), '1,850,000 ብር');
  assert.equal(formatPrice(99.5, 'en'), 'ETB 99.50');
});

test('relative time', () => {
  const now = new Date('2026-11-11T12:00:00Z');
  assert.equal(timeAgo('2026-11-08T12:00:00Z', 'en', now), '3 days ago');
});

test('query string skips empty values', () => {
  assert.equal(qs({ q: 'toyota', page: 2, city: '', region: undefined }), '?q=toyota&page=2');
  assert.equal(qs({}), '');
});

test('decodeToken reads valid and rejects expired tokens', () => {
  const enc = (o: object) => Buffer.from(JSON.stringify(o)).toString('base64url');
  const future = Math.floor(Date.now() / 1000) + 600;
  assert.deepEqual(decodeToken(`${enc({ alg: 'EdDSA' })}.${enc({ sub: 'u1', phone: '+251911000000', exp: future })}.sig`),
    { sub: 'u1', phone: '+251911000000', exp: future });
  assert.equal(decodeToken(`${enc({})}.${enc({ sub: 'u1', exp: 1 })}.sig`), null);
  assert.equal(decodeToken('garbage'), null);
});

test('every English text has an Amharic translation', () => {
  for (const key of Object.keys(DICTS.en)) {
    assert.ok((DICTS.am as Record<string, string>)[key], `missing am translation for ${key}`);
  }
  assert.equal(normalizeLang('am'), 'am');
  assert.equal(normalizeLang('fr'), 'en');
  assert.equal(pick({ en: 'Cars', am: 'መኪናዎች' }, 'am'), 'መኪናዎች');
});
