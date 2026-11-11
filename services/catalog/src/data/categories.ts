// Category tree of ethio-fair-market. Names in English (en) and Amharic (am).
// Slugs are stable identifiers stored with every listing ("vehicles/cars").

export type AttributeType = 'text' | 'number' | 'select' | 'boolean';

export interface Attribute {
  key: string;
  type: AttributeType;
  label: { en: string; am: string };
  required?: boolean;
  unit?: string;
  options?: string[];
  min?: number;
  max?: number;
}

export interface CategoryNode {
  slug: string; // path segment, unique among siblings
  name: { en: string; am: string };
  icon?: string;
  attributes?: Attribute[];
  children?: CategoryNode[];
}

const year: Attribute = { key: 'year', type: 'number', label: { en: 'Year', am: 'ዓመት' }, min: 1960, max: 2030 };
const brand = (options: string[]): Attribute => ({ key: 'brand', type: 'select', label: { en: 'Brand', am: 'ብራንድ' }, options });
const sizeM2: Attribute = { key: 'size_m2', type: 'number', label: { en: 'Size', am: 'ስፋት' }, unit: 'm²', min: 1 };

export const CATEGORIES: CategoryNode[] = [
  {
    slug: 'vehicles', icon: 'car', name: { en: 'Vehicles', am: 'ተሽከርካሪዎች' },
    children: [
      {
        slug: 'cars', name: { en: 'Cars', am: 'መኪናዎች' },
        attributes: [
          { key: 'make', type: 'select', label: { en: 'Make', am: 'አምራች' }, required: true,
            options: ['Toyota', 'Hyundai', 'Suzuki', 'Nissan', 'Volkswagen', 'Mitsubishi', 'Isuzu', 'BYD', 'Other'] },
          { key: 'model', type: 'text', label: { en: 'Model', am: 'ሞዴል' } },
          { ...year, required: true },
          { key: 'mileage_km', type: 'number', label: { en: 'Mileage', am: 'የተጓዘው ርቀት' }, unit: 'km', min: 0 },
          { key: 'fuel', type: 'select', label: { en: 'Fuel', am: 'ነዳጅ' }, options: ['petrol', 'diesel', 'electric', 'hybrid'] },
          { key: 'transmission', type: 'select', label: { en: 'Transmission', am: 'ማርሽ' }, options: ['manual', 'automatic'] },
        ],
      },
      { slug: 'motorcycles', name: { en: 'Motorcycles & Bajaj', am: 'ሞተር ሳይክሎች እና ባጃጅ' },
        attributes: [{ key: 'make', type: 'text', label: { en: 'Make', am: 'አምራች' } }, year] },
      { slug: 'trucks', name: { en: 'Trucks & Buses', am: 'የጭነት መኪናዎች እና አውቶቡሶች' }, attributes: [year] },
      { slug: 'spare-parts', name: { en: 'Spare parts', am: 'መለዋወጫዎች' } },
    ],
  },
  {
    slug: 'property', icon: 'home', name: { en: 'Property', am: 'ቤት እና ንብረት' },
    children: [
      {
        slug: 'apartments-rent', name: { en: 'Apartments for rent', am: 'የሚከራዩ አፓርታማዎች' },
        attributes: [
          { key: 'bedrooms', type: 'number', label: { en: 'Bedrooms', am: 'መኝታ ክፍሎች' }, required: true, min: 0, max: 20 },
          { key: 'bathrooms', type: 'number', label: { en: 'Bathrooms', am: 'መታጠቢያ ቤቶች' }, min: 0, max: 20 },
          sizeM2,
          { key: 'furnished', type: 'boolean', label: { en: 'Furnished', am: 'የተሟላ የቤት ዕቃ' } },
        ],
      },
      { slug: 'houses-sale', name: { en: 'Houses for sale', am: 'የሚሸጡ ቤቶች' },
        attributes: [{ key: 'bedrooms', type: 'number', label: { en: 'Bedrooms', am: 'መኝታ ክፍሎች' }, min: 0, max: 30 }, sizeM2] },
      { slug: 'land', name: { en: 'Land & plots', am: 'መሬት' },
        attributes: [{ ...sizeM2, required: true }, { key: 'title_deed', type: 'boolean', label: { en: 'Title deed (ካርታ)', am: 'ካርታ አለው' } }] },
      { slug: 'commercial', name: { en: 'Shops & offices', am: 'ሱቆች እና ቢሮዎች' }, attributes: [sizeM2] },
    ],
  },
  {
    slug: 'electronics', icon: 'phone', name: { en: 'Electronics', am: 'ኤሌክትሮኒክስ' },
    children: [
      { slug: 'phones', name: { en: 'Mobile phones', am: 'ሞባይል ስልኮች' },
        attributes: [brand(['Samsung', 'Apple', 'Tecno', 'Infinix', 'Xiaomi', 'Nokia', 'Other']),
          { key: 'storage_gb', type: 'number', label: { en: 'Storage', am: 'ማከማቻ' }, unit: 'GB' }] },
      { slug: 'laptops', name: { en: 'Laptops & computers', am: 'ላፕቶፖች እና ኮምፒውተሮች' },
        attributes: [brand(['HP', 'Dell', 'Lenovo', 'Apple', 'Acer', 'Asus', 'Other']),
          { key: 'ram_gb', type: 'number', label: { en: 'RAM', am: 'ራም' }, unit: 'GB' },
          { key: 'storage_gb', type: 'number', label: { en: 'Storage', am: 'ማከማቻ' }, unit: 'GB' }] },
      { slug: 'tv-audio', name: { en: 'TV & audio', am: 'ቴሌቪዥን እና ድምጽ' } },
      { slug: 'solar', name: { en: 'Solar & power', am: 'የፀሐይ ኃይል እና ጀነሬተር' },
        attributes: [{ key: 'watts', type: 'number', label: { en: 'Power', am: 'ኃይል' }, unit: 'W' }] },
    ],
  },
  {
    slug: 'home', icon: 'sofa', name: { en: 'Home & furniture', am: 'የቤት ዕቃዎች' },
    children: [
      { slug: 'furniture', name: { en: 'Furniture', am: 'የቤት ዕቃዎች' } },
      { slug: 'kitchen', name: { en: 'Kitchen & coffee ceremony', am: 'የወጥ ቤት እና የቡና ዕቃዎች' } },
      { slug: 'appliances', name: { en: 'Appliances', am: 'የቤት መሣሪያዎች' } },
    ],
  },
  {
    slug: 'fashion', icon: 'shirt', name: { en: 'Fashion', am: 'ፋሽን' },
    children: [
      { slug: 'women', name: { en: 'Women', am: 'የሴቶች' }, attributes: [{ key: 'size', type: 'select', label: { en: 'Size', am: 'መጠን' }, options: ['XS', 'S', 'M', 'L', 'XL', 'XXL'] }] },
      { slug: 'men', name: { en: 'Men', am: 'የወንዶች' }, attributes: [{ key: 'size', type: 'select', label: { en: 'Size', am: 'መጠን' }, options: ['XS', 'S', 'M', 'L', 'XL', 'XXL'] }] },
      { slug: 'traditional', name: { en: 'Traditional clothes', am: 'ባህላዊ አልባሳት' } },
      { slug: 'shoes', name: { en: 'Shoes', am: 'ጫማዎች' }, attributes: [{ key: 'shoe_size', type: 'number', label: { en: 'Size (EU)', am: 'መጠን' }, min: 16, max: 50 }] },
    ],
  },
  {
    slug: 'jobs', icon: 'briefcase', name: { en: 'Jobs', am: 'ሥራ' },
    children: ['finance:Finance & accounting:ፋይናንስ እና ሂሳብ', 'it:IT & software:አይቲ እና ሶፍትዌር', 'sales:Sales & marketing:ሽያጭ እና ማርኬቲንግ',
      'construction:Construction:ግንባታ', 'hospitality:Hotel & restaurant:ሆቴል እና ሬስቶራንት'].map((s) => {
      const [slug, en, am] = s.split(':');
      return {
        slug, name: { en, am },
        attributes: [
          { key: 'job_type', type: 'select', label: { en: 'Job type', am: 'የሥራ ዓይነት' }, options: ['full_time', 'part_time', 'contract', 'internship'] },
          { key: 'experience_years', type: 'number', label: { en: 'Experience', am: 'የሥራ ልምድ' }, unit: 'years', min: 0, max: 50 },
        ],
      } as CategoryNode;
    }),
  },
  {
    slug: 'services', icon: 'tools', name: { en: 'Services', am: 'አገልግሎቶች' },
    children: [
      { slug: 'repair', name: { en: 'Repair', am: 'ጥገና' } },
      { slug: 'tutoring', name: { en: 'Tutoring & lessons', am: 'ትምህርት እና ስልጠና' } },
      { slug: 'events', name: { en: 'Events & weddings', am: 'ዝግጅቶች እና ሠርግ' } },
      { slug: 'transport', name: { en: 'Transport & moving', am: 'ትራንስፖርት እና ዕቃ ማጓጓዝ' } },
    ],
  },
  {
    slug: 'agriculture', icon: 'wheat', name: { en: 'Agriculture', am: 'ግብርና' },
    children: [
      { slug: 'grains', name: { en: 'Grains & teff', am: 'እህል እና ጤፍ' },
        attributes: [{ key: 'unit', type: 'select', label: { en: 'Unit', am: 'መለኪያ' }, options: ['kg', 'quintal'] },
          { key: 'quantity', type: 'number', label: { en: 'Quantity', am: 'መጠን' }, min: 0 }] },
      { slug: 'livestock', name: { en: 'Livestock', am: 'የቤት እንስሳት' } },
      { slug: 'coffee', name: { en: 'Coffee', am: 'ቡና' } },
      { slug: 'equipment', name: { en: 'Farm equipment', am: 'የእርሻ መሣሪያዎች' } },
    ],
  },
  {
    slug: 'sports', icon: 'ball', name: { en: 'Sports & hobbies', am: 'ስፖርት' },
    children: [
      { slug: 'bicycles', name: { en: 'Bicycles', am: 'ብስክሌቶች' } },
      { slug: 'fitness', name: { en: 'Fitness', am: 'የአካል ብቃት' } },
    ],
  },
  {
    slug: 'kids', icon: 'baby', name: { en: 'Kids & baby', am: 'ልጆች' },
    children: [
      { slug: 'toys', name: { en: 'Toys', am: 'መጫወቻዎች' } },
      { slug: 'baby', name: { en: 'Baby items', am: 'የሕፃናት ዕቃዎች' } },
    ],
  },
];
