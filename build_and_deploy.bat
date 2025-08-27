@echo off
echo Building FormHub Backend with Web3Forms Bug Fix...
cd /d "D:\Mejona Workspace\Product\FormHub\backend"

echo Step 1: Building Go binary...
go build -o formhub-api-fixed.exe main.go

if %ERRORLEVEL% neq 0 (
    echo ERROR: Go build failed!
    pause
    exit /b 1
)

echo Step 2: Build successful!
echo.
echo The fixed binary is ready: formhub-api-fixed.exe
echo.
echo To deploy:
echo 1. Stop the current backend service on the EC2 server
echo 2. Upload formhub-api-fixed.exe to the server
echo 3. Restart the service with the new binary
echo.
echo For testing, you can run:
echo   formhub-api-fixed.exe
echo.

pause