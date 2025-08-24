@echo off
echo ========================================
echo    FormHub Email Configuration
echo ========================================
echo.

set /p EMAIL="Enter your Gmail address: "
echo.

echo Creating environment configuration with your email settings...
echo.

(
echo # FormHub Development Environment with Email Configuration
echo ENVIRONMENT=development
echo PORT=8080
echo DATABASE_URL=postgres://formhub:formhub123@postgres:5432/formhub?sslmode=disable
echo REDIS_URL=redis://redis:6379
echo JWT_SECRET=formhub-dev-secret-change-in-production
echo ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173,http://localhost:8080
echo.
echo # Gmail SMTP Configuration
echo SMTP_HOST=smtp.gmail.com
echo SMTP_PORT=587
echo SMTP_USERNAME=%EMAIL%
echo SMTP_PASSWORD=pkjs cehq vhpc atek
echo FROM_EMAIL=noreply@formhub.com
echo FROM_NAME=FormHub
) > backend\.env

echo ✓ Email configuration saved to backend\.env
echo.
echo Your settings:
echo - SMTP Host: Gmail
echo - Username: %EMAIL%
echo - Password: pkjs cehq vhpc atek
echo.
echo FormHub is now configured with your email settings!
echo You can now run: quick-start.bat
echo.
pause