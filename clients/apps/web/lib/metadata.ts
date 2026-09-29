import type { Metadata } from "next";

export function constructMetadata({
  title,
  description,
  image = "/og.png",
  path = "",
  noIndex = false,
}: {
  title: string;
  description: string;
  image?: string;
  path?: string;
  noIndex?: boolean;
}): Metadata {
  const url = new URL(path, "https://leamout.com").toString();
  const metadataTitle = `${title} | Leamout`;

  return {
    metadataBase: new URL("https://leamout.com"),
    title: metadataTitle,
    description,

    alternates: {
      canonical: url,
    },

    openGraph: {
      title: metadataTitle,
      description,
      url,
      siteName: "Leamout",
      locale: "en_US",
      type: "website",
      images: [
        {
          url: image,
          width: 1200,
          height: 630,
          alt: metadataTitle,
        },
      ],
    },

    twitter: {
      card: "summary_large_image",
      title: metadataTitle,
      description,
      images: [image],
    },

    robots: {
      index: !noIndex,
      follow: !noIndex,
      googleBot: {
        index: !noIndex,
        follow: !noIndex,
        "max-video-preview": -1,
        "max-image-preview": "large",
        "max-snippet": -1,
      },
    },
  };
}
