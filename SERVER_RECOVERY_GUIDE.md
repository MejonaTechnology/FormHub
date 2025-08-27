# 🚨 FormHub Server Recovery & Troubleshooting Guide

## Current Situation Analysis

**Status**: ❌ **EC2 SERVER UNRESPONSIVE**  
**IP Address**: 13.127.59.135  
**Last Known Status**: Backend operational, frontend deployment in progress  
**Issue Timeframe**: Started around 2025-08-27 17:05 UTC  

### Confirmed Issues
- ❌ SSH connections timing out (port 22)
- ❌ HTTP requests failing (ports 3000, 9000)
- ❌ Complete network unresponsiveness
- ❌ Both `ping` and `curl` requests failing

---

## 🔍 AWS EC2 Server Recovery Steps

### Step 1: AWS Console Investigation
**Access Required**: AWS Management Console with EC2 access

1. **Check Instance Status**
   ```
   AWS Console → EC2 → Instances → Select 13.127.59.135
   - Instance State: Check if "running", "stopped", or "terminated"
   - Status Checks: System reachability and instance reachability
   - Monitoring: CPU, memory, and network utilization
   ```

2. **Common Instance Issues**
   - **Stopped Instance**: Click "Start" to restart
   - **Failed Status Checks**: May need instance restart
   - **High CPU/Memory**: Instance may have crashed due to resource exhaustion
   - **Network Issues**: Security group or VPC configuration problems

### Step 2: Security Group Verification
**Required Ports**: 22 (SSH), 3000 (Frontend), 9000 (Backend)

1. **Check Security Group Rules**
   ```
   Security Groups → Inbound Rules:
   - SSH (22): 0.0.0.0/0 or your IP
   - HTTP (3000): 0.0.0.0/0
   - API (9000): 0.0.0.0/0
   ```

2. **Common Fixes**
   - Add missing port rules
   - Verify source IP restrictions
   - Check for rule conflicts

### Step 3: Instance Recovery Actions

#### Option A: Instance Restart
```bash
# Via AWS Console
1. Select EC2 instance
2. Instance State → Restart
3. Wait 2-3 minutes for boot
4. Test connectivity: ssh -i "mejonaN.pem" ec2-user@13.127.59.135
```

#### Option B: Stop/Start Cycle
```bash
# Via AWS Console (more thorough than restart)
1. Instance State → Stop → Wait for "stopped"
2. Instance State → Start → Wait for "running"
3. Note: IP address may change!
```

#### Option C: Create New Instance from Snapshot/AMI
If instance is corrupted beyond repair

---

## 🔧 Service Recovery Procedures

### Once Server is Accessible

#### Backend Service Recovery
```bash
# SSH into server
ssh -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" ec2-user@13.127.59.135

# Check service status
sudo systemctl status formhub-api

# If not running, restart
cd /opt/formhub
sudo systemctl start formhub-api

# Manual start if service fails
sudo pkill -f formhub-api
nohup ./formhub-api > formhub.log 2>&1 &

# Verify
curl http://localhost:9000/health
```

#### Frontend Service Recovery
```bash
# Check frontend service
sudo systemctl status formhub-frontend

# If not running, restart
sudo systemctl start formhub-frontend

# Manual restart from deployment files
cd /opt/mejona/formhub-frontend
npm install
npm run build
npm start

# Verify
curl http://localhost:3000/
```

#### Database Service Recovery
```bash
# Check MariaDB status
sudo systemctl status mariadb

# Restart if needed
sudo systemctl start mariadb

# Verify connection
mysql -u root -pmejona123 formhub -e "SELECT COUNT(*) FROM users;"
```

---

## 📋 Complete Deployment Checklist

### If Server Recovery is Successful

1. **Verify Server Access** ✅
   - SSH connectivity working
   - Basic commands responsive

2. **Check Service Status**
   ```bash
   sudo systemctl status formhub-api formhub-frontend mariadb redis6
   ```

3. **Complete Frontend Deployment**
   ```bash
   # Upload fixed files
   cd "D:\Mejona Workspace\Product\FormHub"
   scp -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" -r frontend/* ec2-user@13.127.59.135:/opt/mejona/formhub-frontend/

   # Run deployment fix
   ssh -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" ec2-user@13.127.59.135 "cd /opt/mejona/formhub-frontend && chmod +x deployment-fix.sh && ./deployment-fix.sh"
   ```

4. **End-to-End Verification**
   ```bash
   # Backend test
   curl http://13.127.59.135:9000/health

   # Frontend test
   curl -I http://13.127.59.135:3000/
   curl -I http://13.127.59.135:3000/dashboard
   curl -I http://13.127.59.135:3000/auth/login

   # Web3Forms API test
   curl -X POST http://13.127.59.135:9000/api/v1/submit \
     -H "Content-Type: application/json" \
     -d '{
       "access_key": "ee48ba7c-a5f6-4a6d-a560-4b02bd0a3bdd-c133f5d0-cb9b-4798-8f15-5b47fa0e726a",
       "name": "Test User",
       "email": "test@example.com",
       "message": "Test message"
     }'
   ```

---

## 🚀 Alternative Deployment Strategies

### Option 1: Local Development Environment
If server remains inaccessible, set up local testing:

```bash
# Backend (Windows)
cd "D:\Mejona Workspace\Product\FormHub\backend"
go run main.go

# Frontend (Windows)
cd "D:\Mejona Workspace\Product\FormHub\frontend"
npm install
npm run dev
```

### Option 2: Docker Deployment
```bash
cd "D:\Mejona Workspace\Product\FormHub"
docker-compose up -d
```

### Option 3: GitHub Actions Deployment
If GitHub Actions is configured:
```bash
# Push to trigger deployment
git add .
git commit -m "Trigger deployment"
git push origin main
```

---

## 📊 Troubleshooting Checklist

### Network Connectivity Issues
- [ ] Instance is in "running" state
- [ ] Security groups allow required ports
- [ ] VPC/Subnet configuration correct
- [ ] Elastic IP associated (if used)
- [ ] Route table configuration valid

### Service Issues
- [ ] formhub-api service running
- [ ] formhub-frontend service running
- [ ] MariaDB service running
- [ ] Redis service running
- [ ] Correct environment variables set

### Application Issues
- [ ] Backend health endpoint responds
- [ ] Frontend pages load (not 404)
- [ ] API key submissions work
- [ ] Database connections successful
- [ ] Email sending functional

---

## 🎯 Success Criteria

### ✅ Complete Success
- [ ] SSH access restored
- [ ] Backend API responding on port 9000
- [ ] Frontend UI accessible on port 3000
- [ ] No 404 errors on dashboard, login, etc.
- [ ] Web3Forms submissions working end-to-end
- [ ] Email notifications sending
- [ ] All services stable and monitored

### ⚠️ Partial Success
- [ ] Server accessible but services need restart
- [ ] Backend working but frontend 404 errors persist
- [ ] Services running but performance issues

### ❌ Recovery Required
- [ ] Instance terminated/corrupted
- [ ] Data loss requiring backup restoration
- [ ] Security group/network misconfiguration

---

## 🔄 Monitoring & Prevention

### Ongoing Monitoring
```bash
# Set up monitoring scripts
watch -n 30 'curl -s http://13.127.59.135:9000/health && curl -s -I http://13.127.59.135:3000/'

# Log monitoring
sudo journalctl -u formhub-api -f &
sudo journalctl -u formhub-frontend -f &
```

### Prevention Measures
1. **Resource Monitoring**: Monitor CPU/memory usage
2. **Automatic Restarts**: Ensure systemd restart policies
3. **Health Checks**: Implement application health monitoring
4. **Backup Strategy**: Regular database and configuration backups
5. **Staging Environment**: Test deployments before production

---

## 📞 Emergency Contact Information

### AWS Support
- Use AWS Support cases for instance-level issues
- Check AWS Service Health Dashboard for regional issues

### Service Recovery Scripts
- `D:\Mejona Workspace\Product\FormHub\restart-services.py`
- `D:\Mejona Workspace\Product\FormHub\quick-deploy-frontend.bat`
- `D:\Mejona Workspace\Product\FormHub\test-services.py`

### Quick Recovery Commands
```bash
# Test connectivity
ssh -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" -o ConnectTimeout=10 ec2-user@13.127.59.135 "echo 'Server responsive'"

# Restart all services
ssh -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" ec2-user@13.127.59.135 "sudo systemctl restart formhub-api formhub-frontend"

# Check service status
ssh -i "D:\Mejona Workspace\Product\Mejona complete website\mejonaN.pem" ec2-user@13.127.59.135 "sudo systemctl status formhub-api formhub-frontend --no-pager"
```

---

**Priority**: 🚨 **CRITICAL** - Server recovery required for FormHub completion  
**Next Action**: Check AWS EC2 console for instance status  
**Estimated Recovery Time**: 15-30 minutes once server is accessible  
**Status**: Ready for immediate recovery procedures