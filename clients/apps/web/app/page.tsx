import Link from "next/link";

export default function Home() {
  return (
    <main className="flex min-h-screen items-center bg-background">
      <div className="mx-auto w-full max-w-5xl px-5 py-16 sm:px-6 lg:px-8">
        <div className="max-w-3xl">
          <p className="font-mono text-xs font-medium uppercase tracking-[0.16em] text-brand-orange">
            Coming soon
          </p>

          <h1 className="mt-6 font-heading text-5xl font-semibold leading-[0.96] tracking-[-0.055em] sm:text-6xl lg:text-7xl">
            Leamout
          </h1>

          <p className="mt-6 max-w-2xl text-lg leading-8 text-muted-foreground sm:text-xl">
            Programmable communications infrastructure for voice, messaging,
            numbers, and carrier connectivity.
          </p>

          <p className="mt-4 max-w-2xl text-base leading-7 text-muted-foreground">
            Bring your own carriers or use Leamout-managed connectivity through
            one programmable control plane.
          </p>

          <div className="mt-10 flex flex-col gap-3 sm:flex-row sm:items-center">
            <Link
              href="mailto:hello@leamout.com?subject=Leamout%20Waitlist"
              className="inline-flex h-11 items-center justify-center rounded-full bg-primary px-5 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-85"
            >
              Join the waitlist
            </Link>

            <span className="text-sm text-muted-foreground">
              We&apos;ll share updates as Leamout gets closer to launch.
            </span>
          </div>
        </div>

        <div className="mt-20 border-t border-border pt-6 text-xs text-muted-foreground">
          © {new Date().getFullYear()} Leamout
        </div>
      </div>
    </main>
  );
}
