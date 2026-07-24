import { getBlogPosts } from '@/lib/blog';
import BlogCard from '@/components/BlogCard';

export default async function BlogPage() {
  const posts = await getBlogPosts();

  return (
    <div>
      <h1>Blog</h1>
      <div className="blog-grid">
        {posts.map((post) => (
          <BlogCard key={post.slug} post={post} />
        ))}
      </div>
    </div>
  );
}
