@echo off
echo ========================================
echo    Testing FormHub on EC2
echo ========================================
echo.

set SERVER_IP=13.201.64.45
set API_URL=http://%SERVER_IP%:8080
set FRONTEND_URL=http://%SERVER_IP%:3000

echo Testing FormHub deployment on: %SERVER_IP%
echo.

echo [1/5] Testing API health check...
curl -f %API_URL%/health
if %errorlevel% equ 0 (
    echo ✅ API health check passed
) else (
    echo ❌ API health check failed
    echo Make sure security groups allow port 8080
)
echo.

echo [2/5] Testing user registration...
curl -X POST %API_URL%/api/v1/auth/register ^
  -H "Content-Type: application/json" ^
  -d "{\"email\":\"test@formhub.com\",\"password\":\"password123\",\"first_name\":\"Test\",\"last_name\":\"User\",\"company\":\"Test Co\"}"

if %errorlevel% equ 0 (
    echo ✅ User registration working
) else (
    echo ❌ User registration failed
)
echo.

echo [3/5] Testing login...
curl -X POST %API_URL%/api/v1/auth/login ^
  -H "Content-Type: application/json" ^
  -d "{\"email\":\"test@formhub.com\",\"password\":\"password123\"}"

if %errorlevel% equ 0 (
    echo ✅ Login working
) else (
    echo ❌ Login failed
)
echo.

echo [4/5] Testing frontend accessibility...
curl -I %FRONTEND_URL% | find "200" >nul
if %errorlevel% equ 0 (
    echo ✅ Frontend is accessible
) else (
    echo ❌ Frontend not accessible
    echo Make sure security groups allow port 3000
)
echo.

echo [5/5] Checking FormHub services on server...
ssh -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" ec2-user@%SERVER_IP% "cd /opt/formhub && /usr/local/bin/docker-compose ps"
echo.

echo ========================================
echo    Test Summary
echo ========================================
echo.
echo If all tests pass, FormHub is ready!
echo.
echo 🔗 Test these URLs in your browser:
echo   API Health: %API_URL%/health
echo   Frontend: %FRONTEND_URL%
echo.
echo 📧 Email configured with: mejona.tech@gmail.com
echo.
echo 🎯 Ready to create client accounts and start business!
echo.

echo Opening test URLs...
start %API_URL%/health
timeout /t 3 >nul
start %FRONTEND_URL%

echo.
pause