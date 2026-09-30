import "@leamout/ui/globals.css";
import { plexMono, plexSans } from "@/lib/fonts";
import { constructMetadata } from "@/lib/metadata";
import RootProviders from "./providers";

export const metadata = constructMetadata({
  title: "Modern communications infrastructure. On your terms",
  description:
    "Leamout is a programmable communications control plane for building and operating voice, messaging, numbering, routing, and carrier-connected telecom products.",
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
