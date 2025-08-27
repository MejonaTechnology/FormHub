/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // GitHub Pages requires static export
  output: 'export',
  // Disable server features for static export
  trailingSlash: true,
  skipTrailingSlashRedirect: true,
  // Set base path for GitHub Pages (will be updated if using custom domain)
  basePath: process.env.NODE_ENV === 'production' ? '/FormHub' : '',
  assetPrefix: process.env.NODE_ENV === 'production' ? '/FormHub/' : '',
  // Environment variables
  env: {
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || 'http://13.127.59.135:9000/api/v1',
  },
  // Ensure proper page extensions
  pageExtensions: ['tsx', 'ts', 'jsx', 'js'],
  // Disable source maps in production
  productionBrowserSourceMaps: false,
  // Ensure proper static optimization
  experimental: {
    optimizePackageImports: ['@heroicons/react', '@headlessui/react'],
  },
  // Configure redirects to handle potential routing issues
  async redirects() {
    return [
      {
        source: '/dashboard',
        destination: '/dashboard',
        permanent: false,
      },
    ]
  },
  // Configure headers for proper serving
  async headers() {
    return [
      {
        source: '/(.*)',
        headers: [
          {
            key: 'X-Frame-Options',
            value: 'DENY',
          },
          {
            key: 'X-Content-Type-Options',
            value: 'nosniff',
          },
        ],
      },
    ]
  },
}

module.exports = nextConfig