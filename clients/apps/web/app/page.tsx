import Link from "next/link";

const capabilities = ["Voice", "Messaging", "Numbers", "Connectivity"];

export default function Home() {
  return (
    <main className="min-h-screen bg-background text-foreground">
      <div className="mx-auto flex min-h-screen w-full max-w-6xl flex-col px-5 sm:px-6 lg:px-8">
        <header className="flex h-16 items-center justify-between border-b border-border">
          <p className="font-heading text-lg font-semibold tracking-[-0.035em]">
            Leamout
          </p>

          <div className="flex items-center gap-2 font-mono text-[10px] uppercase tracking-[0.14em] text-muted-foreground">
            <span className="size-1.5 rounded-full bg-brand-orange" />
            Coming soon
          </div>
        </header>

        <section className="grid flex-1 items-center gap-14 py-16 lg:grid-cols-[minmax(0,1.15fr)_minmax(320px,0.85fr)] lg:gap-16 lg:py-20">
          <div className="max-w-2xl">
            <p className="font-mono text-[11px] font-medium uppercase tracking-[0.16em] text-brand-orange">
              Programmable communications
            </p>

            <h1 className="mt-6 font-heading text-[clamp(3.5rem,7vw,6.5rem)] font-semibold leading-[0.92] tracking-[-0.06em]">
              Communications infrastructure,
              <span className="block">under your control.</span>
            </h1>

            <p className="mt-7 max-w-xl text-lg leading-8 text-muted-foreground">
              Build voice, messaging, and number products through one control
              plane. Bring your own carriers or use Leamout-managed
              connectivity.
            </p>

            <div className="mt-9 flex flex-col gap-3 sm:flex-row sm:items-center">
              <Link
                href="mailto:hello@leamout.com?subject=Leamout%20Waitlist"
                className="inline-flex h-11 w-fit items-center justify-center rounded-lg bg-primary px-5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/25"
              >
                Join the waitlist
              </Link>

              <p className="text-sm text-muted-foreground">
                Early access and launch updates.
              </p>
            </div>
          </div>

          <div className="rounded-2xl border border-border bg-muted/30 p-5 sm:p-6">
            <div className="flex items-center justify-between border-b border-border pb-4">
              <p className="font-mono text-[10px] uppercase tracking-[0.14em] text-muted-foreground">
                Connection path
              </p>
              <span className="size-2 rounded-full bg-brand-orange" />
            </div>

            <div className="relative mt-6 min-h-[320px]">
              <div className="mx-auto w-fit rounded-lg border border-border bg-background px-4 py-3 text-center">
                <p className="font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground">
                  Your application
                </p>
              </div>

              <div className="mx-auto h-10 w-px bg-brand-orange" />

              <div className="mx-auto w-fit rounded-lg bg-brand-charcoal px-5 py-3 text-center text-brand-white">
                <p className="font-heading text-sm font-semibold tracking-[-0.02em]">
                  Leamout
                </p>
              </div>

              <div className="relative mx-auto h-14 w-40">
                <div className="absolute left-1/2 top-0 h-7 w-px -translate-x-1/2 bg-brand-orange" />
                <div className="absolute left-[25%] right-[25%] top-7 h-px bg-brand-orange" />
                <div className="absolute left-[25%] top-7 h-7 w-px bg-brand-orange" />
                <div className="absolute right-[25%] top-7 h-7 w-px bg-brand-orange" />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="rounded-lg border border-border bg-background px-3 py-4 text-center">
                  <p className="font-mono text-[10px] uppercase tracking-[0.11em] text-muted-foreground">
                    Your carrier
                  </p>
                  <p className="mt-1 text-xs text-foreground">BYOC</p>
                </div>

                <div className="rounded-lg border border-border bg-background px-3 py-4 text-center">
                  <p className="font-mono text-[10px] uppercase tracking-[0.11em] text-muted-foreground">
                    Leamout carrier
                  </p>
                  <p className="mt-1 text-xs text-foreground">Managed</p>
                </div>
              </div>
            </div>
          </div>
        </section>

        <div className="border-t border-border">
          <div className="grid grid-cols-2 md:grid-cols-4">
            {capabilities.map((item) => (
              <div
                key={item}
                className="border-b border-border py-4 odd:pr-4 even:pl-4 md:border-b-0 md:border-r md:px-4 md:first:pl-0 md:last:border-r-0 md:last:pr-0"
              >
                <p className="font-mono text-[10px] uppercase tracking-[0.14em] text-muted-foreground">
                  {item}
                </p>
              </div>
            ))}
          </div>
        </div>

        <footer className="flex items-center justify-between border-t border-border py-5 text-xs text-muted-foreground">
          <p>© {new Date().getFullYear()} Leamout</p>
          <p className="font-mono text-[10px] uppercase tracking-[0.12em]">
            Built for communications
          </p>
        </footer>
      </div>
    </main>
  );
}
