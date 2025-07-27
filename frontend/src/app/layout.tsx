import './globals.css'
import type { Metadata } from 'next'
import { Inter } from 'next/font/google'

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: 'KanOpt - Agentic Kanban Sprint Optimizer',
  description: 'AI-powered Kanban board with predictive analytics and autonomous task optimization',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body className={inter.className}>
        <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100">
          <header className="bg-white/80 backdrop-blur-sm border-b border-gray-200 sticky top-0 z-50">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
              <div className="flex justify-between items-center h-16">
                <div className="flex items-center">
                  <h1 className="text-2xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
                    KanOpt
                  </h1>
                  <span className="ml-2 px-2 py-1 text-xs bg-green-100 text-green-800 rounded-full">
                    AI-Powered
                  </span>
                </div>
                <nav className="flex items-center space-x-4">
                  <button className="text-gray-600 hover:text-gray-900 text-sm font-medium">
                    Dashboard
                  </button>
                  <button className="text-gray-600 hover:text-gray-900 text-sm font-medium">
                    Analytics
                  </button>
                  <button className="text-gray-600 hover:text-gray-900 text-sm font-medium">
                    Settings
                  </button>
                  <div className="w-8 h-8 bg-gradient-to-r from-blue-500 to-purple-500 rounded-full"></div>
                </nav>
              </div>
            </div>
          </header>
          <main className="flex-1">
            {children}
          </main>
        </div>
      </body>
    </html>
  )
}
