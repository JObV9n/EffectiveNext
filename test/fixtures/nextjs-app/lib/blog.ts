export interface BlogPost {
  slug: string;
  title: string;
  excerpt: string;
  date: string;
  content: string;
}

export async function getBlogPosts(): Promise<BlogPost[]> {
  return [
    {
      slug: 'hello-world',
      title: 'Hello World',
      excerpt: 'Our first blog post.',
      date: '2024-01-15',
      content: '<p>Welcome to our blog!</p>',
    },
    {
      slug: 'getting-started',
      title: 'Getting Started with effectiveNext',
      excerpt: 'Learn how to use effectiveNext for faster builds.',
      date: '2024-01-20',
      content: '<p>effectiveNext makes your builds faster.</p>',
    },
  ];
}

export async function getPostBySlug(
  slug: string
): Promise<BlogPost | null> {
  const posts = await getBlogPosts();
  return posts.find((p) => p.slug === slug) || null;
}
