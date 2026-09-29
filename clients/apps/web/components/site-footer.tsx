import Link from "next/link";

export function SiteFooter() {
  return (
    <footer className="border-t border-border bg-background">
      <div className="mx-auto grid max-w-7xl gap-10 px-5 py-10 sm:px-6 md:grid-cols-[1fr_auto] lg:px-8">
        <div>
          <Link
            href="/"
            className="font-heading text-lg font-semibold tracking-[-0.03em]"
          >
            Leamout
          </Link>
          <p className="mt-3 max-w-md text-sm leading-6 text-muted-foreground">
            Programmable communications infrastructure for voice, messaging,
            numbers, and carrier connectivity.
          </p>
        </div>

        <nav
          aria-label="Footer navigation"
          className="grid grid-cols-2 gap-x-10 gap-y-3 text-sm"
        >
          <Link href="#product" className="text-muted-foreground hover:text-foreground">
            Product
          </Link>
          <Link
            href="#connectivity"
            className="text-muted-foreground hover:text-foreground"
          >
            Connectivity
          </Link>
          <Link
            href="#developers"
            className="text-muted-foreground hover:text-foreground"
          >
            Developers
          </Link>
          <Link href="#contact" className="text-muted-foreground hover:text-foreground">
            Get started
          </Link>
        </nav>
      </div>

      <div className="border-t border-border">
        <div className="mx-auto flex max-w-7xl flex-col gap-2 px-5 py-5 text-xs text-muted-foreground sm:flex-row sm:items-center sm:justify-between sm:px-6 lg:px-8">
          <p>© {new Date().getFullYear()} Leamout.</p>
          <p>Programmable communications control plane.</p>
        </div>
      </div>
    </footer>
  );
}
