@echo off
echo ========================================
echo    FormHub Complete Setup & Test
echo ========================================
echo.

echo [STEP 1] Configuring environment with your Gmail settings...
(
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
echo SMTP_USERNAME=GMAIL_ADDRESS_PLACEHOLDER
echo SMTP_PASSWORD=pkjs cehq vhpc atek
echo FROM_EMAIL=noreply@formhub.com
echo FROM_NAME=FormHub
) > backend\.env

echo ✓ Environment file created
echo.

set /p GMAIL_EMAIL="Enter your Gmail address (the one with App Password): "
echo.

echo Updating configuration with your Gmail address...
powershell -Command "(gc backend\.env) -replace 'GMAIL_ADDRESS_PLACEHOLDER', '%GMAIL_EMAIL%' | Out-File -encoding UTF8 backend\.env"
echo ✓ Gmail address configured: %GMAIL_EMAIL%
echo.

echo [STEP 2] Starting FormHub services with Docker...
echo This may take a few minutes for the first time...
echo.
docker-compose up -d

echo.
echo [STEP 3] Waiting for services to initialize...
echo Please wait 30 seconds for database setup...
timeout /t 30 >nul

echo.
echo [STEP 4] Testing FormHub system...
echo.

echo Testing API health...
curl -s http://localhost:8080/health
echo.
echo.

echo [STEP 5] Creating test user account...
echo.
curl -X POST http://localhost:8080/api/v1/auth/register ^
  -H "Content-Type: application/json" ^
  -d "{\"email\":\"test@formhub.local\",\"password\":\"password123\",\"first_name\":\"Test\",\"last_name\":\"User\",\"company\":\"Test Company\"}"

echo.
echo.
echo ========================================
echo    FormHub Setup Complete!
echo ========================================
echo.
echo ✓ Backend API: http://localhost:8080/health
echo ✓ Frontend Dashboard: http://localhost:3000
echo ✓ Database: Running on port 5432
echo ✓ Redis Cache: Running on port 6379
echo ✓ Email: Configured with %GMAIL_EMAIL%
echo.
echo NEXT STEPS:
echo 1. Open test form: backend\test\example_form.html
echo 2. Visit dashboard: http://localhost:3000
echo 3. Check API health: http://localhost:8080/health
echo.
echo Opening test form in browser...
start backend\test\example_form.html
echo.
echo Opening API health check...
start http://localhost:8080/health
echo.
pause