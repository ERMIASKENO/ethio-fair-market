export type Name = { en: string; am: string };

export interface Listing {
  id: string;
  seller_id: string;
  title: string;
  description: string;
  price_etb: number;
  negotiable: boolean;
  category: string;
  region: string;
  city: string;
  subcity?: string;
  condition: 'new' | 'used' | 'not_applicable';
  attributes: Record<string, unknown>;
  image_urls: string[];
  contact_phone?: string;
  status: 'active' | 'sold' | 'expired' | 'removed';
  view_count: number;
  featured: boolean;
  created_at: string;
  updated_at: string;
  expires_at: string;
}

export interface ListingPage {
  items: Listing[];
  total: number;
  page: number;
  page_size: number;
}

export interface CategoryTreeNode {
  slug: string;
  name: Name;
  icon?: string;
  children: CategoryTreeNode[];
}

export interface Attribute {
  key: string;
  type: 'text' | 'number' | 'select' | 'boolean';
  label: Name;
  required?: boolean;
  unit?: string;
  options?: string[];
  min?: number;
  max?: number;
}

export interface Category {
  slug: string;
  name: Name;
  icon?: string;
  parent: string | null;
  breadcrumbs: { slug: string; name: Name }[];
  attributes: Attribute[];
  children: { slug: string; name: Name }[];
}

export interface City {
  slug: string;
  name: Name;
  subcities: { name: Name }[];
}

export interface Region {
  code: string;
  name: Name;
  kind: string;
  cities?: City[];
}
