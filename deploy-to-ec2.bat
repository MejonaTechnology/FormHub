@echo off
echo ========================================
echo    FormHub EC2 Deployment
echo ========================================
echo.

set EC2_HOST=13.127.59.135
set EC2_USER=ec2-user
set KEY_PATH="D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem"

echo Deploying FormHub to AWS EC2...
echo Server: %EC2_HOST%
echo User: %EC2_USER%
echo Key: %KEY_PATH%
echo.

echo [1/6] Testing EC2 connection...
ssh -i %KEY_PATH% -o ConnectTimeout=10 %EC2_USER%@%EC2_HOST% "echo 'Connection successful'"
if %errorlevel% neq 0 (
    echo ERROR: Cannot connect to EC2 server
    echo Please check:
    echo 1. Security group allows SSH (port 22)
    echo 2. Key file permissions
    echo 3. Server is running
    pause
    exit /b 1
)
echo ✓ EC2 connection successful
echo.

echo [2/6] Creating deployment directory on server...
ssh -i %KEY_PATH% %EC2_USER%@%EC2_HOST% "sudo mkdir -p /opt/formhub && sudo chown -R ec2-user:ec2-user /opt/formhub"
echo ✓ Deployment directory created
echo.

echo [3/6] Copying FormHub files to server...
echo This may take a few minutes...
scp -i %KEY_PATH% -r backend frontend docker-compose.yml deploy scripts %EC2_USER%@%EC2_HOST%:/opt/formhub/
echo ✓ Files copied successfully
echo.

echo [4/6] Setting up environment configuration...
ssh -i %KEY_PATH% %EC2_USER%@%EC2_HOST% "cd /opt/formhub && cp backend/.env backend/.env.backup"
echo ✓ Environment backed up
echo.

echo [5/6] Installing Docker on EC2 (if not installed)...
ssh -i %KEY_PATH% %EC2_USER%@%EC2_HOST% "
sudo yum update -y &&
sudo yum install -y docker &&
sudo systemctl start docker &&
sudo systemctl enable docker &&
sudo usermod -a -G docker ec2-user &&
sudo curl -L 'https://github.com/docker/compose/releases/download/1.29.2/docker-compose-$(uname -s)-$(uname -m)' -o /usr/local/bin/docker-compose &&
sudo chmod +x /usr/local/bin/docker-compose
"
echo ✓ Docker installation completed
echo.

echo [6/6] Starting FormHub services...
ssh -i %KEY_PATH% %EC2_USER%@%EC2_HOST% "cd /opt/formhub && docker-compose up -d"
echo ✓ FormHub services started
echo.

echo ========================================
echo    FormHub Deployed Successfully!
echo ========================================
echo.
echo Your FormHub is now running at:
echo ✓ API: http://%EC2_HOST%:8080/health
echo ✓ Frontend: http://%EC2_HOST%:3000
echo.
echo IMPORTANT: Configure Security Groups
echo 1. Allow port 8080 (API) from 0.0.0.0/0
echo 2. Allow port 3000 (Frontend) from 0.0.0.0/0  
echo 3. Allow port 80/443 for domain setup
echo.
echo Testing API endpoint...
curl -f http://%EC2_HOST%:8080/health
echo.
echo.
echo Next steps:
echo 1. Configure security groups in AWS
echo 2. Test: http://%EC2_HOST%:8080/health
echo 3. Set up domain (optional)
echo 4. Configure SSL (optional)
echo.
pause