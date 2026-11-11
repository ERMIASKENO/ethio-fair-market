import { Controller, Get, Header, Res } from '@nestjs/common';
import type { Response } from 'express';
import { register } from '../metrics';
import { CatalogService } from '../catalog.service';
import { lifecycle } from '../lifecycle';

@Controller()
export class HealthController {
  constructor(private readonly catalog: CatalogService) {}

  /** Liveness: the process is running. */
  @Get('healthz')
  healthz() {
    return { status: 'ok' };
  }

  /** Readiness: data loaded and not shutting down. */
  @Get('readyz')
  readyz(@Res() res: Response) {
    const ready = !lifecycle.shuttingDown && this.catalog.count() > 0;
    res.status(ready ? 200 : 503).json({ status: ready ? 'ready' : 'not ready', categories: this.catalog.count() });
  }

  @Get('metrics')
  @Header('Content-Type', register.contentType)
  async metrics() {
    return register.metrics();
  }
}
