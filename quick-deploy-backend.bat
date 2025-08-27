@echo off
echo ========================================
echo   Quick Backend Deployment for FormHub
echo ========================================
echo.
echo Deploying only the backend changes with Web3Forms functionality...
echo.

cd backend
echo Building FormHub backend...
go build -o formhub-api main.go

echo.
echo Uploading to server...
scp formhub-api ubuntu@13.127.59.135:/tmp/
scp .env.example ubuntu@13.127.59.135:/tmp/.env

echo.
echo Deploying on server...
ssh ubuntu@13.127.59.135 "
    sudo systemctl stop formhub-api 2>/dev/null || true
    sudo cp /tmp/formhub-api /opt/mejona/formhub/
    sudo cp /tmp/.env /opt/mejona/formhub/
    sudo chown ubuntu:ubuntu /opt/mejona/formhub/*
    sudo chmod +x /opt/mejona/formhub/formhub-api
    sudo systemctl start formhub-api
    sleep 3
    sudo systemctl status formhub-api --no-pager
"

echo.
echo Testing deployment...
curl -X GET "http://13.127.59.135:9000/health"

echo.
echo ✅ Quick backend deployment completed!
echo 🧪 Test your API key now
pause