import Link from "next/link";

const navigation = [
  { href: "#product", label: "Product" },
  { href: "#connectivity", label: "Connectivity" },
  { href: "#developers", label: "Developers" },
];

export function SiteHeader() {
  return (
    <header className="sticky top-0 z-50 border-b border-border/70 bg-background/95 backdrop-blur">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-5 sm:px-6 lg:px-8">
        <Link
          href="/"
          className="font-heading text-lg font-semibold tracking-[-0.03em]"
        >
          Leamout
        </Link>

        <nav
          aria-label="Primary navigation"
          className="hidden items-center gap-7 md:flex"
        >
          {navigation.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className="text-sm text-muted-foreground transition-colors hover:text-foreground"
            >
              {item.label}
            </Link>
          ))}
        </nav>

        <div className="hidden items-center gap-3 md:flex">
          <Link
            href="#developers"
            className="inline-flex h-9 items-center justify-center rounded-full border border-border px-4 text-sm font-medium transition-colors hover:bg-muted"
          >
            Read the API
          </Link>
          <Link
            href="#contact"
            className="inline-flex h-9 items-center justify-center rounded-full bg-primary px-4 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-85"
          >
            Get started
          </Link>
        </div>

        <details className="relative md:hidden">
          <summary className="cursor-pointer list-none rounded-full border border-border px-4 py-2 text-sm font-medium">
            Menu
          </summary>
          <div className="absolute right-0 mt-3 w-56 rounded-2xl border border-border bg-background p-2 shadow-lg">
            <nav aria-label="Mobile navigation" className="grid">
              {navigation.map((item) => (
                <Link
                  key={item.href}
                  href={item.href}
                  className="rounded-xl px-3 py-2.5 text-sm text-muted-foreground hover:bg-muted hover:text-foreground"
                >
                  {item.label}
                </Link>
              ))}
              <Link
                href="#contact"
                className="mt-1 rounded-xl bg-primary px-3 py-2.5 text-sm font-medium text-primary-foreground"
              >
                Get started
              </Link>
            </nav>
          </div>
        </details>
      </div>
    </header>
  );
}
