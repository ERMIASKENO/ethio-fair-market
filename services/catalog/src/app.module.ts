import { Module } from '@nestjs/common';
import { CatalogController } from './catalog.controller';
import { CatalogService } from './catalog.service';
import { HealthController } from './health/health.controller';

@Module({
  controllers: [HealthController, CatalogController],
  providers: [CatalogService],
})
export class AppModule {}
