/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // GitHub Pages requires static export
  output: 'export',
  // Critical settings for GitHub Pages static export
  trailingSlash: true,
  skipTrailingSlashRedirect: true,
  distDir: 'out',
  // Set base path for GitHub Pages
  basePath: process.env.NODE_ENV === 'production' ? '/FormHub' : '',
  assetPrefix: process.env.NODE_ENV === 'production' ? '/FormHub/' : '',
  // Environment variables
  env: {
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || 'https://formhub.mejona.in/api/v1',
    NEXT_PUBLIC_BASE_PATH: process.env.NODE_ENV === 'production' ? '/FormHub' : '',
  },
  // Ensure proper page extensions
  pageExtensions: ['tsx', 'ts', 'jsx', 'js'],
  // Disable source maps in production for GitHub Pages
  productionBrowserSourceMaps: false,
  // Image optimization for static export
  images: {
    unoptimized: true
  },
  // Ensure proper static optimization
  experimental: {
    optimizePackageImports: ['@heroicons/react', '@headlessui/react'],
  },
}

module.exports = nextConfig