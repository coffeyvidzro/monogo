"use client";

import { ReactNode } from "react";
import { Toaster } from "@leamout/ui/components/sonner";
import { TooltipProvider } from "@leamout/ui/components/tooltip";
import { Analytics } from "@vercel/analytics/next";
import { SpeedInsights } from "@vercel/speed-insights/next";

export default function RootProviders({ children }: { children: ReactNode }) {
  return (
    <TooltipProvider>
      <Toaster className="pointer-events-auto" position="bottom-center" />
      {children}
      <Analytics />
      <SpeedInsights />
    </TooltipProvider>
  );
}
