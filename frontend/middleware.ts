import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

export function middleware(request: NextRequest) {
  // Get the pathname of the request (e.g. /dashboard, /auth/login)
  const { pathname } = request.nextUrl

  // Handle authentication routes
  if (pathname.startsWith('/dashboard')) {
    // Check if user is authenticated (in a real app, you'd validate the token)
    // For now, let the client-side handle auth checks
    return NextResponse.next()
  }

  // Handle API route proxying if needed
  if (pathname.startsWith('/api/')) {
    // Proxy API requests to backend if needed
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://13.127.59.135:9000/api/v1'
    const url = request.nextUrl.clone()
    url.href = `${apiUrl}${pathname.replace('/api', '')}`
    return NextResponse.redirect(url)
  }

  // Allow all other requests to proceed normally
  return NextResponse.next()
}

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     * - public files (public folder)
     */
    '/((?!_next/static|_next/image|favicon.ico|public/).*)',
  ],
}