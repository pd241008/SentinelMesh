import type { Metadata } from "next"
import "./globals.css"

export const metadata: Metadata = {
  title: "SentinelMesh",
  description: "Gossip-Propagated Collective Anomaly Detection",
}

const navLinks = [
  { href: "/", label: "Sweep Overview" },
  { href: "/crosscheck", label: "ML Crosscheck" },
]

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className="dark">
      <body className="min-h-screen bg-gray-950 text-gray-100">
        <header className="border-b border-gray-800 px-6 py-3">
          <div className="mx-auto flex max-w-7xl items-center justify-between">
            <span className="text-lg font-bold tracking-tight">SentinelMesh</span>
            <nav className="flex gap-6 text-sm">
              {navLinks.map((link) => (
                <a key={link.href} href={link.href} className="text-gray-400 hover:text-white transition-colors">
                  {link.label}
                </a>
              ))}
            </nav>
          </div>
        </header>
        <main className="mx-auto max-w-7xl px-6 py-8">{children}</main>
      </body>
    </html>
  )
}
