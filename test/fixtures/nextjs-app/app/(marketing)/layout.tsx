export default function MarketingLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div>
      <nav>Marketing Nav</nav>
      {children}
    </div>
  );
}
