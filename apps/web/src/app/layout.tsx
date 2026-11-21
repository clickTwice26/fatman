import type { Metadata, Viewport } from "next";
import { Onest } from "next/font/google";
import type { ReactNode } from "react";
import { ThemeRegistry } from "@/theme/ThemeRegistry";
import "./globals.css";

const onest = Onest({
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
  variable: "--font-onest",
  display: "swap",
});

export const metadata: Metadata = {
  title: {
    default: "Fatman — Supportive Weight-Loss Community & Quests",
    template: "%s | Fatman",
  },
  description:
    "Connect with accountability buddies, share daily wins, and complete collaborative weight-loss quests together.",
};

export const viewport: Viewport = {
  colorScheme: "light dark",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" className={onest.variable}>
      <body>
        <ThemeRegistry>{children}</ThemeRegistry>
      </body>
    </html>
  );
}
