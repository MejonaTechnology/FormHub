# 🚀 FormHub EC2 Deployment Guide

## Your Server Details
- **Server IP**: 13.127.59.135
- **User**: ec2-user  
- **Key File**: `D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem`
- **Email Configured**: mejona.tech@gmail.com

## 🎯 Quick Deploy (Automated)

Just run this command:
```bash
cd "D:\Mejona Workspace\Product\FormHub"
deploy-to-ec2.bat
```

## 📋 Manual Deployment Steps

### Step 1: Test Connection
```bash
ssh -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" ec2-user@13.127.59.135
```

### Step 2: Prepare Server
```bash
# On EC2 server
sudo yum update -y
sudo yum install -y docker git
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -a -G docker ec2-user

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/download/1.29.2/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Logout and login again for docker group
exit
```

### Step 3: Upload FormHub Files
```bash
# From your Windows machine
scp -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" -r FormHub ec2-user@13.127.59.135:/home/ec2-user/
```

### Step 4: Deploy FormHub
```bash
# SSH back to server
ssh -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" ec2-user@13.127.59.135

# Move to deployment location
sudo mkdir -p /opt/formhub
sudo chown -R ec2-user:ec2-user /opt/formhub
mv FormHub/* /opt/formhub/
cd /opt/formhub

# Start FormHub
docker-compose up -d
```

## 🔧 AWS Security Group Configuration

**Add these inbound rules:**

| Type | Protocol | Port Range | Source | Description |
|------|----------|------------|---------|-------------|
| SSH | TCP | 22 | Your IP | SSH access |
| HTTP | TCP | 80 | 0.0.0.0/0 | Web traffic |
| HTTPS | TCP | 443 | 0.0.0.0/0 | Secure web traffic |
| Custom TCP | TCP | 8080 | 0.0.0.0/0 | FormHub API |
| Custom TCP | TCP | 3000 | 0.0.0.0/0 | FormHub Frontend |

## 🧪 Testing Your Deployment

### 1. Health Check
```bash
curl http://13.127.59.135:8080/health
```
**Expected**: `{"status":"healthy","version":"1.0.0"}`

### 2. Register Test User
```bash
curl -X POST http://13.127.59.135:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@formhub.com","password":"password123","first_name":"Test","last_name":"User"}'
```

### 3. Access Frontend
Open browser: `http://13.127.59.135:3000`

### 4. Test Form Submission
```bash
curl -X POST http://13.127.59.135:8080/api/v1/submit \
  -H "Content-Type: application/json" \
  -d '{"access_key":"YOUR_API_KEY","email":"test@example.com","message":"Test from FormHub!"}'
```

## 🌐 Domain Setup (Optional)

### 1. Configure DNS
Point your domain to: `13.127.59.135`
- A record: `formhub.yourdomain.com → 13.127.59.135`
- A record: `api.yourdomain.com → 13.127.59.135`

### 2. Update Environment
```bash
# On EC2 server
cd /opt/formhub
nano backend/.env

# Update these lines:
ALLOWED_ORIGINS=https://formhub.yourdomain.com,https://api.yourdomain.com
FROM_EMAIL=noreply@yourdomain.com
```

### 3. Setup SSL with Let's Encrypt
```bash
# Install certbot
sudo yum install -y certbot

# Get certificate
sudo certbot certonly --standalone -d formhub.yourdomain.com -d api.yourdomain.com

# Update nginx configuration
sudo cp deploy/nginx/nginx.conf /etc/nginx/nginx.conf
sudo systemctl restart nginx
```

## 🔍 Troubleshooting

### Check Services Status
```bash
docker-compose ps
docker-compose logs api
docker-compose logs postgres
```

### Restart Services
```bash
docker-compose restart
```

### View Logs
```bash
docker-compose logs -f api
```

### Database Issues
```bash
# Check database connection
docker-compose exec postgres pg_isready -U formhub

# View database logs
docker-compose logs postgres
```

## 📊 Monitoring Commands

```bash
# System resources
htop
df -h

# Docker stats
docker stats

# Service logs
docker-compose logs -f --tail=100 api
```

## 🎉 Success Indicators

✅ **API Health**: http://13.127.59.135:8080/health returns 200  
✅ **Frontend**: http://13.127.59.135:3000 loads dashboard  
✅ **Email**: Test form submission sends email to mejona.tech@gmail.com  
✅ **Database**: User registration works  
✅ **File Upload**: Form submissions save to database  

## 🚀 Next Steps After Deployment

1. **Test with real client forms**
2. **Set up domain and SSL**
3. **Configure monitoring and backups**
4. **Create client accounts**
5. **Start onboarding existing clients**

---

**FormHub is production-ready on your EC2 server!** 🎯