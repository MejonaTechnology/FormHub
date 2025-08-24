import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import { Toaster } from 'react-hot-toast'

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: 'FormHub - Form Backend Service',
  description: 'Professional form backend service for static websites and modern applications',
  keywords: ['form backend', 'contact forms', 'static website forms', 'form API'],
  authors: [{ name: 'Mejona Technology LLP' }],
  creator: 'Mejona Technology LLP',
  publisher: 'Mejona Technology LLP',
  robots: 'index, follow',
  openGraph: {
    type: 'website',
    locale: 'en_US',
    url: 'https://formhub.com',
    title: 'FormHub - Form Backend Service',
    description: 'Professional form backend service for static websites and modern applications',
    siteName: 'FormHub',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'FormHub - Form Backend Service',
    description: 'Professional form backend service for static websites and modern applications',
  },
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body className={`${inter.className} antialiased bg-gray-50 min-h-screen`}>
        <div id="root">
          {children}
        </div>
        <Toaster 
          position="top-right"
          toastOptions={{
            duration: 4000,
            style: {
              background: '#fff',
              color: '#333',
              borderRadius: '10px',
              border: '1px solid #e5e7eb',
              boxShadow: '0 10px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04)',
            },
            success: {
              iconTheme: {
                primary: '#10b981',
                secondary: '#fff',
              },
            },
            error: {
              iconTheme: {
                primary: '#ef4444',
                secondary: '#fff',
              },
            },
          }}
        />
      </body>
    </html>
  )
}