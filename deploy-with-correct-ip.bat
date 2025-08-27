@echo off
echo ========================================
echo   FormHub Manual Deployment - Correct IP
echo ========================================
echo Deploying to NEW IP: 13.127.59.135
echo.

cd backend
echo Building FormHub backend...
go mod tidy
go build -ldflags="-w -s" -o formhub-api main.go

if not exist formhub-api.exe (
    echo ❌ Build failed!
    pause
    exit /b 1
)

echo ✅ Build successful!
echo.

echo Creating environment file...
(
echo ENVIRONMENT=production
echo PORT=9000
echo DATABASE_URL=root:mejona123@tcp(localhost:3306)/formhub?parseTime=true
echo REDIS_URL=redis://localhost:6379
echo JWT_SECRET=formhub-prod-secret-2025
echo ALLOWED_ORIGINS=http://13.127.59.135:9000,https://formhub.mejona.in
echo SMTP_HOST=smtp.gmail.com
echo SMTP_PORT=587
echo SMTP_USERNAME=mejona.tech@gmail.com
echo SMTP_PASSWORD=pkjs cehq vhpc atek
echo FROM_EMAIL=noreply@formhub.com
echo FROM_NAME=FormHub by Mejona Technology
) > .env

echo Uploading to correct IP: 13.127.59.135...
scp -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" formhub-api ubuntu@13.127.59.135:/tmp/
scp -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" .env ubuntu@13.127.59.135:/tmp/

echo.
echo Deploying on server...
ssh -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" ubuntu@13.127.59.135 ^
"sudo systemctl stop formhub-api 2>/dev/null || true; ^
 sudo mkdir -p /opt/mejona/formhub; ^
 sudo cp /tmp/formhub-api /opt/mejona/formhub/; ^
 sudo cp /tmp/.env /opt/mejona/formhub/; ^
 sudo chown ubuntu:ubuntu /opt/mejona/formhub/*; ^
 sudo chmod +x /opt/mejona/formhub/formhub-api; ^
 cd /opt/mejona/formhub; ^
 sudo systemctl start formhub-api; ^
 sleep 3; ^
 sudo systemctl status formhub-api --no-pager"

echo.
echo Testing deployment...
curl -X GET "http://13.127.59.135:9000/health"

echo.
echo 🧪 Testing your API key...
curl -X POST "http://13.127.59.135:9000/api/v1/submit" -H "Content-Type: application/json" -d "{\"access_key\":\"ee48ba7c-a5f6-4a6d-a560-4b02bd0a3bdd-c133f5d0-cb9b-4798-8f15-5b47fa0e726a\",\"email\":\"test@example.com\",\"subject\":\"Test\",\"message\":\"Testing Web3Forms functionality\"}"

echo.
echo ✅ Manual deployment with correct IP completed!
pause