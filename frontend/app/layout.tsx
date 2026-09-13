import type { Metadata, Viewport } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'Vaultory',
  description: 'Your collection, kept properly.',
}

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  // Both appearances declared, so the browser chrome matches whichever the collector's system
  // prefers (FR-046).
  themeColor: [
    { media: '(prefers-color-scheme: dark)', color: 'rgb(12 12 16)' },
    { media: '(prefers-color-scheme: light)', color: 'rgb(248 247 245)' },
  ],
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  )
}
