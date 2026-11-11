import { Injectable } from '@nestjs/common';
import { Attribute, CATEGORIES, CategoryNode } from './data/categories';

export interface CategoryView {
  slug: string; // full path, e.g. "vehicles/cars"
  name: { en: string; am: string };
  icon?: string;
  parent: string | null;
  breadcrumbs: { slug: string; name: { en: string; am: string } }[];
  attributes: Attribute[];
  children: { slug: string; name: { en: string; am: string } }[];
}

export interface TreeNode {
  slug: string;
  name: { en: string; am: string };
  icon?: string;
  children: TreeNode[];
}

@Injectable()
export class CatalogService {
  private readonly index = new Map<string, CategoryView>();
  private readonly tree: TreeNode[];

  constructor() {
    this.tree = CATEGORIES.map((c) => this.build(c, null, []));
  }

  private build(node: CategoryNode, parent: CategoryView | null, crumbs: CategoryView['breadcrumbs']): TreeNode {
    const slug = parent ? `${parent.slug}/${node.slug}` : node.slug;
    if (this.index.has(slug)) throw new Error(`duplicate category slug ${slug}`);
    const view: CategoryView = {
      slug,
      name: node.name,
      icon: node.icon ?? parent?.icon,
      parent: parent?.slug ?? null,
      breadcrumbs: [...crumbs, { slug, name: node.name }],
      // children inherit nothing; attributes are defined on the leaf
      attributes: node.attributes ?? [],
      children: [],
    };
    this.index.set(slug, view);
    const children = (node.children ?? []).map((c) => this.build(c, view, view.breadcrumbs));
    view.children = children.map((c) => ({ slug: c.slug, name: c.name }));
    return { slug, name: node.name, icon: view.icon, children };
  }

  getTree(): TreeNode[] {
    return this.tree;
  }

  find(slug: string): CategoryView | undefined {
    return this.index.get(slug.replace(/^\/+|\/+$/g, '').toLowerCase());
  }

  count(): number {
    return this.index.size;
  }
}
