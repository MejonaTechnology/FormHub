# 🚀 FormHub GitHub CI/CD Deployment

## Overview
FormHub uses GitHub Actions for automated deployment to AWS EC2, similar to your employee-management-microservice setup.

## 📋 Setup Instructions

### 1. Create GitHub Repository
```bash
# Navigate to FormHub directory
cd "D:\Mejona Workspace\Product\FormHub"

# Initialize git repository
git init
git add .
git commit -m "Initial FormHub commit"

# Create repository on GitHub (FormHub)
# Then push
git remote add origin https://github.com/MejonaTechnology/FormHub.git
git branch -M main
git push -u origin main
```

### 2. Configure GitHub Secrets

Add these secrets in GitHub repository settings:

| Secret Name | Value | Description |
|------------|-------|-------------|
| `EC2_SSH_KEY` | Contents of `mejonaN.pem` file | SSH private key for EC2 |
| `EC2_HOST` | `13.201.64.45` | Your EC2 server IP |
| `EC2_USER` | `ec2-user` | SSH username |
| `DB_PASSWORD` | `mejona123` | MySQL root password |
| `JWT_SECRET` | `formhub-prod-secret-2025` | JWT signing secret |
| `SMTP_USERNAME` | `mejona.tech@gmail.com` | Gmail address |
| `SMTP_PASSWORD` | `pkjs cehq vhpc atek` | Gmail app password |

### 3. GitHub Secrets Setup Commands

```bash
# Copy SSH key content
cat "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem"
# Copy this content to EC2_SSH_KEY secret

# Set other secrets in GitHub repository:
# Settings → Secrets and variables → Actions → New repository secret
```

## 🔧 Deployment Process

### Automatic Deployment
1. **Push to main branch** triggers automatic deployment
2. **GitHub Actions** builds Go binary
3. **Deploys to EC2** on port 9000
4. **Sets up systemd service** for auto-restart
5. **Verifies deployment** with health check

### Manual Deployment
```bash
# Trigger manual deployment
# Go to GitHub → Actions → FormHub Service CI/CD → Run workflow
```

## 📊 Service Configuration

### FormHub will run as:
- **Port**: 9000 (avoids your busy 8080-8087 range)
- **Service**: `formhub-api.service` (systemd)
- **Deploy Path**: `/opt/mejona/formhub`
- **Database**: MySQL `formhub` database
- **Logs**: `journalctl -u formhub-api -f`

### Service Management Commands:
```bash
# On EC2 server
sudo systemctl status formhub-api
sudo systemctl restart formhub-api
sudo systemctl logs -u formhub-api -f

# Health check
curl http://13.201.64.45:9000/health
```

## 🎯 After Deployment

### 1. Update AWS Security Group
Add inbound rule:
- **Type**: Custom TCP
- **Port**: 9000
- **Source**: 0.0.0.0/0
- **Description**: FormHub API

### 2. Test FormHub
```bash
# Health check
curl http://13.201.64.45:9000/health

# Create user
curl -X POST http://13.201.64.45:9000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@formhub.com","password":"password123","first_name":"Test","last_name":"User"}'
```

### 3. Domain Setup (Optional)
- Point `formhub.mejona.in` to `13.201.64.45`
- Update `ALLOWED_ORIGINS` in GitHub secrets
- Set up SSL with Let's Encrypt

## 📁 Repository Structure

```
FormHub/
├── .github/
│   └── workflows/
│       └── deploy.yml          # CI/CD pipeline
├── backend/                    # Go API source code
├── frontend/                   # Next.js dashboard
├── deploy/                     # Deployment configs
├── scripts/                    # Helper scripts
└── README.md                   # Project documentation
```

## 🔄 Development Workflow

1. **Local Development**: Make changes locally
2. **Commit & Push**: `git commit -m "feature" && git push`
3. **Auto Deploy**: GitHub Actions deploys automatically
4. **Verify**: Check health endpoint and test features
5. **Monitor**: Use `journalctl -u formhub-api -f` for logs

## 📈 Monitoring

### Health Checks
- **Endpoint**: `http://13.201.64.45:9000/health`
- **Expected**: `{"status":"healthy","version":"1.0.0"}`

### Service Status
```bash
# Service status
sudo systemctl status formhub-api

# Service logs
journalctl -u formhub-api -f

# Resource usage
htop
```

## 🚨 Troubleshooting

### Deployment Fails
1. Check GitHub Actions logs
2. Verify all secrets are set correctly
3. Ensure EC2 server is accessible
4. Check disk space: `df -h`

### Service Won't Start
```bash
# Check logs
journalctl -u formhub-api -f

# Check binary permissions
ls -la /opt/mejona/formhub/formhub-api

# Test manual start
cd /opt/mejona/formhub
./formhub-api
```

### Database Issues
```bash
# Check MySQL connection
mysql -u root -pmejona123 -e "SHOW DATABASES;"

# Check FormHub database
mysql -u root -pmejona123 formhub -e "SHOW TABLES;"
```

---

**Ready to deploy FormHub with professional CI/CD!** 🚀

Just create the GitHub repository and push the code - automatic deployment will handle the rest!