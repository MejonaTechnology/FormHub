# FormHub - Form Backend Service Platform

A comprehensive form backend service similar to Web3Forms, designed to serve clients with a white-label SaaS solution.

## 🚀 Quick Start

### Backend API Service
```bash
cd backend
go mod tidy
go run main.go
```

### Frontend Dashboard
```bash
cd frontend
npm install
npm run dev
```

## 📋 Features

### Core Features (MVP)
- ✅ Form Submission API
- ✅ Email Notifications
- ✅ Basic Spam Protection
- ✅ Client Dashboard
- ✅ API Key Management

### Advanced Features
- 🔄 Webhook Integration
- 📁 File Upload Handling
- 🛡️ Advanced Anti-Spam (reCAPTCHA v3)
- 📧 Custom Email Templates
- 📊 Analytics Dashboard

## 🏗️ Architecture

### Technology Stack
- **Backend**: Go with Gin framework
- **Database**: PostgreSQL + Redis
- **Frontend**: Next.js 15 with TypeScript
- **Email**: AWS SES / SendGrid
- **Deployment**: Docker + AWS/Railway

### API Endpoints
```
POST /api/v1/submit        - Form submission endpoint
GET  /api/v1/forms         - List client forms
POST /api/v1/forms         - Create new form
GET  /api/v1/submissions   - Get form submissions
POST /api/v1/webhooks      - Configure webhooks
```

## 💰 Pricing Model

- **Free**: 100 submissions/month, 1 form
- **Starter ($9/month)**: 1,000 submissions/month, 5 forms
- **Professional ($29/month)**: 10,000 submissions/month, unlimited forms
- **Enterprise ($99/month)**: 100,000 submissions/month, white-label

## 📖 Documentation

- [API Reference](./docs/api.md)
- [Integration Guide](./docs/integration.md)
- [Deployment Guide](./docs/deployment.md)

---

**Built with ❤️ by Mejona Technology LLP**