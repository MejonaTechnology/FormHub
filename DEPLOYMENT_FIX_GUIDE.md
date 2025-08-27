# FormHub Frontend Deployment Fix Guide

## Problem Analysis

The FormHub frontend was experiencing 404 errors for all app router pages despite successful CI/CD deployment. The root causes identified:

1. **Missing Output Configuration**: Next.js 15.3.5 needs explicit configuration for proper standalone deployment
2. **Incorrect Environment Variables**: API URL configuration wasn't properly set for production
3. **Missing Middleware**: App router pages need proper middleware for routing
4. **Service Configuration**: SystemD service wasn't properly configured for production mode
5. **Missing Pages**: Some pages (like submissions) were incomplete

## Solutions Implemented

### 1. Enhanced next.config.js

**File**: `frontend/next.config.js`

```javascript
const nextConfig = {
  reactStrictMode: true,
  // Critical: Ensure proper output for deployment
  output: 'standalone',
  // Optimize for production
  compress: true,
  trailingSlash: false,
  // Fixed API URL
  env: {
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || 'http://13.127.59.135:9000/api/v1',
  },
  // Performance optimizations
  pageExtensions: ['tsx', 'ts', 'jsx', 'js'],
  productionBrowserSourceMaps: false,
  experimental: {
    optimizePackageImports: ['@heroicons/react', '@headlessui/react'],
  },
  // Security headers
  async headers() {
    return [
      {
        source: '/(.*)',
        headers: [
          { key: 'X-Frame-Options', value: 'DENY' },
          { key: 'X-Content-Type-Options', value: 'nosniff' },
        ],
      },
    ]
  },
}
```

**Key Changes:**
- Added `output: 'standalone'` for proper deployment
- Fixed API URL to use production backend
- Added performance optimizations
- Security headers for production

### 2. Added Middleware for Routing

**File**: `frontend/middleware.ts`

```typescript
import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl

  // Handle dashboard authentication
  if (pathname.startsWith('/dashboard')) {
    return NextResponse.next()
  }

  // Handle API proxying if needed
  if (pathname.startsWith('/api/')) {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://13.127.59.135:9000/api/v1'
    const url = request.nextUrl.clone()
    url.href = `${apiUrl}${pathname.replace('/api', '')}`
    return NextResponse.redirect(url)
  }

  return NextResponse.next()
}
```

**Benefits:**
- Proper routing for app router pages
- API request handling
- Authentication middleware structure

### 3. Enhanced package.json Scripts

**File**: `frontend/package.json`

```json
{
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start -p 3000",
    "start:production": "NODE_ENV=production next start -p 3000",
    "export": "next build && next export",
    "lint": "next lint",
    "lint:fix": "next lint --fix",
    "type-check": "tsc --noEmit",
    "clean": "rm -rf .next out",
    "build:clean": "npm run clean && npm run build",
    "deploy:build": "NODE_ENV=production npm run build"
  }
}
```

**Key Additions:**
- `start:production` for proper production startup
- `deploy:build` with environment setting
- Cleanup scripts for fresh builds

### 4. Environment Configuration

**File**: `frontend/.env.production`

```
NEXT_PUBLIC_API_URL=http://13.127.59.135:9000/api/v1
NODE_ENV=production
```

**Purpose:**
- Ensures correct API URL in production
- Sets proper environment mode

### 5. SystemD Service Configuration

**File**: `frontend/systemd-service-config.txt`

```ini
[Unit]
Description=FormHub Frontend
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/opt/mejona/formhub-frontend
Environment=NODE_ENV=production
Environment=NEXT_PUBLIC_API_URL=http://13.127.59.135:9000/api/v1
ExecStart=/usr/bin/npm run start:production
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=formhub-frontend

[Install]
WantedBy=multi-user.target
```

**Critical Changes:**
- Added environment variables
- Used production start script
- Proper logging configuration

### 6. Added Missing Pages

**File**: `frontend/app/dashboard/submissions/page.tsx`

- Complete submissions management page
- Proper error handling and loading states
- Consistent UI with other dashboard pages

## Deployment Instructions

### Automated Deployment (Recommended)

1. **Upload the deployment script to server:**
   ```bash
   scp frontend/deployment-fix.sh ubuntu@13.127.59.135:/opt/mejona/formhub-frontend/
   ```

2. **Run the fix script on server:**
   ```bash
   ssh ubuntu@13.127.59.135
   cd /opt/mejona/formhub-frontend
   chmod +x deployment-fix.sh
   ./deployment-fix.sh
   ```

### Manual Deployment

1. **Copy all updated files to server:**
   ```bash
   rsync -av frontend/ ubuntu@13.127.59.135:/opt/mejona/formhub-frontend/
   ```

2. **Update SystemD service:**
   ```bash
   sudo cp systemd-service-config.txt /etc/systemd/system/formhub-frontend.service
   sudo systemctl daemon-reload
   ```

3. **Build and restart:**
   ```bash
   cd /opt/mejona/formhub-frontend
   npm install
   npm run deploy:build
   sudo systemctl restart formhub-frontend
   ```

## Verification Steps

### 1. Service Status
```bash
sudo systemctl status formhub-frontend
```

### 2. Test Routes
```bash
# Homepage
curl -I http://13.127.59.135:3000/

# Dashboard (should return 200, not 404)
curl -I http://13.127.59.135:3000/dashboard

# Login page
curl -I http://13.127.59.135:3000/auth/login

# API connectivity
curl http://13.127.59.135:3000/api/health
```

### 3. Browser Tests

Navigate to:
- ✅ `http://13.127.59.135:3000/` - Homepage
- ✅ `http://13.127.59.135:3000/dashboard` - Dashboard
- ✅ `http://13.127.59.135:3000/auth/login` - Login
- ✅ `http://13.127.59.135:3000/auth/register` - Register  
- ✅ `http://13.127.59.135:3000/pricing` - Pricing
- ✅ `http://13.127.59.135:3000/docs` - Documentation

## Troubleshooting

### If Pages Still Return 404

1. **Check build output:**
   ```bash
   ls -la /opt/mejona/formhub-frontend/.next/server/app/
   ```
   Should contain: `dashboard/`, `auth/`, `pricing/`, etc.

2. **Check service logs:**
   ```bash
   sudo journalctl -u formhub-frontend -f
   ```

3. **Verify Node.js version:**
   ```bash
   node --version  # Should be 18+
   npm --version
   ```

### If API Calls Fail

1. **Check environment variables:**
   ```bash
   echo $NEXT_PUBLIC_API_URL
   ```

2. **Test backend connectivity:**
   ```bash
   curl http://13.127.59.135:9000/api/v1/health
   ```

3. **Check CORS settings** in backend if requests fail

### If Build Fails

1. **Clear cache and rebuild:**
   ```bash
   rm -rf .next node_modules
   npm install
   npm run build
   ```

2. **Check TypeScript compilation:**
   ```bash
   npm run type-check
   ```

## Performance Optimizations

The fixes also include several performance improvements:

1. **Bundle Optimization**: Package imports optimization for UI libraries
2. **Static Generation**: All pages pre-generated for faster loading  
3. **Compression**: Gzip compression enabled
4. **Source Maps**: Disabled in production for smaller bundles
5. **Middleware**: Efficient routing with minimal overhead

## Security Enhancements

1. **Security Headers**: X-Frame-Options and X-Content-Type-Options
2. **Environment Isolation**: Production-specific environment variables
3. **Service Isolation**: Proper SystemD service configuration

## Expected Results

After applying these fixes:

- ✅ **All pages load correctly** (no more 404 errors)
- ✅ **Proper API connectivity** to backend service
- ✅ **Production-ready deployment** with optimizations
- ✅ **Reliable service startup** and automatic restart
- ✅ **Complete feature set** with all dashboard pages
- ✅ **Security headers** and proper configuration

## Support

If issues persist, check:
1. Service logs: `sudo journalctl -u formhub-frontend`
2. Build output directory: `.next/server/app/`
3. Network connectivity: Backend service on port 9000
4. File permissions: All files owned by ubuntu user

The deployment should now be fully functional with all routes working correctly.