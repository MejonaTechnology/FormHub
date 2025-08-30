# FormHub Enhanced Webhook System - Deployment Guide

## Overview

This guide covers the deployment of FormHub's Enhanced Webhook System with comprehensive webhook support and third-party integrations.

## Architecture Summary

The Enhanced Webhook System adds:
- ✅ **Multiple webhook endpoints per form** with conditional triggering
- ✅ **Advanced retry logic** with exponential backoff and circuit breaker
- ✅ **Third-party integrations** (Google Sheets, Airtable, Notion, Slack, Discord, Telegram, Zapier)
- ✅ **Real-time analytics and monitoring** with health checks
- ✅ **Enterprise features** including load balancing, failover, and security
- ✅ **Integration marketplace** with pre-built templates
- ✅ **Comprehensive testing and validation tools**

## Pre-Deployment Checklist

### 1. System Requirements

```bash
# Minimum requirements
- CPU: 4 cores
- RAM: 8GB
- Storage: 100GB SSD
- Network: 1Gbps

# Recommended for production
- CPU: 8 cores
- RAM: 16GB
- Storage: 500GB SSD
- Network: 10Gbps
```

### 2. Dependencies

```bash
# Go 1.21+
go version

# MySQL 8.0+ or PostgreSQL 13+
mysql --version

# Redis 6.0+
redis-server --version

# Docker (optional but recommended)
docker --version
docker-compose --version
```

### 3. Environment Variables

```bash
# Copy environment template
cp .env.example .env

# Required environment variables
export DATABASE_URL="mysql://user:password@localhost:3306/formhub"
export REDIS_URL="redis://localhost:6379"
export JWT_SECRET="your-jwt-secret-key"
export SMTP_HOST="smtp.gmail.com"
export SMTP_PORT="587"
export SMTP_USERNAME="your-email@gmail.com"
export SMTP_PASSWORD="your-app-password"

# Optional third-party service keys
export GOOGLE_SHEETS_CREDENTIALS_JSON=""
export SLACK_CLIENT_ID=""
export SLACK_CLIENT_SECRET=""
export AIRTABLE_API_KEY=""
export NOTION_API_KEY=""
export GEOIP_API_KEY=""
```

## Deployment Steps

### Step 1: Database Setup

1. **Run Database Migrations**

```bash
# Navigate to backend directory
cd backend

# Run all migrations including the new enhanced webhook migration
mysql -u root -p formhub < migrations/001_initial_schema.sql
mysql -u root -p formhub < migrations/002_advanced_form_fields.sql
mysql -u root -p formhub < migrations/003_spam_protection_tables.sql
mysql -u root -p formhub < migrations/004_email_template_system.sql
mysql -u root -p formhub < migrations/005_form_analytics_system.sql
mysql -u root -p formhub < migrations/006_enhanced_webhook_system.sql
```

2. **Verify Database Schema**

```bash
# Check if new tables were created
mysql -u root -p -e "SHOW TABLES LIKE 'webhook%'" formhub

# Expected output:
# webhook_alerts
# webhook_analytics
# webhook_circuit_breakers
# webhook_health_checks
# webhook_logs
# webhook_monitor_events
# webhook_oauth_tokens
# webhook_rate_limits
# webhook_retry_queue
# webhook_security_settings
# webhook_templates
```

### Step 2: Build and Deploy Backend

1. **Install Go Dependencies**

```bash
# Install new dependencies
go mod download
go mod tidy
```

2. **Build the Application**

```bash
# Build for production
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o formhub-api main.go

# Or build for current OS
go build -o formhub-api main.go
```

3. **Deploy with Docker (Recommended)**

```bash
# Build Docker image
docker build -t formhub-enhanced:latest .

# Run with Docker Compose
docker-compose up -d
```

4. **Deploy Manually**

```bash
# Copy binary to server
scp formhub-api user@server:/opt/formhub/

# Create systemd service
sudo cp formhub-api.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable formhub-api
sudo systemctl start formhub-api
```

### Step 3: Configure Load Balancer

Update your Nginx configuration for enhanced webhook endpoints:

```nginx
upstream formhub_backend {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081; # Add multiple instances for scaling
}

server {
    listen 443 ssl http2;
    server_name api.formhub.io;

    # SSL configuration
    ssl_certificate /path/to/ssl/cert.pem;
    ssl_certificate_key /path/to/ssl/key.pem;

    # Enhanced webhook endpoints
    location /api/v1/forms/ {
        proxy_pass http://formhub_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Webhook-specific settings
        proxy_read_timeout 300;
        proxy_connect_timeout 30;
        proxy_send_timeout 300;
    }
    
    # WebSocket support for real-time monitoring
    location /api/v1/forms/*/webhooks/monitoring/ws {
        proxy_pass http://formhub_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }

    # Rate limiting for webhook API
    location /api/v1/integrations/ {
        limit_req zone=api burst=20 nodelay;
        proxy_pass http://formhub_backend;
    }
}
```

### Step 4: Configure Third-Party Integrations

1. **Google Sheets Integration**

```bash
# 1. Create Google Cloud Project
# 2. Enable Google Sheets API
# 3. Create Service Account
# 4. Download credentials JSON
# 5. Set environment variable
export GOOGLE_SHEETS_CREDENTIALS_JSON='{"type": "service_account", ...}'
```

2. **Slack Integration**

```bash
# 1. Create Slack App at api.slack.com
# 2. Get client credentials
export SLACK_CLIENT_ID="your-client-id"
export SLACK_CLIENT_SECRET="your-client-secret"
```

3. **Airtable Integration**

```bash
# 1. Get Personal Access Token from Airtable
export AIRTABLE_API_KEY="your-personal-access-token"
```

### Step 5: Verify Deployment

1. **Health Check**

```bash
# Check API health
curl -X GET "https://api.formhub.io/health"

# Expected response:
{
  "status": "healthy",
  "version": "2.0.0",
  "time": "2024-01-15T10:30:00Z"
}
```

2. **Test Enhanced Webhook System**

```bash
# Run comprehensive test suite
cd backend/test
python3 webhook_system_test.py --base-url https://api.formhub.io --api-key YOUR_API_KEY

# Expected output:
# FORMHUB WEBHOOK SYSTEM TEST REPORT
# =====================================
# SUMMARY: 45/45 tests passed (100.0%)
# 🎉 ALL TESTS PASSED! Webhook system is fully functional.
```

3. **Verify Database Population**

```bash
# Check marketplace integrations
mysql -u root -p -e "SELECT name, category, downloads FROM integration_marketplace" formhub

# Check webhook system tables
mysql -u root -p -e "SELECT COUNT(*) as webhook_logs FROM webhook_logs" formhub
```

## Performance Optimization

### 1. Redis Configuration

```bash
# Optimize Redis for webhook caching
redis-cli CONFIG SET maxmemory-policy allkeys-lru
redis-cli CONFIG SET maxmemory 2gb

# Enable persistence for analytics
redis-cli CONFIG SET save "900 1 300 10 60 10000"
```

### 2. MySQL Optimization

```sql
-- Optimize webhook-related tables
ALTER TABLE webhook_logs ADD INDEX idx_created_at_form_id (created_at, form_id);
ALTER TABLE webhook_analytics ADD INDEX idx_date_form_endpoint (date, form_id, endpoint_id);

-- Set appropriate MySQL settings
SET GLOBAL innodb_buffer_pool_size = 4294967296; -- 4GB
SET GLOBAL max_connections = 1000;
```

### 3. Application Scaling

```yaml
# Docker Compose scaling
version: '3.8'
services:
  formhub-api:
    image: formhub-enhanced:latest
    deploy:
      replicas: 3
      resources:
        limits:
          memory: 2G
          cpus: '1.0'
    environment:
      - WORKER_POOL_SIZE=20
      - MAX_CONCURRENT_WEBHOOKS=50
```

## Monitoring and Alerting

### 1. Application Monitoring

```bash
# Install monitoring tools
apt-get install prometheus node-exporter grafana

# Configure Prometheus to scrape FormHub metrics
# Add to prometheus.yml:
scrape_configs:
  - job_name: 'formhub'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

### 2. Log Monitoring

```bash
# Configure log rotation
cat > /etc/logrotate.d/formhub << EOF
/var/log/formhub/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0644 formhub formhub
}
EOF
```

### 3. Alerting Rules

```yaml
# Prometheus alerting rules
groups:
- name: formhub_webhook_alerts
  rules:
  - alert: WebhookHighFailureRate
    expr: (webhook_failed_total / webhook_total) > 0.1
    for: 5m
    annotations:
      summary: "High webhook failure rate detected"
      
  - alert: WebhookQueueBacklog
    expr: webhook_queue_size > 1000
    for: 2m
    annotations:
      summary: "Webhook queue backlog detected"
```

## Security Hardening

### 1. Network Security

```bash
# Configure firewall
ufw allow 80/tcp
ufw allow 443/tcp
ufw allow 22/tcp
ufw deny incoming
ufw enable

# Restrict database access
iptables -A INPUT -p tcp --dport 3306 -s localhost -j ACCEPT
iptables -A INPUT -p tcp --dport 3306 -j DROP
```

### 2. SSL/TLS Configuration

```bash
# Generate strong SSL certificate
certbot --nginx -d api.formhub.io

# Test SSL configuration
curl -I https://api.formhub.io
```

### 3. API Security

```bash
# Enable rate limiting in application
export RATE_LIMIT_ENABLED=true
export MAX_REQUESTS_PER_HOUR=10000

# Configure webhook signature verification
export WEBHOOK_SIGNATURE_REQUIRED=true
export DEFAULT_WEBHOOK_SECRET=$(openssl rand -hex 32)
```

## Backup and Recovery

### 1. Database Backup

```bash
# Create backup script
cat > /opt/formhub/backup.sh << 'EOF'
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
mysqldump -u root -p formhub > /backups/formhub_$DATE.sql
find /backups -name "formhub_*.sql" -mtime +7 -delete
EOF

chmod +x /opt/formhub/backup.sh

# Schedule daily backups
crontab -e
# Add: 0 2 * * * /opt/formhub/backup.sh
```

### 2. Redis Backup

```bash
# Configure Redis persistence
echo "save 900 1" >> /etc/redis/redis.conf
echo "save 300 10" >> /etc/redis/redis.conf
echo "save 60 10000" >> /etc/redis/redis.conf

systemctl restart redis
```

## Troubleshooting

### Common Issues

1. **Webhook Delivery Failures**

```bash
# Check webhook logs
docker logs formhub-api | grep "webhook"

# Check database for failed webhooks
mysql -u root -p -e "SELECT * FROM webhook_logs WHERE success = 0 ORDER BY created_at DESC LIMIT 10" formhub
```

2. **Integration Authentication Errors**

```bash
# Test Google Sheets credentials
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/credentials.json"
go run test/integration_test.go -service=google_sheets

# Test Slack webhook
curl -X POST -H 'Content-type: application/json' \
  --data '{"text":"Test message"}' \
  YOUR_SLACK_WEBHOOK_URL
```

3. **Performance Issues**

```bash
# Check system resources
htop
iostat -x 1

# Monitor webhook queue
redis-cli LLEN webhook_queue

# Check database performance
mysql -u root -p -e "SHOW PROCESSLIST" formhub
```

### Debug Commands

```bash
# Enable debug logging
export LOG_LEVEL=debug

# Check webhook endpoint health
curl -X POST "https://api.formhub.io/api/v1/forms/FORM_ID/webhooks/endpoints/ENDPOINT_ID/test" \
  -H "Authorization: Bearer YOUR_API_KEY"

# Monitor real-time webhook activity
curl -X GET "https://api.formhub.io/api/v1/forms/FORM_ID/webhooks/monitoring" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

## Rollback Plan

If issues arise, follow this rollback procedure:

```bash
# 1. Stop the new service
systemctl stop formhub-api

# 2. Restore previous version
cp /opt/formhub/formhub-api.backup /opt/formhub/formhub-api

# 3. Rollback database changes (if necessary)
mysql -u root -p formhub < /backups/pre_upgrade_backup.sql

# 4. Start the service
systemctl start formhub-api

# 5. Verify functionality
curl -X GET "https://api.formhub.io/health"
```

## Post-Deployment Tasks

### 1. Documentation Updates

- Update API documentation with new endpoints
- Create user guides for new features
- Update integration setup instructions

### 2. User Training

- Train support team on new webhook features
- Create video tutorials for customers
- Update help documentation

### 3. Monitoring Setup

- Configure alerting rules
- Set up monitoring dashboards
- Establish SLA metrics

## Success Metrics

Track these metrics to measure deployment success:

- **Webhook Delivery Success Rate**: > 99%
- **Average Response Time**: < 200ms
- **System Uptime**: > 99.9%
- **Integration Adoption**: Track usage of marketplace integrations
- **Customer Satisfaction**: Monitor support tickets related to webhooks

## Support and Maintenance

### Regular Maintenance Tasks

```bash
# Weekly tasks
- Review webhook performance metrics
- Check error logs and investigate failures
- Update third-party integration credentials if needed

# Monthly tasks  
- Analyze webhook usage patterns
- Review and optimize database performance
- Update system dependencies

# Quarterly tasks
- Security audit and penetration testing
- Capacity planning and scaling review
- Disaster recovery testing
```

### Support Contacts

- **Technical Issues**: tech@formhub.io
- **Integration Issues**: integrations@formhub.io
- **Emergency Support**: +1-800-FORMHUB

---

## Deployment Checklist

- [ ] System requirements verified
- [ ] Database migrations applied
- [ ] Application built and deployed
- [ ] Load balancer configured
- [ ] Third-party integrations configured
- [ ] SSL certificates installed
- [ ] Monitoring and alerting configured
- [ ] Backup procedures implemented
- [ ] Security hardening applied
- [ ] Comprehensive testing completed
- [ ] Documentation updated
- [ ] Team training completed

**Deployment Status**: ✅ **PRODUCTION READY**

*The FormHub Enhanced Webhook System has been successfully implemented and is ready for production deployment with enterprise-grade features, comprehensive monitoring, and extensive third-party integrations.*