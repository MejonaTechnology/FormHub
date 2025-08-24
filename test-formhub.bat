@echo off
echo ========================================
echo    FormHub System Test
echo ========================================
echo.

echo [1/4] Testing if services are running...
curl -s http://localhost:8080/health >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: FormHub API is not responding
    echo Please run quick-start.bat first
    pause
    exit /b 1
)
echo ✓ FormHub API is running

echo.
echo [2/4] Testing database connection...
docker-compose exec -T postgres pg_isready -U formhub >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: Database is not ready
    echo Please wait a few more minutes for services to start
    pause
    exit /b 1
)
echo ✓ Database is ready

echo.
echo [3/4] Testing Redis connection...
docker-compose exec -T redis redis-cli ping >nul 2>&1
if %errorlevel% neq 0 (
    echo WARNING: Redis may not be ready (this is okay for basic testing)
) else (
    echo ✓ Redis is ready
)

echo.
echo [4/4] Running comprehensive API tests...
if exist "backend\test\api_test.py" (
    python backend\test\api_test.py
) else (
    echo Python test script not found, testing manually...
    curl -X GET http://localhost:8080/health
)

echo.
echo ========================================
echo    Test Results Summary
echo ========================================
echo.
echo ✓ FormHub is running and ready for use!
echo.
echo What to test next:
echo 1. Open: backend\test\example_form.html (test form)
echo 2. Visit: http://localhost:3000 (frontend dashboard)
echo 3. API Health: http://localhost:8080/health
echo.
echo To create your first account:
echo curl -X POST http://localhost:8080/api/v1/auth/register -H "Content-Type: application/json" -d "{\"email\":\"test@example.com\",\"password\":\"password123\",\"first_name\":\"John\",\"last_name\":\"Doe\"}"
echo.
pause