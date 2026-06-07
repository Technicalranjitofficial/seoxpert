import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "SeoXpert — AI-Powered SEO Platform | Rank Higher, Grow Faster",
  description:
    "SeoXpert is the all-in-one AI-powered SEO platform for agencies and businesses. Keyword research, technical audits, backlink analysis, rank tracking, content optimization, and competitor intelligence — all in one place.",
  keywords: [
    "SEO platform", "AI SEO tools", "keyword research tool",
    "technical SEO audit", "backlink analysis", "rank tracker",
    "content optimization", "SEO agency tools", "on-page SEO",
    "off-page SEO", "SERP analysis", "SEO competitor analysis",
    "local SEO", "enterprise SEO software", "SeoXpert",
  ],
  authors: [{ name: "SeoXpert" }],
  creator: "SeoXpert",
  metadataBase: new URL("https://seoxpert.io"),
  openGraph: {
    type: "website",
    locale: "en_US",
    url: "https://seoxpert.io",
    title: "SeoXpert — AI-Powered SEO Platform",
    description: "Rank higher, grow faster. AI-powered SEO tools for keyword research, technical audits, rank tracking, and content optimization.",
    siteName: "SeoXpert",
  },
  twitter: {
    card: "summary_large_image",
    title: "SeoXpert — AI-Powered SEO Platform",
    description: "Rank higher, grow faster. The complete SEO platform for modern businesses.",
    creator: "@seoxpert",
  },
  robots: { index: true, follow: true },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className="scroll-smooth">
      <body className={inter.className}>{children}</body>
    </html>
  );
}
