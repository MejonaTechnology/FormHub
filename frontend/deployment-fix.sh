#!/bin/bash

# FormHub Frontend Deployment Fix Script
# This script fixes the deployment issues on the EC2 server

echo "🚀 Starting FormHub Frontend Deployment Fix..."

# Set environment variables
export NODE_ENV=production
export NEXT_PUBLIC_API_URL=http://13.127.59.135:9000/api/v1

# Navigate to the frontend directory
cd /opt/mejona/formhub-frontend

echo "📦 Installing dependencies..."
npm install --production=false

echo "🧹 Cleaning previous build..."
rm -rf .next
rm -rf out

echo "🔨 Building the application..."
npm run deploy:build

echo "🔍 Checking build output..."
if [ ! -d ".next" ]; then
    echo "❌ Build failed - .next directory not found"
    exit 1
fi

echo "✅ Build successful!"

echo "🔄 Restarting the frontend service..."
sudo systemctl stop formhub-frontend
sudo systemctl start formhub-frontend
sudo systemctl status formhub-frontend

echo "🔍 Checking if the service is running..."
sleep 3
if curl -f http://localhost:3000 > /dev/null 2>&1; then
    echo "✅ Frontend is running on port 3000"
else
    echo "❌ Frontend is not responding on port 3000"
    echo "📋 Service logs:"
    sudo journalctl -u formhub-frontend --no-pager -n 20
fi

echo "🌐 Testing routes..."
echo "Testing homepage..."
curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/
echo

echo "Testing dashboard..."
curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/dashboard
echo

echo "Testing login..."
curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/auth/login
echo

echo "🎉 Deployment fix completed!"
echo "🔗 Frontend should now be accessible at: http://13.127.59.135:3000"