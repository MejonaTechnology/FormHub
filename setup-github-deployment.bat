@echo off
echo ========================================
echo    FormHub GitHub CI/CD Setup
echo ========================================
echo.

echo Setting up FormHub for GitHub deployment...
echo Similar to your employee-management-microservice setup
echo.

echo [1/4] Initializing Git repository...
cd /d "D:\Mejona Workspace\Product\FormHub"
git init
git add .
git commit -m "Initial FormHub commit - Form backend service with CI/CD"

echo.
echo [2/4] Repository created locally
echo Next steps to complete setup:
echo.

echo ========================================
echo    MANUAL STEPS REQUIRED
echo ========================================
echo.

echo 1. CREATE GITHUB REPOSITORY:
echo    - Go to: https://github.com/MejonaTechnology
echo    - Click "New repository" 
echo    - Name: FormHub
echo    - Description: Professional form backend service for client websites
echo    - Make it Public or Private (your choice)
echo    - Create repository
echo.

echo 2. PUSH CODE TO GITHUB:
echo    git remote add origin https://github.com/MejonaTechnology/FormHub.git
echo    git branch -M main  
echo    git push -u origin main
echo.

echo 3. SETUP GITHUB SECRETS:
echo    Go to: Repository Settings → Secrets and variables → Actions
echo    Add these secrets:
echo.
echo    EC2_SSH_KEY: [Copy content from mejonaN.pem file]
echo    EC2_HOST: 13.127.59.135
echo    EC2_USER: ec2-user
echo    DB_PASSWORD: mejona123
echo    JWT_SECRET: formhub-prod-secret-2025
echo    SMTP_USERNAME: mejona.tech@gmail.com
echo    SMTP_PASSWORD: pkjs cehq vhpc atek
echo.

echo [3/4] Preparing SSH key for copy...
echo.
echo SSH Key content to copy to EC2_SSH_KEY secret:
echo ================================================
type "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem"
echo ================================================
echo.

echo [4/4] Ready for deployment!
echo.
echo After setting up GitHub repository and secrets:
echo ✓ Push code to trigger automatic deployment
echo ✓ FormHub will deploy to port 9000 on your EC2
echo ✓ Add port 9000 to AWS Security Group
echo ✓ Test: http://13.127.59.135:9000/health
echo.

echo ========================================
echo    Deployment URLs After Setup
echo ========================================
echo.
echo 🔗 FormHub API: http://13.127.59.135:9000/health
echo 📧 Email: mejona.tech@gmail.com (configured)
echo 🗄️ Database: MySQL formhub database
echo 📊 Service: systemd formhub-api.service
echo.

echo Would you like me to:
echo A) Open GitHub to create repository
echo B) Show the SSH key content again
echo C) Continue with manual setup
echo.

set /p CHOICE="Choose option (A/B/C): "

if "%CHOICE%"=="A" (
    start https://github.com/MejonaTechnology
    echo Opening GitHub for you to create FormHub repository...
)

if "%CHOICE%"=="B" (
    echo.
    echo SSH Key for EC2_SSH_KEY secret:
    echo ================================
    type "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem"
    echo ================================
)

echo.
echo FormHub is ready for professional GitHub deployment!
echo Follow the manual steps above to complete the setup.
echo.
pause