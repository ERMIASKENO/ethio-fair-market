// Server-side data access. Pages run on the server and call the services over the cluster network
// (http://listings, http://catalog ...). The browser only talks to the same HTTPS origin.
import 'server-only';
import { cookies } from 'next/headers';
import { LANG_COOKIE, normalizeLang, type Lang } from './i18n';
import { decodeToken, qs } from './format';
import type { Category, CategoryTreeNode, Listing, ListingPage, Region } from './types';

const LISTINGS_URL = process.env.LISTINGS_URL ?? 'http://listings';
const CATALOG_URL = process.env.CATALOG_URL ?? 'http://catalog';
const LOCATIONS_URL = process.env.LOCATIONS_URL ?? 'http://locations';
const TOKEN_COOKIE = 'efm_token';

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

async function get<T>(url: string, opts: { revalidate?: number; token?: string } = {}): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' };
  if (opts.token) headers.Authorization = `Bearer ${opts.token}`;
  const res = await fetch(url, {
    headers,
    // public data is cached briefly; personal data never
    ...(opts.token ? { cache: 'no-store' as const } : { next: { revalidate: opts.revalidate ?? 30 } }),
    signal: AbortSignal.timeout(5000),
  });
  if (!res.ok) throw new ApiError(res.status, `${url} -> ${res.status}`);
  return (await res.json()) as T;
}

export async function getLang(): Promise<Lang> {
  return normalizeLang((await cookies()).get(LANG_COOKIE)?.value);
}

export async function getSession() {
  const token = (await cookies()).get(TOKEN_COOKIE)?.value;
  const claims = decodeToken(token);
  return claims && token ? { ...claims, token } : null;
}

export function searchListings(params: Record<string, string | undefined>) {
  return get<ListingPage>(`${LISTINGS_URL}/v1/listings${qs(params)}`, { revalidate: 15 });
}

export async function getListing(id: string): Promise<Listing | null> {
  try {
    return await get<Listing>(`${LISTINGS_URL}/v1/listings/${encodeURIComponent(id)}`, { revalidate: 10 });
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) return null;
    throw e;
  }
}

export function myListings(token: string) {
  return get<{ items: Listing[]; total: number }>(`${LISTINGS_URL}/v1/listings/mine`, { token });
}

export async function getCategoryTree(): Promise<CategoryTreeNode[]> {
  return (await get<{ items: CategoryTreeNode[] }>(`${CATALOG_URL}/v1/categories`, { revalidate: 300 })).items;
}

export async function getCategory(slug: string): Promise<Category | null> {
  try {
    return await get<Category>(`${CATALOG_URL}/v1/categories/${slug.split('/').map(encodeURIComponent).join('/')}`, { revalidate: 300 });
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) return null;
    throw e;
  }
}

export async function getRegions(withCities = false): Promise<Region[]> {
  return (await get<{ items: Region[] }>(`${LOCATIONS_URL}/v1/locations/regions${withCities ? '?include_cities=true' : ''}`, { revalidate: 3600 })).items;
}
