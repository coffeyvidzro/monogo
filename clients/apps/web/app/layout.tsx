import { Toaster } from "@leamout/ui/components/sonner";
import { TooltipProvider } from "@leamout/ui/components/tooltip";
import "@leamout/ui/globals.css";
import { plexMono, plexSans } from "@/lib/fonts";
import { constructMetadata } from "@/lib/metadata";
import { constructMetagraph, serializeSchema } from "@/lib/metagraph";

export const metadata = constructMetadata({
  title: "Programmable Communications Control Plane",
  description:
    "Leamout is a programmable communications control plane for building voice and messaging products using your own telecom carriers.",
});

const metagraph = constructMetagraph();

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      suppressHydrationWarning
      className={`${plexSans.variable} ${plexMono.variable}`}
    >
      <body className="bg-background text-foreground antialiased">
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{
            __html: serializeSchema(metagraph),
          }}
        />

        <div className="fixed -z-10 h-screen w-screen bg-linear-to-br from-emerald-100 via-blue-50 to-rose-100" />

        <TooltipProvider>
          <main className="min-h-screen py-32 antialiased">{children}</main>

          <Toaster richColors closeButton />
        </TooltipProvider>
      </body>
    </html>
  );
}
