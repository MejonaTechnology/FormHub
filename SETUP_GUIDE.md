# FormHub Setup Guide

A comprehensive guide to set up and run FormHub - a professional form backend service similar to Web3Forms.

## 🚀 Quick Start

### Prerequisites
- **Go 1.21+** for backend API
- **Node.js 18+** for frontend dashboard
- **PostgreSQL 12+** for database
- **Redis 6+** for caching and rate limiting
- **Docker & Docker Compose** (optional, for containerized setup)

## 📋 Setup Options

### Option 1: Docker Setup (Recommended)

1. **Clone the repository**
```bash
git clone <repository-url>
cd FormHub
```

2. **Configure environment variables**
```bash
cp backend/.env.example backend/.env
# Edit backend/.env with your SMTP settings
```

3. **Start all services**
```bash
docker-compose up -d
```

4. **Verify services are running**
```bash
# Check API health
curl http://localhost:8080/health

# Check database connection
docker-compose logs postgres

# Check Redis connection
docker-compose logs redis
```

### Option 2: Manual Setup

#### Backend Setup

1. **Navigate to backend directory**
```bash
cd backend
```

2. **Install Go dependencies**
```bash
go mod tidy
```

3. **Set up PostgreSQL database**
```sql
CREATE DATABASE formhub;
CREATE USER formhub WITH ENCRYPTED PASSWORD 'formhub123';
GRANT ALL PRIVILEGES ON DATABASE formhub TO formhub;
```

4. **Run database migrations**
```bash
# Connect to PostgreSQL and run the migration
psql -U formhub -d formhub -f migrations/001_initial_schema.sql
```

5. **Configure environment variables**
```bash
cp .env.example .env
# Edit .env file with your settings
```

6. **Start the backend server**
```bash
go run main.go
```

#### Frontend Setup

1. **Navigate to frontend directory**
```bash
cd frontend
```

2. **Install Node.js dependencies**
```bash
npm install
```

3. **Configure environment variables**
```bash
echo "NEXT_PUBLIC_API_URL=http://localhost:8080" > .env.local
```

4. **Start the frontend development server**
```bash
npm run dev
```

## 🔧 Configuration

### Backend Environment Variables

```env
# Required Settings
ENVIRONMENT=development
PORT=8080
DATABASE_URL=postgres://formhub:formhub123@localhost:5432/formhub?sslmode=disable
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# Email Configuration (Required)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
FROM_EMAIL=noreply@formhub.com
FROM_NAME=FormHub

# Optional Settings
REDIS_URL=redis://localhost:6379
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
```

### Gmail SMTP Setup

1. Enable 2-Factor Authentication on your Gmail account
2. Generate an App Password:
   - Go to Google Account settings
   - Security → 2-Step Verification → App passwords
   - Generate password for "Mail"
   - Use this password in `SMTP_PASSWORD`

### Frontend Environment Variables

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
```

## 🧪 Testing the Setup

### 1. Backend API Tests

Run the comprehensive test suite:

```bash
cd backend/test
python3 api_test.py
```

### 2. Frontend Form Test

1. Open the test form in your browser:
```bash
# From the project root
open backend/test/example_form.html
# or
firefox backend/test/example_form.html
```

2. Create a user account and API key:
   - Go to http://localhost:3000 (frontend dashboard)
   - Register a new account
   - Create an API key
   - Copy the API key to the test form

3. Submit a test form to verify email delivery

### 3. Manual API Testing

```bash
# Test health endpoint
curl http://localhost:8080/health

# Register a user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "testpassword123",
    "first_name": "John",
    "last_name": "Doe",
    "company": "Test Company"
  }'

# Submit a form (replace YOUR_API_KEY with actual key)
curl -X POST http://localhost:8080/api/v1/submit \
  -H "Content-Type: application/json" \
  -d '{
    "access_key": "YOUR_API_KEY",
    "email": "user@example.com",
    "subject": "Test Submission",
    "message": "This is a test message"
  }'
```

## 📊 Service URLs

After successful setup, these services will be available:

- **Backend API**: http://localhost:8080
- **Frontend Dashboard**: http://localhost:3000
- **API Documentation**: http://localhost:8080/health (basic health check)
- **PostgreSQL**: localhost:5432
- **Redis**: localhost:6379

## 🔐 Security Considerations

### Production Deployment

1. **Change default passwords and secrets**
2. **Use environment-specific configurations**
3. **Enable HTTPS with proper SSL certificates**
4. **Configure firewall rules**
5. **Set up proper backup strategies**
6. **Enable logging and monitoring**

### Database Security

```sql
-- Create read-only user for analytics
CREATE USER formhub_readonly WITH ENCRYPTED PASSWORD 'readonly_password';
GRANT CONNECT ON DATABASE formhub TO formhub_readonly;
GRANT USAGE ON SCHEMA public TO formhub_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO formhub_readonly;
```

## 📈 Monitoring and Maintenance

### Health Checks

```bash
# Backend health
curl http://localhost:8080/health

# Database connection
docker-compose exec postgres pg_isready -U formhub

# Redis connection  
docker-compose exec redis redis-cli ping
```

### Log Files

```bash
# Docker logs
docker-compose logs -f api
docker-compose logs -f postgres
docker-compose logs -f redis

# Manual setup logs
tail -f /var/log/formhub/api.log
```

### Database Maintenance

```sql
-- View submission statistics
SELECT 
  DATE_TRUNC('day', created_at) as date,
  COUNT(*) as submissions,
  COUNT(CASE WHEN is_spam THEN 1 END) as spam_count
FROM submissions 
WHERE created_at > NOW() - INTERVAL '30 days'
GROUP BY DATE_TRUNC('day', created_at)
ORDER BY date;

-- Clean up old submissions (optional)
DELETE FROM submissions 
WHERE created_at < NOW() - INTERVAL '6 months' 
AND is_spam = true;
```

## 🆘 Troubleshooting

### Common Issues

**1. Database connection failed**
```bash
# Check PostgreSQL is running
sudo systemctl status postgresql
# or
docker-compose ps postgres

# Test connection
psql -U formhub -h localhost -d formhub -c "SELECT version();"
```

**2. SMTP authentication failed**
- Verify Gmail App Password is correct
- Check 2FA is enabled on Gmail account
- Try different SMTP providers (SendGrid, Mailgun)

**3. CORS errors in frontend**
- Verify `ALLOWED_ORIGINS` includes frontend URL
- Check frontend API URL configuration

**4. Redis connection failed**
```bash
# Check Redis is running
redis-cli ping
# or
docker-compose ps redis
```

### Getting Help

1. Check the logs for detailed error messages
2. Verify all environment variables are set correctly
3. Ensure all services are running and accessible
4. Test with minimal configuration first
5. Check firewall and network configurations

## 🎯 Next Steps

After successful setup:

1. **Customize email templates** in `pkg/email/smtp.go`
2. **Add your company branding** to frontend
3. **Configure custom domains** for production
4. **Set up monitoring and alerts**
5. **Create backup strategies**
6. **Implement additional spam protection**
7. **Add webhook integrations**

---

**Built with ❤️ by Mejona Technology LLP**

For more documentation and advanced configurations, see the `docs/` directory.