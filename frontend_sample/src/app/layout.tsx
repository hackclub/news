import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./global.css";
const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: {
    default: "Hack Club News",
    template: "%s | Hack Club News",
  },
  description:
    "Stay updated with the latest news, announcements, and stories from the Hack Club community. Read emails from Shiba, Summer of Making, and other Hack Club programs.",
  keywords: [
    "Hack Club",
    "news",
    "emails",
    "community",
    "programming",
    "education",
    "teenagers",
  ],
  authors: [{ name: "Hack Club" }],
  creator: "Hack Club",
  publisher: "Hack Club",
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      "max-video-preview": -1,
      "max-image-preview": "large",
      "max-snippet": -1,
    },
  },
  openGraph: {
    type: "website",
    locale: "en_US",
    url: "https://news.hackclub.com",
    siteName: "Hack Club News",
    title: "Hack Club News",
    description:
      "Stay updated with the latest news, announcements, and stories from the Hack Club community.",
    images: [
      {
        url: "/og-image.png",
        width: 1200,
        height: 630,
        alt: "Hack Club News",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "Hack Club News",
    description:
      "Stay updated with the latest news, announcements, and stories from the Hack Club community.",
    images: ["/og-image.png"],
  },
  verification: {
    google: "your-google-verification-code",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        {children}
      </body>
    </html>
  );
}
