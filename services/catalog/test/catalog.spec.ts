import type { INestApplication } from '@nestjs/common';
import request from 'supertest';
import { createApp } from '../src/main';
import { CatalogService } from '../src/catalog.service';

describe('catalog', () => {
  let app: INestApplication;

  beforeAll(async () => {
    app = await createApp();
    await app.init();
  });
  afterAll(async () => app.close());

  it('returns the category tree with Amharic names', async () => {
    const res = await request(app.getHttpServer()).get('/v1/categories').expect(200);
    const vehicles = res.body.items.find((c: { slug: string }) => c.slug === 'vehicles');
    expect(vehicles.name.am).toBe('ተሽከርካሪዎች');
    expect(vehicles.children.map((c: { slug: string }) => c.slug)).toContain('vehicles/cars');
  });

  it('returns one nested category with breadcrumbs and attributes', async () => {
    const res = await request(app.getHttpServer()).get('/v1/categories/vehicles/cars').expect(200);
    expect(res.body.parent).toBe('vehicles');
    expect(res.body.breadcrumbs).toHaveLength(2);
    expect(res.body.attributes.map((a: { key: string }) => a.key)).toEqual(expect.arrayContaining(['make', 'year']));
  });

  it('returns a top-level category', async () => {
    await request(app.getHttpServer()).get('/v1/categories/jobs').expect(200);
  });

  it('returns 404 for unknown categories', async () => {
    const res = await request(app.getHttpServer()).get('/v1/categories/vehicles/boats').expect(404);
    expect(res.body.error.code).toBe('not_found');
  });

  it('exposes health and metrics', async () => {
    await request(app.getHttpServer()).get('/healthz').expect(200);
    await request(app.getHttpServer()).get('/readyz').expect(200);
    const m = await request(app.getHttpServer()).get('/metrics').expect(200);
    expect(m.text).toContain('http_requests_total');
  });

  it('knows every category used by the demo listings', () => {
    const svc = new CatalogService();
    for (const slug of ['vehicles/cars', 'property/apartments-rent', 'electronics/phones', 'vehicles/motorcycles', 'home/kitchen',
      'electronics/laptops', 'fashion/women', 'property/land', 'jobs/finance', 'electronics/solar', 'sports/bicycles', 'agriculture/grains']) {
      expect(svc.find(slug)).toBeDefined();
    }
  });
});
