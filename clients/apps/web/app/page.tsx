export default function Home() {
  return (
    <main className="min-h-screen bg-background text-foreground">
      <div className="mx-auto flex min-h-screen w-full max-w-6xl flex-col px-5 sm:px-6 lg:px-8">
        <section className="grid flex-1 items-center gap-14 py-16 lg:grid-cols-[minmax(0,1.15fr)_minmax(320px,0.85fr)] lg:gap-16 lg:py-20">
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
      </div>
    </main>
  );
}
