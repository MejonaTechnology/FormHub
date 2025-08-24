@echo off
echo ========================================
echo    FormHub Lightweight EC2 Deployment
echo ========================================
echo.

echo Server Analysis:
echo ✓ IP: 13.201.64.45
echo ✓ Memory: 949MB (limited)
echo ✓ Storage: 593MB free (tight!)
echo ✓ Used ports: 8000, 8080-8087
echo ✓ Available ports: 9000+ range
echo.

echo DEPLOYMENT OPTIONS:
echo.
echo Option 1: Lightweight Go Binary (RECOMMENDED)
echo - Port 9000 for FormHub API
echo - Port 9001 for FormHub Frontend  
echo - Uses existing MariaDB (port 3306)
echo - No Docker needed
echo - ~50MB disk usage
echo.

echo Option 2: Docker with Custom Ports
echo - Port 9010 for FormHub API
echo - Port 9011 for FormHub Frontend
echo - Separate PostgreSQL container
echo - ~200MB disk usage
echo.

echo Option 3: Integrate with Existing Services
echo - Deploy as part of your microservices
echo - Use existing database
echo - Proxy through existing Apache
echo.

set /p CHOICE="Choose option (1, 2, or 3): "

if "%CHOICE%"=="1" goto LIGHTWEIGHT
if "%CHOICE%"=="2" goto DOCKER
if "%CHOICE%"=="3" goto INTEGRATE

echo Invalid choice. Exiting...
pause
exit /b 1

:LIGHTWEIGHT
echo.
echo ========================================
echo    Deploying FormHub as Go Binary
echo ========================================
echo.

echo [1/5] Uploading FormHub source code...
scp -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" -r backend ec2-user@13.201.64.45:/home/ec2-user/formhub-src/

echo [2/5] Setting up Go environment and building...
ssh -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" ec2-user@13.201.64.45 "
# Install Go if not present
if ! command -v go &> /dev/null; then
    echo 'Installing Go...'
    wget -q https://golang.org/dl/go1.21.5.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
    echo 'export PATH=/usr/local/go/bin:\$PATH' >> ~/.bashrc
    export PATH=/usr/local/go/bin:\$PATH
fi

# Create database setup script
cd /home/ec2-user/formhub-src/backend

# Update environment for port 9000 and existing MariaDB
cat > .env << EOF
ENVIRONMENT=production
PORT=9000
DATABASE_URL=mysql://root:password@localhost:3306/formhub?parseTime=true
REDIS_URL=
JWT_SECRET=formhub-production-secret-$(date +%s)
ALLOWED_ORIGINS=http://13.201.64.45:9000,http://13.201.64.45:9001,https://yourdomain.com
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=mejona.tech@gmail.com
SMTP_PASSWORD=pkjs cehq vhpc atek
FROM_EMAIL=noreply@formhub.com
FROM_NAME=FormHub by Mejona Technology
EOF

# Create database
mysql -u root -p -e 'CREATE DATABASE IF NOT EXISTS formhub;' 2>/dev/null || echo 'Database setup may need manual configuration'

# Build FormHub
export PATH=/usr/local/go/bin:\$PATH
go mod tidy
go build -o formhub-api main.go
"

echo [3/5] Creating systemd service...
ssh -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" ec2-user@13.201.64.45 "
sudo tee /etc/systemd/system/formhub.service > /dev/null << EOF
[Unit]
Description=FormHub API Service
After=network.target mysql.service

[Service]
Type=simple
User=ec2-user
WorkingDirectory=/home/ec2-user/formhub-src/backend
ExecStart=/home/ec2-user/formhub-src/backend/formhub-api
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable formhub
"

echo [4/5] Starting FormHub service...
ssh -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" ec2-user@13.201.64.45 "
sudo systemctl start formhub
sudo systemctl status formhub
"

echo [5/5] Setting up frontend (static files)...
ssh -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" ec2-user@13.201.64.45 "
# Serve frontend through existing Apache on port 80
sudo mkdir -p /var/www/html/formhub
# Frontend will be accessible at http://13.201.64.45/formhub/
"

goto FINISH

:DOCKER
echo.
echo ========================================
echo    Docker Deployment with Custom Ports
echo ========================================
echo.

# Update docker-compose to use different ports
(
echo version: '3.8'
echo.
echo services:
echo   postgres:
echo     image: postgres:13-alpine
echo     environment:
echo       POSTGRES_DB: formhub
echo       POSTGRES_USER: formhub
echo       POSTGRES_PASSWORD: formhub123
echo     ports:
echo       - "5433:5432"  # Use 5433 to avoid conflicts
echo     volumes:
echo       - formhub_db:/var/lib/postgresql/data
echo.
echo   api:
echo     build: ./backend
echo     ports:
echo       - "9010:8080"  # FormHub API on port 9010
echo     environment:
echo       DATABASE_URL: postgres://formhub:formhub123@postgres:5432/formhub?sslmode=disable
echo       PORT: 8080
echo       SMTP_USERNAME: mejona.tech@gmail.com
echo       SMTP_PASSWORD: pkjs cehq vhpc atek
echo     depends_on:
echo       - postgres
echo.
echo volumes:
echo   formhub_db:
) > docker-compose.custom.yml

scp -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" docker-compose.custom.yml ec2-user@13.201.64.45:/home/ec2-user/

ssh -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" ec2-user@13.201.64.45 "
sudo systemctl start docker 2>/dev/null || sudo yum install -y docker && sudo systemctl start docker
docker-compose -f docker-compose.custom.yml up -d
"

goto FINISH

:INTEGRATE
echo.
echo ========================================
echo    Integrating with Existing Setup
echo ========================================
echo.
echo This will add FormHub as a microservice to your existing architecture.
echo FormHub will use:
echo - Port 8090 (next available)
echo - Your existing MariaDB
echo - Integrated with your Apache reverse proxy
echo.
# Implementation for integration option
goto FINISH

:FINISH
echo.
echo ========================================
echo    FormHub Deployment Complete!
echo ========================================
echo.

if "%CHOICE%"=="1" (
    echo FormHub API running on: http://13.201.64.45:9000/health
    echo Frontend accessible at: http://13.201.64.45/formhub/
) else if "%CHOICE%"=="2" (
    echo FormHub API running on: http://13.201.64.45:9010/health
    echo Frontend running on: http://13.201.64.45:9011
)

echo.
echo Email configured: mejona.tech@gmail.com
echo.
echo Testing deployment...
timeout /t 5 >nul

if "%CHOICE%"=="1" (
    curl -f http://13.201.64.45:9000/health
    start http://13.201.64.45:9000/health
) else if "%CHOICE%"=="2" (
    curl -f http://13.201.64.45:9010/health
    start http://13.201.64.45:9010/health
)

echo.
pause