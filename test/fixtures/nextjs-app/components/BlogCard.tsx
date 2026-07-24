import React from 'react';
import Link from 'next/link';

interface Post {
  slug: string;
  title: string;
  excerpt: string;
  date: string;
}

interface BlogCardProps {
  post: Post;
}

export default function BlogCard({ post }: BlogCardProps) {
  return (
    <div className="blog-card">
      <Link href={`/blog/${post.slug}`}>
        <h2>{post.title}</h2>
        <p>{post.excerpt}</p>
        <time>{post.date}</time>
      </Link>
    </div>
  );
}
