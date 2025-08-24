@echo off
echo ========================================
echo    FormHub Quick Start Setup
echo ========================================
echo.

echo [1/5] Checking if Docker is installed...
docker --version >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: Docker is not installed or not in PATH
    echo Please install Docker Desktop from: https://www.docker.com/products/docker-desktop
    pause
    exit /b 1
)
echo ✓ Docker is installed

echo.
echo [2/5] Checking if Docker Compose is available...
docker-compose --version >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: Docker Compose is not available
    echo Please make sure Docker Desktop is running
    pause
    exit /b 1
)
echo ✓ Docker Compose is available

echo.
echo [3/5] Creating environment file...
if not exist "backend\.env" (
    copy "backend\.env.example" "backend\.env"
    echo ✓ Environment file created at backend\.env
    echo WARNING: You need to edit backend\.env with your SMTP settings
) else (
    echo ✓ Environment file already exists
)

echo.
echo [4/5] Starting FormHub services...
echo This may take a few minutes for the first time...
docker-compose up -d

echo.
echo [5/5] Waiting for services to be ready...
timeout /t 10 >nul

echo.
echo ========================================
echo    FormHub is starting up!
echo ========================================
echo.
echo Backend API: http://localhost:8080/health
echo Frontend Dashboard: http://localhost:3000
echo Database: PostgreSQL on localhost:5432
echo Cache: Redis on localhost:6379
echo.
echo Opening health check in browser...
start http://localhost:8080/health

echo.
echo Next steps:
echo 1. Edit backend\.env with your SMTP credentials
echo 2. Test the API: python backend\test\api_test.py
echo 3. Open test form: backend\test\example_form.html
echo.
pause