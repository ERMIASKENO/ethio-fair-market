import { Controller, Get, Header, NotFoundException, Param } from '@nestjs/common';
import { CatalogService } from './catalog.service';

@Controller('v1/categories')
export class CatalogController {
  constructor(private readonly catalog: CatalogService) {}

  /** Full tree for menus and the "post an ad" form. */
  @Get()
  @Header('Cache-Control', 'public, max-age=300')
  tree() {
    return { items: this.catalog.getTree() };
  }

  /** One category incl. breadcrumbs and attribute schema: /v1/categories/vehicles/cars */
  @Get('*slug')
  @Header('Cache-Control', 'public, max-age=300')
  one(@Param('slug') slug: string | string[]) {
    const path = Array.isArray(slug) ? slug.join('/') : slug;
    const category = this.catalog.find(path);
    if (!category) throw new NotFoundException({ error: { code: 'not_found', message: `unknown category ${path}` } });
    return category;
  }
}
