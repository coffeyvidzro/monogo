import "@leamout/ui/globals.css";
import { plexMono, plexSans } from "@/lib/fonts";
import { constructMetadata } from "@/lib/metadata";
import RootProviders from "./providers";

export const metadata = constructMetadata({
  title: "Programmable Communications Control Plane",
  description:
    "Leamout is a programmable communications control plane for voice, messaging, numbers, and carrier connectivity. Bring your own carriers or use Leamout-managed connectivity.",
});

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
        <RootProviders>{children}</RootProviders>
      </body>
    </html>
  );
}
