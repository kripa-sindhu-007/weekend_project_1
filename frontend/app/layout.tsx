import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Task Queue Dashboard",
  description: "Monitor and manage the concurrent task queue",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
