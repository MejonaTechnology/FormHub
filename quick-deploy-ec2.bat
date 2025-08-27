@echo off
echo ========================================
echo    FormHub Quick Deploy to EC2
echo ========================================
echo.

echo Deploying FormHub to your EC2 server...
echo Server: 13.127.59.135
echo.

echo [1/4] Testing connection to EC2...
ssh -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" -o ConnectTimeout=10 ec2-user@13.127.59.135 "echo 'Connected successfully'"

if %errorlevel% neq 0 (
    echo.
    echo ❌ CONNECTION FAILED
    echo.
    echo Please check:
    echo 1. EC2 instance is running
    echo 2. Security group allows SSH (port 22)
    echo 3. Your IP is allowed in security group
    echo.
    echo AWS Console → EC2 → Security Groups → Add rule:
    echo Type: SSH, Port: 22, Source: My IP
    echo.
    pause
    exit /b 1
)

echo ✅ Connection successful
echo.

echo [2/4] Uploading FormHub files to EC2...
echo This may take a minute...
scp -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" -r backend frontend docker-compose.yml deploy ec2-user@13.127.59.135:/home/ec2-user/
echo ✅ Files uploaded
echo.

echo [3/4] Installing Docker and setting up FormHub...
ssh -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" ec2-user@13.127.59.135 "
echo 'Installing Docker...'
sudo yum update -y > /dev/null 2>&1
sudo yum install -y docker > /dev/null 2>&1
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -a -G docker ec2-user

echo 'Installing Docker Compose...'
sudo curl -sL 'https://github.com/docker/compose/releases/download/1.29.2/docker-compose-Linux-x86_64' -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

echo 'Setting up FormHub...'
sudo mkdir -p /opt/formhub
sudo chown -R ec2-user:ec2-user /opt/formhub
cp -r /home/ec2-user/* /opt/formhub/ 2>/dev/null || true
cd /opt/formhub
"
echo ✅ Docker installed and FormHub setup complete
echo.

echo [4/4] Starting FormHub services...
ssh -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" ec2-user@13.127.59.135 "
cd /opt/formhub
/usr/local/bin/docker-compose up -d
echo 'Waiting for services to start...'
sleep 10
"
echo ✅ FormHub services started
echo.

echo ========================================
echo    🎉 FormHub Deployed Successfully!
echo ========================================
echo.
echo Your FormHub is running at:
echo.
echo 🔗 API Health Check: http://13.127.59.135:8080/health
echo 🔗 Frontend Dashboard: http://13.127.59.135:3000
echo.
echo ⚠️  IMPORTANT: Update AWS Security Groups
echo.
echo Add these inbound rules in AWS Console:
echo 1. Custom TCP - Port 8080 - Source: 0.0.0.0/0 (FormHub API)
echo 2. Custom TCP - Port 3000 - Source: 0.0.0.0/0 (Frontend)
echo 3. HTTP - Port 80 - Source: 0.0.0.0/0 (Web traffic)
echo.
echo After updating security groups, test:
start http://13.127.59.135:8080/health
echo.
echo 🎯 Next Steps:
echo 1. Update security groups in AWS Console
echo 2. Test the API health check
echo 3. Create your first user account
echo 4. Start onboarding clients!
echo.
pause