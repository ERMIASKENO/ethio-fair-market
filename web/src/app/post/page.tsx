import type { Metadata } from 'next';
import { redirect } from 'next/navigation';
import { getCategoryTree, getLang, getRegions, getSession } from '@/lib/api';
import { DICTS } from '@/lib/i18n';
import { PostForm } from './PostForm';

export const dynamic = 'force-dynamic';
export const metadata: Metadata = { title: 'Post an ad' };

export default async function PostPage() {
  if (!(await getSession())) redirect('/login?next=/post');
  const lang = await getLang();
  const [tree, regions] = await Promise.all([getCategoryTree(), getRegions(true)]);
  return (
    <div className="narrow">
      <h1>{DICTS[lang].postAd}</h1>
      <PostForm lang={lang} tree={tree} regions={regions} />
    </div>
  );
}
