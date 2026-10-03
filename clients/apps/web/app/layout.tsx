import "@leamout/ui/globals.css";
import { plexMono, plexSans } from "@/lib/fonts";
import { constructMetadata } from "@/lib/metadata";
import RootProviders from "./providers";

export const metadata = constructMetadata({
  title: "Carrier-grade voice agents",
  description:
    "The open platform for building and operating autonomous voice agents. Run it yourself or use Leamout Cloud.",
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
