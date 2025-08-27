@echo off
echo.
echo ===============================================
echo   FormHub Frontend Deployment Fix - Windows
echo ===============================================
echo.

set SERVER=13.127.59.135
set USER=ubuntu
set REMOTE_PATH=/opt/mejona/formhub-frontend

echo 🚀 Starting FormHub Frontend deployment fix...
echo.

echo 📤 Uploading frontend files...
scp -r frontend/* %USER%@%SERVER%:%REMOTE_PATH%/

if %ERRORLEVEL% NEQ 0 (
    echo ❌ Upload failed!
    pause
    exit /b 1
)

echo.
echo 🔧 Executing deployment fix on server...
ssh %USER%@%SERVER% "cd %REMOTE_PATH% && chmod +x deployment-fix.sh && ./deployment-fix.sh"

if %ERRORLEVEL% NEQ 0 (
    echo ❌ Deployment fix failed!
    pause
    exit /b 1
)

echo.
echo ✅ Deployment fix completed successfully!
echo 🌐 Frontend should now be accessible at: http://%SERVER%:3000
echo.
echo 🔍 Testing accessibility...
curl -s -o NUL -w "Status: %%{http_code}" http://%SERVER%:3000/
echo.
echo.
echo 📋 Next steps:
echo   1. Open browser to http://%SERVER%:3000
echo   2. Test login at http://%SERVER%:3000/auth/login  
echo   3. Test dashboard at http://%SERVER%:3000/dashboard
echo   4. Verify API connectivity with backend
echo.
pause