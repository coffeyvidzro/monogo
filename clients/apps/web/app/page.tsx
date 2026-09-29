import Link from "next/link";

import { SiteFooter } from "@/components/site-footer";
import { SiteHeader } from "@/components/site-header";

const primitives = [
  {
    title: "Voice",
    description:
      "Originate and control calls with programmable call actions, routing, and real-time events.",
  },
  {
    title: "Messaging",
    description:
      "Send and receive application messaging through one consistent platform surface.",
  },
  {
    title: "Numbers",
    description:
      "Connect phone numbers to applications, routes, and communication workflows.",
  },
];

const code = `const call = await leamout.calls.create({
  from: "+233200000000",
  to: "+233240000000",
  connection: "primary-carrier",
  url: "https://example.com/voice"
});`;

export default function Home() {
  return (
    <>
      <SiteHeader />

      <main>
        <section className="relative overflow-hidden border-b border-border">
          <div className="mx-auto grid min-h-[calc(100vh-4rem)] max-w-7xl items-center gap-16 px-5 py-20 sm:px-6 lg:grid-cols-[1.05fr_0.95fr] lg:px-8 lg:py-24">
            <div className="max-w-3xl">
              <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-border bg-muted px-3 py-1 text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">
                <span className="size-2 rounded-full bg-brand-orange" />
                Communications infrastructure
              </div>

              <h1 className="font-heading text-5xl font-semibold leading-[0.98] tracking-[-0.055em] sm:text-6xl lg:text-7xl">
                Build communications.
                <span className="mt-2 block text-brand-orange">Keep control.</span>
              </h1>

              <p className="mt-7 max-w-2xl text-lg leading-8 text-muted-foreground sm:text-xl">
                Leamout gives developers one programmable control plane for
                voice, messaging, numbers, and carrier connectivity.
              </p>

              <div className="mt-9 flex flex-col gap-3 sm:flex-row">
                <Link
                  href="#contact"
                  className="inline-flex h-11 items-center justify-center rounded-full bg-primary px-5 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-85"
                >
                  Get started
                </Link>
                <Link
                  href="#developers"
                  className="inline-flex h-11 items-center justify-center rounded-full border border-border px-5 text-sm font-medium transition-colors hover:bg-muted"
                >
                  Explore the API
                </Link>
              </div>
            </div>

            <div className="relative">
              <div className="absolute -inset-10 -z-10 bg-[radial-gradient(circle_at_center,color-mix(in_srgb,var(--brand-orange)_12%,transparent),transparent_68%)]" />
              <div className="overflow-hidden rounded-3xl border border-border bg-brand-charcoal text-brand-white shadow-2xl shadow-brand-charcoal/10">
                <div className="flex items-center justify-between border-b border-white/10 px-5 py-4">
                  <div className="flex items-center gap-2">
                    <span className="size-2 rounded-full bg-brand-orange" />
                    <span className="font-mono text-xs text-white/70">
                      create-call.ts
                    </span>
                  </div>
                  <span className="font-mono text-[11px] text-white/40">
                    API
                  </span>
                </div>
                <pre className="overflow-x-auto p-5 font-mono text-sm leading-7 text-white/85 sm:p-6">
                  <code>{code}</code>
                </pre>
                <div className="grid grid-cols-3 border-t border-white/10 text-center">
                  {["Voice", "Messaging", "Numbers"].map((item) => (
                    <div
                      key={item}
                      className="border-r border-white/10 px-3 py-4 text-xs text-white/55 last:border-r-0"
                    >
                      {item}
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </section>

        <section id="product" className="border-b border-border">
          <div className="mx-auto max-w-7xl px-5 py-24 sm:px-6 lg:px-8">
            <div className="grid gap-12 lg:grid-cols-[0.8fr_1.2fr]">
              <div>
                <p className="font-mono text-xs uppercase tracking-[0.14em] text-brand-orange">
                  Product
                </p>
                <h2 className="mt-4 max-w-md font-heading text-4xl font-semibold tracking-[-0.04em] sm:text-5xl">
                  The primitives for programmable communications.
                </h2>
              </div>

              <div className="grid gap-px overflow-hidden rounded-3xl border border-border bg-border md:grid-cols-3">
                {primitives.map((primitive) => (
                  <article key={primitive.title} className="bg-background p-6 sm:p-7">
                    <div className="mb-10 flex size-10 items-center justify-center rounded-full bg-accent text-sm font-semibold text-brand-orange">
                      {primitive.title.slice(0, 1)}
                    </div>
                    <h3 className="font-heading text-xl font-semibold tracking-[-0.025em]">
                      {primitive.title}
                    </h3>
                    <p className="mt-3 text-sm leading-6 text-muted-foreground">
                      {primitive.description}
                    </p>
                  </article>
                ))}
              </div>
            </div>
          </div>
        </section>

        <section id="connectivity" className="bg-brand-charcoal text-brand-white">
          <div className="mx-auto max-w-7xl px-5 py-24 sm:px-6 lg:px-8">
            <div className="grid gap-14 lg:grid-cols-[0.9fr_1.1fr] lg:items-end">
              <div>
                <p className="font-mono text-xs uppercase tracking-[0.14em] text-brand-orange">
                  Connectivity
                </p>
                <h2 className="mt-4 max-w-xl font-heading text-4xl font-semibold tracking-[-0.04em] sm:text-5xl">
                  Your carriers or ours. One control plane.
                </h2>
                <p className="mt-5 max-w-xl text-base leading-7 text-white/60">
                  Keep your existing carrier relationships with BYOC, or use
                  Leamout-managed connectivity. Your application keeps the same
                  programmable surface.
                </p>
              </div>

              <div className="grid gap-4 md:grid-cols-2">
                <article className="rounded-3xl border border-white/12 p-6">
                  <p className="font-mono text-xs uppercase tracking-[0.12em] text-white/45">
                    BYOC
                  </p>
                  <h3 className="mt-6 font-heading text-2xl font-semibold">
                    Bring your carrier.
                  </h3>
                  <p className="mt-3 text-sm leading-6 text-white/60">
                    Use your own SIP trunks, commercial agreements, and carrier
                    rates while Leamout handles application control and routing.
                  </p>
                </article>

                <article className="rounded-3xl border border-brand-orange/60 bg-brand-orange/8 p-6">
                  <p className="font-mono text-xs uppercase tracking-[0.12em] text-brand-orange">
                    Managed
                  </p>
                  <h3 className="mt-6 font-heading text-2xl font-semibold">
                    Use Leamout connectivity.
                  </h3>
                  <p className="mt-3 text-sm leading-6 text-white/60">
                    Let Leamout provide carrier connectivity and usage billing
                    while you build against the same communications API.
                  </p>
                </article>
              </div>
            </div>
          </div>
        </section>

        <section id="developers" className="border-b border-border">
          <div className="mx-auto max-w-7xl px-5 py-24 sm:px-6 lg:px-8">
            <div className="grid gap-12 lg:grid-cols-2 lg:items-start">
              <div>
                <p className="font-mono text-xs uppercase tracking-[0.14em] text-brand-orange">
                  Developers
                </p>
                <h2 className="mt-4 max-w-xl font-heading text-4xl font-semibold tracking-[-0.04em] sm:text-5xl">
                  Communications should feel like software.
                </h2>
                <p className="mt-5 max-w-xl text-base leading-7 text-muted-foreground">
                  Build call flows, react to events, route traffic, and connect
                  carriers without embedding telecom-specific complexity into
                  every product team.
                </p>
              </div>

              <div className="divide-y divide-border rounded-3xl border border-border">
                {[
                  ["REST API", "Create and control communications from your application."],
                  ["Events", "Receive lifecycle events for calls and messaging workflows."],
                  ["Routing", "Connect applications, numbers, and carrier connections."],
                ].map(([title, description]) => (
                  <div key={title} className="grid gap-3 p-6 sm:grid-cols-[140px_1fr]">
                    <h3 className="font-heading font-semibold">{title}</h3>
                    <p className="text-sm leading-6 text-muted-foreground">
                      {description}
                    </p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        <section id="contact">
          <div className="mx-auto max-w-7xl px-5 py-24 sm:px-6 lg:px-8">
            <div className="overflow-hidden rounded-[2rem] bg-brand-orange px-6 py-14 sm:px-10 lg:flex lg:items-center lg:justify-between lg:px-12">
              <div className="max-w-2xl text-brand-charcoal">
                <p className="font-mono text-xs uppercase tracking-[0.14em]">
                  Leamout
                </p>
                <h2 className="mt-4 font-heading text-4xl font-semibold tracking-[-0.04em] sm:text-5xl">
                  Build on a communications layer you control.
                </h2>
              </div>

              <div className="mt-8 lg:mt-0">
                <Link
                  href="mailto:hello@leamout.com"
                  className="inline-flex h-11 items-center justify-center rounded-full bg-brand-charcoal px-5 text-sm font-medium text-brand-white transition-opacity hover:opacity-90"
                >
                  Talk to us
                </Link>
              </div>
            </div>
          </div>
        </section>
      </main>

      <SiteFooter />
    </>
  );
}
