import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  transpilePackages: ["@leamout/ui"],

  async headers() {
    return [
      {
        source: "/:path*",
        headers: [
          {
            key: "Referrer-Policy",
            value: "no-referrer-when-downgrade",
          },
          {
            key: "X-DNS-Prefetch-Control",
            value: "on",
          },
          {
            key: "X-Frame-Options",
            value: "DENY",
          },
        ],
      },
    ];
  },
  async redirects() {
    return [
      {
        source: "/llms.txt",
        destination: "https://docs.leamout.com/llms.txt",
        permanent: true,
      },
      {
        source: "/llms-full.txt",
        destination: "https://docs.leamout.com/llms-full.txt",
        permanent: true,
      },
      {
        source: "/docs/:path*",
        destination: "https://docs.leamout.com/:path*",
        permanent: true,
      },
    ];
  },
};

export default nextConfig;
