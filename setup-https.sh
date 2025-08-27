#!/bin/bash

# FormHub HTTPS Setup Script
# This script sets up Caddy as a reverse proxy to enable HTTPS for the FormHub API

set -e

echo "🔒 Setting up HTTPS for FormHub Backend..."

SERVER_IP="13.127.59.135"
SERVER_USER="ec2-user"
SSH_KEY="D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem"
DOMAIN="formhub-api.mejonatech.com"  # We'll need to configure DNS for this

echo "📋 HTTPS Configuration:"
echo "  Server: $SERVER_IP"
echo "  Domain: $DOMAIN"
echo "  Backend Port: 9000"
echo "  HTTPS Port: 443"

# Step 1: Install Caddy on the server
echo "📦 Installing Caddy server..."

ssh -i "$SSH_KEY" $SERVER_USER@$SERVER_IP << 'EOF'
    # Add Caddy repository
    sudo dnf install -y 'dnf-command(copr)'
    sudo dnf copr enable -y @caddy/caddy
    
    # Install Caddy
    sudo dnf install -y caddy
    
    # Enable and start Caddy
    sudo systemctl enable caddy
EOF

# Step 2: Create Caddyfile configuration
echo "🔧 Creating Caddy configuration..."

cat > Caddyfile << EOF
# Caddy configuration for FormHub API
{
    # Global options
    email admin@mejonatech.com
    auto_https on
}

# HTTPS endpoint for FormHub API
$DOMAIN {
    # Reverse proxy to local FormHub API
    reverse_proxy localhost:9000
    
    # CORS headers for GitHub Pages
    header Access-Control-Allow-Origin "https://mejonatechnology.github.io"
    header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS"
    header Access-Control-Allow-Headers "Origin, Content-Type, Accept, Authorization, X-API-Key"
    header Access-Control-Allow-Credentials "true"
    header Access-Control-Max-Age "43200"
    
    # Security headers
    header Strict-Transport-Security "max-age=31536000; includeSubDomains"
    header X-Content-Type-Options "nosniff"
    header X-Frame-Options "DENY"
    header X-XSS-Protection "1; mode=block"
    
    # Handle preflight requests
    @options {
        method OPTIONS
    }
    respond @options 204
    
    # Logging
    log {
        output file /var/log/caddy/formhub-api.log
        format json
    }
}

# Fallback configuration using IP (for testing without domain)
$SERVER_IP:8443 {
    reverse_proxy localhost:9000
    
    # Same CORS and security headers
    header Access-Control-Allow-Origin "https://mejonatechnology.github.io"
    header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS"
    header Access-Control-Allow-Headers "Origin, Content-Type, Accept, Authorization, X-API-Key"
    header Access-Control-Allow-Credentials "true"
    
    @options {
        method OPTIONS
    }
    respond @options 204
    
    # Self-signed certificate for IP access
    tls internal
}
EOF

# Step 3: Upload and configure Caddy
echo "📤 Uploading Caddy configuration..."

# Upload Caddyfile
scp -i "$SSH_KEY" Caddyfile $SERVER_USER@$SERVER_IP:/tmp/

# Configure and start Caddy
ssh -i "$SSH_KEY" $SERVER_USER@$SERVER_IP << 'EOF'
    # Move Caddyfile to proper location
    sudo mv /tmp/Caddyfile /etc/caddy/
    sudo chown caddy:caddy /etc/caddy/Caddyfile
    sudo chmod 644 /etc/caddy/Caddyfile
    
    # Create log directory
    sudo mkdir -p /var/log/caddy
    sudo chown caddy:caddy /var/log/caddy
    
    # Test configuration
    sudo caddy validate --config /etc/caddy/Caddyfile
    
    # Restart Caddy with new configuration
    sudo systemctl restart caddy
    sudo systemctl status caddy --no-pager
    
    echo "✅ Caddy configuration completed"
EOF

# Step 4: Test HTTPS endpoints
echo "🧪 Testing HTTPS endpoints..."

sleep 5  # Wait for Caddy to start

# Test IP-based HTTPS (self-signed)
echo "Testing self-signed HTTPS endpoint..."
if curl -k --connect-timeout 10 https://$SERVER_IP:8443/health; then
    echo "✅ Self-signed HTTPS endpoint working"
else
    echo "❌ Self-signed HTTPS endpoint failed"
fi

# Step 5: Update firewall rules
echo "🔥 Configuring firewall..."

ssh -i "$SSH_KEY" $SERVER_USER@$SERVER_IP << 'EOF'
    # Check if firewall is active
    if sudo systemctl is-active --quiet firewalld; then
        echo "Configuring firewalld..."
        sudo firewall-cmd --permanent --add-port=443/tcp
        sudo firewall-cmd --permanent --add-port=8443/tcp
        sudo firewall-cmd --reload
    else
        echo "Firewalld not active, checking iptables or using AWS Security Groups"
    fi
    
    # Show open ports
    sudo netstat -tlnp | grep -E ":(443|8443|9000)"
EOF

echo "🎉 HTTPS Setup Completed!"
echo ""
echo "📋 Available Endpoints:"
echo "  🔒 HTTPS API (Self-signed): https://$SERVER_IP:8443"
echo "  🌐 Domain HTTPS (when DNS configured): https://$DOMAIN"
echo "  🔓 HTTP API (Legacy): http://$SERVER_IP:9000"
echo ""
echo "🔧 Next Steps:"
echo "  1. Configure DNS: $DOMAIN -> $SERVER_IP"
echo "  2. Update frontend to use HTTPS endpoint"
echo "  3. Test all functionality with HTTPS"

# Cleanup
rm -f Caddyfile

echo "✅ HTTPS setup script completed!"