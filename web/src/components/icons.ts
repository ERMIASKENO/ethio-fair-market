const EMOJI: Record<string, string> = {
  vehicles: '🚗', property: '🏠', electronics: '📱', home: '🛋️', fashion: '👗', jobs: '💼',
  services: '🛠️', agriculture: '🌾', sports: '⚽', kids: '🧸',
};

export function categoryEmoji(slug: string): string {
  return EMOJI[slug.split('/')[0]] ?? '📦';
}
