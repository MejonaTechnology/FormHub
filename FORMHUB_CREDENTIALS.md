# 🔐 FormHub Production Credentials & Access Details

## 🚀 Service Information
- **Service Name**: FormHub - Form Backend Service
- **URL**: http://13.127.59.135:9000
- **Health Check**: http://13.127.59.135:9000/health
- **Status**: ✅ FULLY OPERATIONAL
- **Deployed**: 2025-08-25 03:09:38 UTC

## 👤 Admin User Account
- **Email**: admin@mejona.tech
- **Password**: FormHub2025!
- **User ID**: 8099e3d1-9672-4418-bee8-26e1966529dd
- **Plan**: Free
- **Created**: 2025-08-25 03:11:28 UTC

## 🔑 API Keys
### Test API Key (Active)
- **Name**: Test API Key
- **Key**: `b73fb689-7901-4122-8cb0-3dfcc2498235-4621361e-a4a1-448d-bf81-8a2302d452ea`
- **ID**: 1105be18-37ad-4a9d-b929-47907121a872
- **Permissions**: form_submit
- **Rate Limit**: 1000 requests/minute
- **Status**: Active ✅

### Default API Key (Active)
- **Name**: Default API Key
- **ID**: 4b735f57-632e-4dee-86be-e1f877ba4518
- **Status**: Active (Key hash only stored in DB)

## 📝 Sample Form
- **Form Name**: Contact Form
- **Form ID**: `32c55b70-a3e0-4af7-ab2a-aa53db09bf19`
- **Target Email**: mejona.tech@gmail.com
- **Description**: Main contact form for Mejona Technology
- **Success Message**: "Thank you for your message! We will get back to you soon."

## 🗄️ Database Access
- **Server**: 13.127.59.135
- **Database**: formhub
- **User**: root
- **Password**: mejona123
- **Engine**: MariaDB 10.5.29

### Database Tables Created ✅
- `users` - User accounts and authentication
- `api_keys` - API key management
- `forms` - Form configurations
- `submissions` - Form submission data

## ⚙️ Server Configuration
- **Server**: AWS EC2 (13.127.59.135)
- **OS**: Amazon Linux 2023
- **Service**: systemd (formhub-api.service)
- **Port**: 9000
- **Memory**: 5MB usage
- **CPU**: Minimal usage
- **Redis**: redis6.service (port 6379) ✅

## 📧 Email Configuration (SMTP)
- **Host**: smtp.gmail.com
- **Port**: 587
- **Username**: mejona.tech@gmail.com
- **Password**: pkjs cehq vhpc atek (Gmail App Password)
- **From Name**: FormHub by Mejona Technology
- **Status**: ✅ WORKING (Email sent successfully)

## 🔄 GitHub Repository & CI/CD
- **Repository**: https://github.com/MejonaTechnology/FormHub.git
- **Branch**: master
- **CI/CD**: GitHub Actions ✅
- **Auto Deploy**: On push to master
- **Build Status**: ✅ All builds passing

### GitHub Secrets Configured
- `EC2_SSH_KEY`: SSH private key for deployment
- `EC2_HOST`: 13.127.59.135
- `EC2_USER`: ec2-user
- `DB_PASSWORD`: mejona123
- `JWT_SECRET`: formhub-prod-secret-2025
- `SMTP_USERNAME`: mejona.tech@gmail.com
- `SMTP_PASSWORD`: pkjs cehq vhpc atek

## 🧪 Verified Test Results
### Form Submission Test ✅
- **Submission ID**: 493b6a63-8c8a-436d-8afe-a1e607237cdd
- **Test Email**: john@example.com
- **Email Sent**: ✅ Confirmed
- **Response Time**: 4.2 seconds
- **Status**: SUCCESS (200)

## 🔗 API Endpoints

### Public Endpoints
- `POST /api/v1/submit` - Form submission
- `GET /health` - Health check

### Authentication
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/refresh` - Token refresh

### Protected Endpoints (Require JWT)
- `GET /api/v1/profile` - Get user profile
- `GET /api/v1/forms` - List user forms
- `POST /api/v1/forms` - Create new form
- `GET /api/v1/api-keys` - List API keys
- `POST /api/v1/api-keys` - Create new API key
- `GET /api/v1/submissions` - View submissions

## 🎯 Client Integration Example
```html
<form id="contact-form">
  <input type="hidden" name="access_key" value="b73fb689-7901-4122-8cb0-3dfcc2498235-4621361e-a4a1-448d-bf81-8a2302d452ea">
  <input type="hidden" name="form_id" value="32c55b70-a3e0-4af7-ab2a-aa53db09bf19">
  
  <input type="text" name="name" placeholder="Your Name" required>
  <input type="email" name="email" placeholder="Your Email" required>
  <textarea name="message" placeholder="Your Message" required></textarea>
  
  <button type="submit">Send Message</button>
</form>

<script>
document.getElementById('contact-form').onsubmit = async (e) => {
  e.preventDefault();
  const formData = new FormData(e.target);
  const data = Object.fromEntries(formData);
  
  const response = await fetch('http://13.127.59.135:9000/api/v1/submit', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(data)
  });
  
  const result = await response.json();
  alert(result.message);
};
</script>
```

## 🛡️ Security Features
- JWT-based authentication ✅
- API key validation ✅
- Rate limiting (Redis) ✅
- SQL injection protection ✅
- CORS configuration ✅
- Input validation ✅
- Secure password hashing (bcrypt) ✅

## 📈 Monitoring & Logs
- **Service Status**: `sudo systemctl status formhub-api`
- **Logs**: `journalctl -u formhub-api -f`
- **Health Check**: `curl http://13.127.59.135:9000/health`
- **Database Check**: `mysql -u root -pmejona123 formhub`

## 🚀 Production Readiness
- ✅ Database migrations completed
- ✅ SMTP email sending verified
- ✅ API endpoints fully functional
- ✅ User authentication working
- ✅ Form submissions processing
- ✅ Rate limiting active
- ✅ Auto-restart on failure (systemd)
- ✅ GitHub CI/CD pipeline
- ✅ SSL-ready (can add domain + certificate)

---
**FormHub is 100% operational and ready to serve clients!** 🎉

**Last Updated**: 2025-08-25 03:15 UTC
**Deployment Status**: ✅ PRODUCTION READY