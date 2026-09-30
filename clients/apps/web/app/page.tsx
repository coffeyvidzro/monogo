import Link from "next/link";

export default function Home() {
  return (
    <main className="min-h-screen bg-background text-foreground">
      <div className="border-b border-border">
        <div className="mx-auto flex h-16 max-w-[1440px] items-center justify-between px-5 sm:px-8 lg:px-10">
          <p className="font-heading text-lg font-semibold tracking-[-0.04em]">
            LEAMOUT
          </p>

          <p className="font-mono text-[11px] uppercase tracking-[0.16em] text-muted-foreground">
            Coming soon
          </p>
        </div>
      </div>

      <div aria-hidden="true" className="border-b border-brand-orange bg-brand-orange">
        <div className="mx-auto flex h-14 max-w-[1440px] overflow-hidden">
          <div className="w-[42%] border-r border-brand-charcoal/15" />
          <div className="w-[18%] -skew-x-[28deg] border-x border-brand-charcoal/15 bg-brand-charcoal/10" />
          <div className="flex-1" />
        </div>
      </div>

      <section className="mx-auto grid min-h-[calc(100vh-7.5rem)] max-w-[1440px] grid-rows-[1fr_auto] px-5 sm:px-8 lg:px-10">
        <div className="grid items-center gap-12 py-16 lg:grid-cols-[1fr_340px] lg:gap-20 lg:py-20">
          <div>
            <p className="font-mono text-xs font-medium uppercase tracking-[0.16em] text-brand-orange">
              Programmable communications infrastructure
            </p>

            <h1 className="mt-7 max-w-6xl font-heading text-[clamp(3.75rem,9vw,9.5rem)] font-medium leading-[0.82] tracking-[-0.065em]">
              ONE CONTROL
              <span className="block">PLANE FOR</span>
              <span className="block text-brand-orange">COMMUNICATIONS.</span>
            </h1>
          </div>

          <aside className="self-end border-t border-border pt-6 lg:self-center">
            <p className="text-base leading-7 text-muted-foreground">
              Build voice, messaging, and number products with your own carriers
              or Leamout-managed connectivity.
            </p>

            <Link
              href="mailto:hello@leamout.com?subject=Leamout%20Waitlist"
              className="mt-7 inline-flex h-11 items-center justify-center rounded-lg bg-primary px-5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/25"
            >
              Join the waitlist
            </Link>

            <p className="mt-3 text-xs leading-5 text-muted-foreground">
              We&apos;ll share launch updates and early-access details.
            </p>
          </aside>
        </div>

        <div className="border-t border-border">
          <div className="grid lg:grid-cols-[1fr_340px]">
            <div className="grid grid-cols-2 border-b border-border lg:grid-cols-4 lg:border-b-0 lg:border-r">
              {["Voice", "Messaging", "Numbers", "Connectivity"].map((item) => (
                <div
                  key={item}
                  className="border-r border-border px-0 py-5 last:border-r-0 even:border-r-0 lg:even:border-r lg:last:border-r-0"
                >
                  <p className="font-mono text-[11px] uppercase tracking-[0.14em] text-muted-foreground">
                    {item}
                  </p>
                </div>
              ))}
            </div>

            <div className="relative flex min-h-28 items-center overflow-hidden lg:min-h-0">
              <div className="absolute left-0 top-1/2 h-px w-full bg-border" />
              <div className="absolute left-0 top-1/2 h-px w-2/3 bg-brand-orange" />
              <div className="absolute left-[64%] top-1/2 size-2 -translate-y-1/2 rounded-full bg-brand-orange" />
              <div className="absolute left-[66%] top-1/2 h-px w-20 origin-left -translate-y-1/2 rotate-[22deg] bg-brand-orange" />
              <div className="absolute left-[66%] top-1/2 h-px w-24 origin-left -translate-y-1/2 -rotate-[22deg] bg-brand-orange" />
            </div>
          </div>

          <div className="flex items-center justify-between border-t border-border py-5">
            <p className="text-xs text-muted-foreground">
              © {new Date().getFullYear()} Leamout
            </p>
            <p className="font-mono text-[10px] uppercase tracking-[0.14em] text-muted-foreground">
              BYOC / Managed connectivity
            </p>
          </div>
        </div>
      </section>
    </main>
  );
}
