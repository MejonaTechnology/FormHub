#!/bin/bash

# FormHub Backend Deployment Script for AWS EC2
# This script deploys the FormHub Go backend to the EC2 server

set -e

echo "🚀 Starting FormHub Backend Deployment..."

# Configuration
SERVER_IP="13.127.59.135"
SERVER_USER="ec2-user"
SSH_KEY="D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem"
APP_DIR="/opt/formhub"
SERVICE_NAME="formhub-api"

echo "📋 Deployment Configuration:"
echo "  Server: $SERVER_IP"
echo "  User: $SERVER_USER" 
echo "  App Directory: $APP_DIR"
echo "  Service: $SERVICE_NAME"

# Step 1: Build the Go application locally
echo "🔨 Building Go application for Linux..."
cd "D:\Mejona Workspace\Product\FormHub\backend"

# Build for Linux x86_64
GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o formhub-api main.go

if [ ! -f "formhub-api" ]; then
    echo "❌ Build failed - executable not found"
    exit 1
fi

echo "✅ Build completed successfully"

# Step 2: Create database setup SQL
echo "📝 Creating database setup script..."
cat > setup-database.sql << EOF
-- FormHub Database Setup
CREATE DATABASE IF NOT EXISTS formhub CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Create user if not exists
CREATE USER IF NOT EXISTS 'formhub_user'@'localhost' IDENTIFIED BY 'formhub_password';
GRANT ALL PRIVILEGES ON formhub.* TO 'formhub_user'@'localhost';
FLUSH PRIVILEGES;

USE formhub;

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    company VARCHAR(255),
    plan_type VARCHAR(50) DEFAULT 'free',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_email (email),
    INDEX idx_plan (plan_type)
);

-- Forms table
CREATE TABLE IF NOT EXISTS forms (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    target_email VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_active (is_active)
);

-- Submissions table
CREATE TABLE IF NOT EXISTS submissions (
    id VARCHAR(36) PRIMARY KEY,
    form_id VARCHAR(36),
    data JSON NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    spam_score FLOAT DEFAULT 0,
    processed BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE SET NULL,
    INDEX idx_form_id (form_id),
    INDEX idx_created_at (created_at),
    INDEX idx_processed (processed)
);

-- API Keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) UNIQUE NOT NULL,
    permissions JSON,
    rate_limit INT DEFAULT 100,
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_key_hash (key_hash),
    INDEX idx_active (is_active)
);

-- Sessions table for JWT blacklist
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_token_hash (token_hash),
    INDEX idx_expires_at (expires_at)
);

EOF

echo "✅ Database setup script created"

# Step 3: Create systemd service file
echo "🔧 Creating systemd service configuration..."
cat > formhub-api.service << EOF
[Unit]
Description=FormHub API Service
After=network.target mysql.service redis.service
Wants=mysql.service redis.service

[Service]
Type=simple
User=ec2-user
WorkingDirectory=$APP_DIR
ExecStart=$APP_DIR/formhub-api
EnvironmentFile=$APP_DIR/.env.production
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=formhub-api

# Security settings
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=$APP_DIR

[Install]
WantedBy=multi-user.target
EOF

echo "✅ Systemd service file created"

# Step 4: Upload files to server
echo "📤 Uploading files to server..."

# Create directory structure
ssh -i "$SSH_KEY" $SERVER_USER@$SERVER_IP "sudo mkdir -p $APP_DIR && sudo chown $SERVER_USER:$SERVER_USER $APP_DIR"

# Upload binary
scp -i "$SSH_KEY" formhub-api $SERVER_USER@$SERVER_IP:$APP_DIR/

# Upload environment file
scp -i "$SSH_KEY" .env.production $SERVER_USER@$SERVER_IP:$APP_DIR/

# Upload database setup
scp -i "$SSH_KEY" setup-database.sql $SERVER_USER@$SERVER_IP:$APP_DIR/

# Upload service file
scp -i "$SSH_KEY" formhub-api.service $SERVER_USER@$SERVER_IP:/tmp/

echo "✅ Files uploaded successfully"

# Step 5: Server setup and configuration
echo "🔧 Configuring server..."

ssh -i "$SSH_KEY" $SERVER_USER@$SERVER_IP << 'EOF'
    # Make binary executable
    chmod +x /opt/formhub/formhub-api
    
    # Install system dependencies if not present
    sudo yum update -y
    sudo yum install -y mysql redis
    
    # Start and enable MySQL and Redis
    sudo systemctl start mysqld redis
    sudo systemctl enable mysqld redis
    
    # Setup database
    echo "Setting up database..."
    mysql -u root < /opt/formhub/setup-database.sql
    
    # Install systemd service
    sudo cp /tmp/formhub-api.service /etc/systemd/system/
    sudo systemctl daemon-reload
    
    # Stop any existing service
    sudo systemctl stop formhub-api || true
    
    # Start and enable service
    sudo systemctl enable formhub-api
    sudo systemctl start formhub-api
    
    # Check service status
    sudo systemctl status formhub-api --no-pager
    
    echo "✅ Server configuration completed"
EOF

# Step 6: Verify deployment
echo "🧪 Verifying deployment..."

# Wait a moment for service to start
sleep 5

# Test health endpoint
if curl -f --connect-timeout 10 http://$SERVER_IP:9000/health; then
    echo "✅ Health check passed"
else
    echo "❌ Health check failed"
    
    # Show service logs for debugging
    echo "📋 Service logs:"
    ssh -i "$SSH_KEY" $SERVER_USER@$SERVER_IP "sudo journalctl -u formhub-api --no-pager -n 20"
    exit 1
fi

# Test API endpoint
echo "Testing API endpoint..."
if curl -f -X POST http://$SERVER_IP:9000/api/v1/submit -H "Content-Type: application/json" -d '{"test": true}'; then
    echo "✅ API endpoint test passed"
else
    echo "⚠️  API endpoint test failed (may be due to missing data - check logs)"
fi

echo "🎉 FormHub Backend Deployment Completed Successfully!"
echo ""
echo "📋 Deployment Summary:"
echo "  🌐 API URL: http://$SERVER_IP:9000"
echo "  🏥 Health Check: http://$SERVER_IP:9000/health"  
echo "  📝 Submit Endpoint: http://$SERVER_IP:9000/api/v1/submit"
echo "  🔐 Auth Endpoints: http://$SERVER_IP:9000/api/v1/auth/*"
echo ""
echo "🔧 Management Commands:"
echo "  sudo systemctl status formhub-api"
echo "  sudo systemctl restart formhub-api"
echo "  sudo journalctl -u formhub-api -f"

# Cleanup local files
rm -f formhub-api setup-database.sql formhub-api.service

echo "✅ Deployment completed and cleanup done!"