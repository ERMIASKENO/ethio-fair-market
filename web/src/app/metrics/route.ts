import { Registry, collectDefaultMetrics } from 'prom-client';

export const dynamic = 'force-dynamic';
export const runtime = 'nodejs';

// one registry per process, also across hot reloads in development
const g = globalThis as unknown as { __efmRegistry?: Registry };
if (!g.__efmRegistry) {
  g.__efmRegistry = new Registry();
  g.__efmRegistry.setDefaultLabels({ service: 'web' });
  collectDefaultMetrics({ register: g.__efmRegistry });
}

export async function GET() {
  const body = await g.__efmRegistry!.metrics();
  return new Response(body, { headers: { 'Content-Type': g.__efmRegistry!.contentType, 'Cache-Control': 'no-store' } });
}
