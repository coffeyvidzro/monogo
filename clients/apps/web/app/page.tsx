const runtimeFeatures = [
  "SIP-native",
  "Realtime orchestration",
  "Bring your own carrier",
];

const waveform = [
  { id: "a", height: 8 },
  { id: "b", height: 14 },
  { id: "c", height: 20 },
  { id: "d", height: 10 },
  { id: "e", height: 24 },
  { id: "f", height: 16 },
  { id: "g", height: 28 },
  { id: "h", height: 12 },
  { id: "i", height: 19 },
  { id: "j", height: 25 },
  { id: "k", height: 11 },
  { id: "l", height: 17 },
  { id: "m", height: 9 },
  { id: "n", height: 21 },
  { id: "o", height: 13 },
  { id: "p", height: 7 },
];

function ArrowIcon() {
  return (
    <svg aria-hidden="true" className="size-4" fill="none" viewBox="0 0 16 16">
      <path
        d="M3.5 8h9m-3.25-3.25L12.5 8l-3.25 3.25"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.5"
      />
    </svg>
  );
}

function GitHubIcon() {
  return (
    <svg
      aria-hidden="true"
      className="size-4"
      fill="currentColor"
      viewBox="0 0 24 24"
    >
      <path d="M12 .7a11.5 11.5 0 0 0-3.64 22.4c.58.1.79-.25.79-.56v-2.23c-3.22.7-3.9-1.37-3.9-1.37-.53-1.34-1.29-1.7-1.29-1.7-1.05-.72.08-.7.08-.7 1.17.08 1.78 1.2 1.78 1.2 1.04 1.77 2.72 1.26 3.38.96.1-.75.4-1.26.74-1.55-2.57-.3-5.27-1.29-5.27-5.69 0-1.26.45-2.28 1.19-3.09-.12-.29-.52-1.46.11-3.05 0 0 .97-.31 3.16 1.18a10.98 10.98 0 0 1 5.75 0c2.2-1.49 3.16-1.18 3.16-1.18.63 1.59.23 2.76.11 3.05.74.81 1.19 1.83 1.19 3.09 0 4.42-2.71 5.39-5.29 5.68.42.36.79 1.06.79 2.14v3.16c0 .31.21.67.8.56A11.5 11.5 0 0 0 12 .7Z" />
    </svg>
  );
}

function BrandMark() {
  return (
    <span
      className="relative flex size-7 items-center justify-center"
      aria-hidden="true"
    >
      <span className="absolute h-5 w-2.5 -translate-x-[5px] rounded-l-full bg-brand-orange" />
      <span className="absolute h-5 w-2.5 translate-x-[5px] rounded-r-full bg-brand-charcoal" />
      <span className="absolute size-1.5 rounded-full bg-background" />
    </span>
  );
}

export default function Home() {
  return (
    <main className="min-h-screen overflow-hidden bg-background text-foreground">
      <div className="mx-auto flex min-h-screen w-full max-w-[1440px] flex-col px-5 sm:px-8 lg:px-12">
        <header className="flex h-20 items-center justify-between border-b border-border/70">
          <a
            className="flex items-center gap-2.5 font-heading text-lg font-semibold tracking-[-0.04em]"
            href="/"
          >
            <BrandMark />
            Leamout
          </a>

          <nav
            aria-label="Primary navigation"
            className="hidden items-center gap-8 text-sm text-muted-foreground md:flex"
          >
            <a
              className="transition-colors hover:text-foreground"
              href="#platform"
            >
              Platform
            </a>
            <a
              className="transition-colors hover:text-foreground"
              href="#deploy"
            >
              Deploy
            </a>
            <a
              className="transition-colors hover:text-foreground"
              href="https://github.com/leamout/monogo"
            >
              Developers
            </a>
          </nav>

          <a
            className="inline-flex h-10 items-center gap-2 rounded-md bg-brand-charcoal px-4 text-sm font-medium text-brand-white transition-transform hover:-translate-y-0.5"
            href="mailto:hello@leamout.com"
          >
            Talk to us
            <ArrowIcon />
          </a>
        </header>

        <section
          className="grid flex-1 items-center gap-14 py-14 lg:grid-cols-[minmax(0,1.05fr)_minmax(440px,0.95fr)] lg:gap-20 lg:py-20"
          id="platform"
        >
          <div className="max-w-3xl">
            <div className="mb-7 flex items-center gap-3 font-mono text-[11px] font-medium uppercase tracking-[0.15em] text-muted-foreground">
              <span className="size-2 rounded-full bg-brand-orange shadow-[0_0_0_4px_rgba(244,62,1,0.1)]" />
              Voice infrastructure, reimagined
            </div>

            <h1 className="font-heading text-[clamp(3rem,6.3vw,6rem)] font-medium leading-[0.95] tracking-[-0.065em]">
              Carrier-grade
              <br />
              voice agents.
            </h1>

            <p className="mt-8 max-w-xl text-lg leading-8 text-muted-foreground sm:text-xl">
              The open platform for building and operating autonomous voice
              agents. Run it yourself or use Leamout Cloud.
            </p>

            <div className="mt-10 flex flex-wrap items-center gap-3">
              <a
                className="inline-flex h-12 items-center gap-2 rounded-md bg-brand-orange px-5 text-sm font-semibold text-white transition-transform hover:-translate-y-0.5"
                href="mailto:hello@leamout.com"
              >
                Build with Leamout
                <ArrowIcon />
              </a>
              <a
                className="inline-flex h-12 items-center gap-2 rounded-md border border-border bg-background px-5 text-sm font-semibold transition-colors hover:bg-muted"
                href="https://github.com/leamout/monogo"
              >
                <GitHubIcon />
                Explore the source
              </a>
            </div>

            <div className="mt-12 flex flex-wrap gap-x-6 gap-y-3 border-t border-border/70 pt-6">
              {runtimeFeatures.map((feature) => (
                <span
                  className="flex items-center gap-2 font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground"
                  key={feature}
                >
                  <span className="size-1 rounded-full bg-brand-orange" />
                  {feature}
                </span>
              ))}
            </div>
          </div>

          <div
            className="relative rounded-2xl border border-border bg-muted/40 p-4 shadow-[0_30px_80px_-45px_rgba(48,43,40,0.45)] sm:p-6"
            id="deploy"
          >
            <div className="pointer-events-none absolute -right-24 -top-24 size-64 rounded-full bg-brand-orange/8 blur-3xl" />
            <div className="relative flex items-center justify-between border-b border-border pb-4">
              <p className="font-mono text-[10px] uppercase tracking-[0.14em] text-muted-foreground">
                Live call path
              </p>
              <div className="flex items-center gap-2 font-mono text-[9px] uppercase tracking-[0.12em] text-muted-foreground">
                <span className="size-2 animate-pulse rounded-full bg-emerald-500" />
                Active
              </div>
            </div>

            <div className="relative mt-5 overflow-hidden rounded-xl border border-border bg-background p-5 sm:p-7">
              <div className="absolute inset-0 bg-[linear-gradient(to_right,rgba(105,105,93,0.06)_1px,transparent_1px),linear-gradient(to_bottom,rgba(105,105,93,0.06)_1px,transparent_1px)] bg-[size:28px_28px]" />
              <div className="relative">
                <div className="mx-auto w-fit rounded-md border border-border bg-background px-5 py-3 text-center shadow-sm">
                  <p className="font-mono text-[9px] uppercase tracking-[0.13em] text-muted-foreground">
                    Caller
                  </p>
                  <p className="mt-1 text-xs font-medium">Your network</p>
                </div>

                <div className="mx-auto flex h-10 w-px items-center bg-brand-orange/60">
                  <span className="-ml-1 block size-2 rounded-full bg-brand-orange" />
                </div>

                <div className="rounded-xl bg-brand-charcoal p-5 text-brand-white shadow-xl shadow-brand-charcoal/15">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2.5">
                      <BrandMark />
                      <span className="font-heading text-sm font-semibold">
                        Leamout runtime
                      </span>
                    </div>
                    <span className="rounded-full border border-white/15 px-2 py-1 font-mono text-[8px] uppercase tracking-[0.12em] text-white/60">
                      38 ms
                    </span>
                  </div>

                  <div className="mt-5 grid grid-cols-3 gap-2">
                    {["Listen", "Reason", "Speak"].map((step, index) => (
                      <div
                        className="rounded-md border border-white/10 bg-white/[0.06] px-2 py-3 text-center"
                        key={step}
                      >
                        <p className="font-mono text-[8px] text-white/40">
                          0{index + 1}
                        </p>
                        <p className="mt-1 text-[11px] font-medium">{step}</p>
                      </div>
                    ))}
                  </div>

                  <div className="mt-4 flex h-8 items-center justify-center gap-1 overflow-hidden rounded-md bg-black/15 px-4">
                    {waveform.map((bar) => (
                      <span
                        className="w-0.5 rounded-full bg-brand-orange"
                        key={bar.id}
                        style={{ height: bar.height }}
                      />
                    ))}
                  </div>
                </div>

                <div className="relative mx-auto h-12 w-48">
                  <div className="absolute left-1/2 top-0 h-6 w-px -translate-x-1/2 bg-brand-orange/60" />
                  <div className="absolute left-1/4 right-1/4 top-6 h-px bg-brand-orange/60" />
                  <div className="absolute left-1/4 top-6 h-6 w-px bg-brand-orange/60" />
                  <div className="absolute right-1/4 top-6 h-6 w-px bg-brand-orange/60" />
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div className="rounded-md border border-border bg-background px-3 py-3.5 text-center shadow-sm">
                    <p className="font-mono text-[9px] uppercase tracking-[0.11em] text-muted-foreground">
                      Your carrier
                    </p>
                    <p className="mt-1 text-xs font-medium">SIP / WebRTC</p>
                  </div>
                  <div className="rounded-md border border-border bg-background px-3 py-3.5 text-center shadow-sm">
                    <p className="font-mono text-[9px] uppercase tracking-[0.11em] text-muted-foreground">
                      AI providers
                    </p>
                    <p className="mt-1 text-xs font-medium">Your choice</p>
                  </div>
                </div>
              </div>
            </div>

            <div className="relative mt-4 flex items-center justify-between px-1 font-mono text-[9px] uppercase tracking-[0.12em] text-muted-foreground">
              <span>Open source core</span>
              <span>Cloud or self-hosted</span>
            </div>
          </div>
        </section>
      </div>
    </main>
  );
}
