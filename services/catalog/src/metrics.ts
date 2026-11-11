import type { NextFunction, Request, Response } from 'express';
import { Counter, Histogram, Registry, collectDefaultMetrics } from 'prom-client';

export const register = new Registry();
register.setDefaultLabels({ service: 'catalog' });
collectDefaultMetrics({ register });

const requests = new Counter({
  name: 'http_requests_total',
  help: 'HTTP requests handled.',
  labelNames: ['route', 'code'],
  registers: [register],
});
const duration = new Histogram({
  name: 'http_request_duration_seconds',
  help: 'HTTP request latency.',
  labelNames: ['route'],
  buckets: [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5],
  registers: [register],
});

const SKIP = new Set(['/healthz', '/readyz', '/metrics']);

/** Access log (one JSON line) + metrics. Route label = Express route pattern, not the raw URL. */
export function observability(req: Request, res: Response, next: NextFunction) {
  if (SKIP.has(req.path)) return next();
  const start = process.hrtime.bigint();
  const requestId = (req.headers['x-request-id'] as string) || Math.random().toString(16).slice(2, 14);
  res.setHeader('X-Request-Id', requestId);
  res.setHeader('X-Content-Type-Options', 'nosniff');
  res.on('finish', () => {
    const seconds = Number(process.hrtime.bigint() - start) / 1e9;
    const route = req.route?.path ? `${req.method} ${req.baseUrl}${req.route.path}` : 'unmatched';
    requests.inc({ route, code: String(res.statusCode) });
    duration.observe({ route }, seconds);
    process.stdout.write(
      JSON.stringify({
        time: new Date().toISOString(), level: 'INFO', msg: 'request', service: 'catalog', request_id: requestId,
        method: req.method, path: req.path, route, status: res.statusCode, duration_ms: Math.round(seconds * 1000),
      }) + '\n',
    );
  });
  next();
}
