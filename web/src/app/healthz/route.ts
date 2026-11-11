// Liveness/readiness for Kubernetes. The web tier has no own state; when the services are down
// the pages show an error message, so readiness does not depend on them (avoids cascading failures).
export const dynamic = 'force-dynamic';

export function GET() {
  return Response.json({ status: 'ok' }, { headers: { 'Cache-Control': 'no-store' } });
}
