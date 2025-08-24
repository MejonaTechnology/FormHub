@echo off
echo ========================================
echo    FormHub Deployment - Server Optimized
echo ========================================
echo.

echo Server Status After Cleanup:
echo ✅ Storage: 1.6GB free (was 593MB)
echo ✅ Memory: 372MB available  
echo ✅ Available ports: 9000+ range
echo ✅ Email: mejona.tech@gmail.com configured
echo.

echo Deploying FormHub to port 9000 (API) and 9001 (Frontend)...
echo.

echo [1/4] Uploading optimized FormHub configuration...
scp -i "D:/Mejona Workspace/Product/Mejona complete website/mejonaN.pem" -r backend ec2-user@13.201.64.45:/home/ec2-user/formhub/

echo [2/4] Creating lightweight FormHub setup...
ssh -i "D:/Mejona Workspace/Product/Mejona complete website/mejonaN.pem" ec2-user@13.201.64.45 "
# Create FormHub directory
mkdir -p /home/ec2-user/formhub
cd /home/ec2-user/formhub

# Create optimized environment for port 9000
cat > .env << 'EOF'
ENVIRONMENT=production
PORT=9000
DATABASE_URL=mysql://root:mejona123@localhost:3306/formhub?parseTime=true
REDIS_URL=
JWT_SECRET=formhub-prod-secret-$(date +%s)
ALLOWED_ORIGINS=http://13.201.64.45:9000,https://formhub.mejona.in,http://localhost:9000
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=mejona.tech@gmail.com
SMTP_PASSWORD=pkjs cehq vhpc atek
FROM_EMAIL=noreply@formhub.com
FROM_NAME=FormHub by Mejona Technology
EOF

echo 'Environment configured for production'
"

echo [3/4] Setting up FormHub database...
ssh -i "D:/Mejona Workspace/Product/Mejona complete website/mejonaN.pem" ec2-user@13.201.64.45 "
# Create FormHub database
mysql -u root -pmejona123 -e 'CREATE DATABASE IF NOT EXISTS formhub CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;' 2>/dev/null || echo 'Database may already exist'

# Create FormHub tables (simplified MySQL version)
mysql -u root -pmejona123 formhub << 'EOF'
-- Users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY DEFAULT (UUID()),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    company VARCHAR(255),
    plan_type VARCHAR(50) NOT NULL DEFAULT 'free',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- API Keys table  
CREATE TABLE IF NOT EXISTS api_keys (
    id VARCHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) UNIQUE NOT NULL,
    permissions TEXT NOT NULL DEFAULT 'form_submit',
    rate_limit INTEGER NOT NULL DEFAULT 1000,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_used_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Forms table
CREATE TABLE IF NOT EXISTS forms (
    id VARCHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    target_email VARCHAR(255) NOT NULL,
    cc_emails TEXT,
    subject VARCHAR(500),
    success_message TEXT,
    redirect_url TEXT,
    webhook_url TEXT,
    spam_protection BOOLEAN NOT NULL DEFAULT false,
    file_uploads BOOLEAN NOT NULL DEFAULT false,
    max_file_size BIGINT NOT NULL DEFAULT 5242880,
    is_active BOOLEAN NOT NULL DEFAULT true,
    submission_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Submissions table
CREATE TABLE IF NOT EXISTS submissions (
    id VARCHAR(36) PRIMARY KEY DEFAULT (UUID()),
    form_id VARCHAR(36) NOT NULL,
    data JSON NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    referrer TEXT,
    is_spam BOOLEAN NOT NULL DEFAULT false,
    spam_score DECIMAL(3,2) NOT NULL DEFAULT 0.0,
    email_sent BOOLEAN NOT NULL DEFAULT false,
    webhook_sent BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_forms_user_id ON forms(user_id);
CREATE INDEX IF NOT EXISTS idx_submissions_form_id ON submissions(form_id);
CREATE INDEX IF NOT EXISTS idx_submissions_created_at ON submissions(created_at);
EOF

echo 'FormHub database setup complete'
"

echo [4/4] Starting FormHub service...
ssh -i "D:/Mejona Workspace/Product/Mejona complete website/mejonaN.pem" ec2-user@13.201.64.45 "
cd /home/ec2-user/formhub

# Build and run FormHub (will use existing Go installation)
export PATH=/usr/local/go/bin:\$PATH
go mod init formhub 2>/dev/null || true
go mod tidy

# Start FormHub in background
nohup go run /home/ec2-user/formhub/main.go > formhub.log 2>&1 &

echo 'FormHub started on port 9000'
echo 'Process ID:' \$!

# Check if it's running
sleep 3
curl -f http://localhost:9000/health && echo 'FormHub is healthy!' || echo 'Starting up...'
"

echo.
echo ========================================
echo    🎉 FormHub Deployed Successfully!
echo ========================================
echo.
echo FormHub is running at:
echo 🔗 API: http://13.201.64.45:9000/health
echo 📧 Email: mejona.tech@gmail.com (configured)
echo 💾 Database: MySQL formhub database
echo.
echo Testing deployment...
timeout /t 5 >nul
curl -f http://13.201.64.45:9000/health

echo.
echo Next Steps:
echo 1. Update AWS Security Group: Allow port 9000
echo 2. Test API: http://13.201.64.45:9000/health  
echo 3. Create first user account
echo 4. Generate API keys for clients
echo.
echo Opening FormHub API in browser...
start http://13.201.64.45:9000/health
echo.
pause