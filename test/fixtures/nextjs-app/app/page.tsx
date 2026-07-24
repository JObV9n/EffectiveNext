import Link from 'next/link';

export default function Home() {
  return (
    <div>
      <h1>effectiveNext Test App</h1>
      <p>Welcome to the effectiveNext build accelerator test application.</p>
      <div>
        <Link href="/dashboard">Dashboard</Link>
        <Link href="/blog">Blog</Link>
        <Link href="/settings">Settings</Link>
      </div>
    </div>
  );
}
