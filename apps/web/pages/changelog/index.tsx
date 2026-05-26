import type { GetServerSideProps } from "next";
import Head from "next/head";
import Link from "next/link";
import MarkdownContent from "../../components/ui/MarkdownContent";
import { requestChangelogPosts } from "../../features/changelog/changelog-client";
import type { ChangelogPost } from "../../features/changelog/types";
import { createRuntimeConfig } from "../../lib/runtime-config";
import { getSiteURL } from "../../lib/site";

type ChangelogIndexProps = {
  posts: ChangelogPost[];
};

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat("en-US", {
    month: "long",
    day: "numeric",
    year: "numeric",
    timeZone: "UTC",
  }).format(date);
}

export const getServerSideProps: GetServerSideProps<ChangelogIndexProps> = async () => {
  try {
    const data = await requestChangelogPosts(createRuntimeConfig());
    return { props: { posts: data.posts || [] } };
  } catch {
    return { props: { posts: [] } };
  }
};

export default function ChangelogIndexPage({ posts }: ChangelogIndexProps) {
  const siteURL = getSiteURL();
  const title = "GeoDuels Changelog";
  const description =
    "Read GeoDuels release notes, gameplay updates, new features, and fixes.";

  return (
    <>
      <Head>
        <title>{title}</title>
        <meta name="description" content={description} />
        <meta name="robots" content="index,follow" />
        <link rel="canonical" href={`${siteURL}/changelog`} />
        <meta property="og:type" content="website" />
        <meta property="og:title" content={title} />
        <meta property="og:description" content={description} />
        <meta property="og:url" content={`${siteURL}/changelog`} />
      </Head>
      <main className="min-h-screen bg-[#0a1018] px-4 py-10 text-[#f4f9ff] sm:px-6">
        <div className="mx-auto max-w-4xl">
          <Link href="/" className="text-sm font-bold text-[#77f0be] hover:text-white">
            GeoDuels
          </Link>
          <header className="mt-8">
            <p className="text-xs font-bold uppercase tracking-[0.16em] text-[#77f0be]">
              Updates
            </p>
            <h1 className="mt-2 text-4xl font-black tracking-tight text-white sm:text-5xl">
              GeoDuels Changelog
            </h1>
            <p className="mt-4 max-w-2xl text-base leading-7 text-[#a9bfd4]">
              Product notes for new game modes, matchmaking updates, quality fixes,
              and other changes to GeoDuels.
            </p>
          </header>
          <div className="mt-10 space-y-4">
            {posts.map((post) => (
              <article key={post.id} className="glass-panel rounded-[18px] p-5 sm:p-6">
                <time dateTime={post.updatedAt} className="text-xs font-bold uppercase tracking-[0.14em] text-[#77f0be]">
                  {formatDate(post.updatedAt)}
                </time>
                <h2 className="mt-2 text-2xl font-black text-white">
                  <Link href={`/changelog/${encodeURIComponent(post.slug)}`} className="hover:text-[#77f0be]">
                    {post.title}
                  </Link>
                </h2>
                {post.summary ? (
                  <MarkdownContent markdown={post.summary} compact className="mt-3" />
                ) : null}
              </article>
            ))}
            {posts.length === 0 ? (
              <div className="glass-panel rounded-[18px] p-6 text-[#a9bfd4]">
                No changelog posts have been published yet.
              </div>
            ) : null}
          </div>
        </div>
      </main>
    </>
  );
}
