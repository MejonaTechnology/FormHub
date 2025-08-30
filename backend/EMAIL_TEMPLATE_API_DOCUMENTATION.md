# FormHub Email Template System API Documentation

## Overview

FormHub's Email Template System provides a comprehensive, production-ready solution for email marketing and form notifications, similar to Web3Forms but with enhanced capabilities including:

- **Advanced Template Engine** with HTML/text support and variables
- **Multiple Email Providers** (SMTP, SendGrid, Mailgun, AWS SES, Postmark)
- **Intelligent Autoresponders** with conditional logic and scheduling
- **Email Queue System** with retry mechanisms and monitoring
- **Drag-and-Drop Template Builder** with responsive designs
- **Email Analytics** with tracking and reporting
- **A/B Testing** for email optimization
- **Multi-language Support** and template inheritance

## Base URL

All API endpoints are prefixed with:
```
https://your-formhub-instance.com/api/v1
```

## Authentication

All email template endpoints require authentication using JWT tokens:

```bash
Authorization: Bearer <jwt_token>
```

## API Endpoints

### Email Templates

#### Create Email Template
**POST** `/email/templates`

Create a new email template with HTML/text content and variables.

```json
{
  "name": "Welcome Email",
  "description": "Welcome new users to our platform",
  "type": "welcome",
  "language": "en",
  "subject": "Welcome to {{company_name}}, {{name}}!",
  "html_content": "<!DOCTYPE html><html><body><h1>Welcome {{name}}!</h1><p>Thank you for joining {{company_name}}. We're excited to have you on board.</p></body></html>",
  "text_content": "Welcome {{name}}! Thank you for joining {{company_name}}.",
  "variables": ["name", "company_name", "email"],
  "tags": ["welcome", "onboarding"]
}
```

**Response:**
```json
{
  "success": true,
  "template": {
    "id": "uuid-template-id",
    "name": "Welcome Email",
    "type": "welcome",
    "subject": "Welcome to {{company_name}}, {{name}}!",
    "variables": ["name", "company_name", "email"],
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

#### List Email Templates
**GET** `/email/templates`

Query Parameters:
- `type` - Filter by template type
- `form_id` - Filter by form ID
- `language` - Filter by language

**Response:**
```json
{
  "success": true,
  "templates": [
    {
      "id": "uuid-template-id",
      "name": "Welcome Email",
      "type": "welcome",
      "language": "en",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### Get Template
**GET** `/email/templates/{id}`

**Response:**
```json
{
  "success": true,
  "template": {
    "id": "uuid-template-id",
    "name": "Welcome Email",
    "description": "Welcome new users to our platform",
    "type": "welcome",
    "subject": "Welcome to {{company_name}}, {{name}}!",
    "html_content": "...",
    "text_content": "...",
    "variables": ["name", "company_name", "email"]
  }
}
```

#### Preview Template
**POST** `/email/templates/{id}/preview`

Preview a template with sample data.

```json
{
  "variables": {
    "name": "John Doe",
    "company_name": "Acme Corp",
    "email": "john@acme.com"
  }
}
```

**Response:**
```json
{
  "success": true,
  "rendered": {
    "subject": "Welcome to Acme Corp, John Doe!",
    "html_content": "<!DOCTYPE html><html><body><h1>Welcome John Doe!</h1>...",
    "text_content": "Welcome John Doe! Thank you for joining Acme Corp."
  }
}
```

#### Clone Template
**POST** `/email/templates/{id}/clone`

```json
{
  "name": "Welcome Email - Copy"
}
```

### Email Providers

#### Create Email Provider
**POST** `/email/providers`

Configure email service providers (SMTP, SendGrid, Mailgun, etc.).

```json
{
  "name": "Primary SMTP",
  "type": "smtp",
  "config": {
    "host": "smtp.gmail.com",
    "port": 587,
    "username": "your-email@gmail.com",
    "password": "your-password",
    "use_tls": true,
    "from_name": "Your Company",
    "from_email": "noreply@yourcompany.com"
  },
  "is_default": true
}
```

**SendGrid Configuration:**
```json
{
  "name": "SendGrid",
  "type": "sendgrid",
  "config": {
    "api_key": "your-sendgrid-api-key",
    "from_name": "Your Company",
    "from_email": "noreply@yourcompany.com"
  }
}
```

#### List Email Providers
**GET** `/email/providers`

**Response:**
```json
{
  "success": true,
  "providers": [
    {
      "id": "uuid-provider-id",
      "name": "Primary SMTP",
      "type": "smtp",
      "is_active": true,
      "is_default": true
    }
  ]
}
```

#### Test Email Provider
**POST** `/email/providers/{id}/test`

Test if the email provider configuration is working.

**Response:**
```json
{
  "success": true,
  "message": "Provider test successful"
}
```

### Autoresponders

#### Create Autoresponder
**POST** `/email/autoresponders`

Create intelligent autoresponders with conditional logic.

```json
{
  "form_id": "uuid-form-id",
  "name": "Welcome Autoresponder",
  "template_id": "uuid-template-id",
  "delay_minutes": 0,
  "send_to_field": "email",
  "conditions": {
    "field_conditions": [
      {
        "field_name": "email",
        "operator": "exists"
      },
      {
        "field_name": "country",
        "operator": "equals",
        "value": "US"
      }
    ],
    "logical_operator": "AND"
  },
  "track_opens": true,
  "track_clicks": true
}
```

**Conditional Operators:**
- `equals`, `not_equals`
- `contains`, `not_contains`
- `starts_with`, `ends_with`
- `in`, `not_in`
- `exists`, `not_exists`
- `greater_than`, `less_than`
- `regex`

#### Time-Based Conditions
```json
{
  "conditions": {
    "time_conditions": {
      "start_time": "09:00",
      "end_time": "17:00",
      "days": ["monday", "tuesday", "wednesday", "thursday", "friday"],
      "timezone": "America/New_York"
    }
  }
}
```

#### List Autoresponders
**GET** `/email/autoresponders?form_id={form_id}`

#### Toggle Autoresponder
**POST** `/email/autoresponders/{id}/toggle`

```json
{
  "enabled": true
}
```

### Email Queue

#### Get Queue Statistics
**GET** `/email/queue/stats`

**Response:**
```json
{
  "success": true,
  "stats": {
    "pending": 25,
    "scheduled": 10,
    "sending": 5,
    "sent": 1000,
    "failed": 15,
    "total": 1055
  }
}
```

#### List Queued Emails
**GET** `/email/queue/emails`

Query Parameters:
- `status` - Filter by email status
- `limit` - Number of results (default: 50)
- `offset` - Pagination offset

#### Process Queue Manually
**POST** `/email/queue/process`

Process pending emails immediately.

**Response:**
```json
{
  "success": true,
  "result": {
    "processed": 25,
    "sent": 22,
    "failed": 3,
    "errors": ["SMTP timeout for recipient@example.com"]
  }
}
```

### Email Analytics

#### Get Template Analytics
**GET** `/email/templates/{id}/analytics`

Query Parameters:
- `start_date` - Start date (YYYY-MM-DD)
- `end_date` - End date (YYYY-MM-DD)

**Response:**
```json
{
  "success": true,
  "report": {
    "template_id": "uuid-template-id",
    "template_name": "Welcome Email",
    "total_sent": 1000,
    "total_delivered": 985,
    "total_opened": 456,
    "total_clicked": 123,
    "open_rate": 46.3,
    "click_rate": 12.5,
    "delivery_rate": 98.5,
    "time_series_data": [
      {
        "date": "2024-01-01",
        "sent": 50,
        "opened": 23,
        "clicked": 8
      }
    ],
    "top_links": [
      {
        "url": "https://example.com/signup",
        "clicks": 89,
        "click_rate": 9.0
      }
    ]
  }
}
```

#### Get User Analytics Overview
**GET** `/email/analytics/overview`

Get overall email performance for the user.

#### Get Top Performing Templates
**GET** `/email/analytics/top-templates?limit=10`

### Template Builder

#### Get Available Components
**GET** `/email/builder/components`

Get drag-and-drop components available for template building.

**Response:**
```json
{
  "success": true,
  "components": {
    "text": {
      "name": "Text Block",
      "description": "Rich text content with formatting options",
      "icon": "text",
      "properties": {
        "content": {
          "type": "richtext",
          "placeholder": "Text content"
        }
      }
    },
    "button": {
      "name": "Button",
      "description": "Call-to-action button",
      "properties": {
        "text": {"type": "text"},
        "url": {"type": "url"},
        "target": {"type": "select", "options": ["_self", "_blank"]}
      }
    }
  }
}
```

#### Create Template Design
**POST** `/email/builder/designs`

Create a design using the drag-and-drop builder.

```json
{
  "name": "Modern Newsletter",
  "description": "Clean modern newsletter design",
  "components": [
    {
      "id": "header-1",
      "type": "header",
      "content": "{{company_name}}",
      "properties": {
        "logo": "https://example.com/logo.png"
      },
      "styles": {
        "background_color": "#f8f9fa",
        "text_align": "center",
        "padding": "20px"
      },
      "order": 1
    },
    {
      "id": "text-1",
      "type": "text",
      "content": "<h1>Hello {{name}}!</h1><p>Welcome to our newsletter.</p>",
      "styles": {
        "padding": "20px",
        "font_family": "Arial, sans-serif"
      },
      "order": 2
    }
  ],
  "global_styles": {
    "container_width": "600px",
    "background_color": "#ffffff",
    "default_font_family": "Arial, sans-serif"
  },
  "category": "newsletter",
  "tags": ["modern", "clean"]
}
```

#### Generate Preview
**POST** `/email/builder/preview`

Generate HTML preview of a template design.

```json
{
  "components": [...],
  "global_styles": {...},
  "variables": {
    "company_name": "Acme Corp",
    "name": "John Doe"
  }
}
```

**Returns:** HTML content

### A/B Testing

#### Create A/B Test
**POST** `/email/ab-tests`

```json
{
  "name": "Subject Line Test",
  "description": "Testing different subject lines",
  "template_a_id": "uuid-template-a",
  "template_b_id": "uuid-template-b",
  "traffic_split": 50,
  "test_metric": "open_rate",
  "min_sample_size": 1000
}
```

#### Start A/B Test
**POST** `/email/ab-tests/{id}/start`

#### Get A/B Test Results
**GET** `/email/ab-tests/{id}/results`

**Response:**
```json
{
  "success": true,
  "result": {
    "test": {
      "id": "uuid-test-id",
      "name": "Subject Line Test",
      "status": "ended"
    },
    "is_significant": true,
    "confidence": 95.2,
    "winner": "A",
    "improvement_percentage": 15.3,
    "recommendation": "Template A is the clear winner with 15.3% better performance and 95.2% confidence."
  }
}
```

## Integration Examples

### 1. Basic Form with Autoresponder

```javascript
// 1. Create an email template
const template = await fetch('/api/v1/email/templates', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer ' + token,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    name: 'Contact Form Thank You',
    type: 'autoresponder',
    subject: 'Thank you for contacting us, {{name}}!',
    html_content: `
      <h1>Thank you {{name}}!</h1>
      <p>We received your message: "{{message}}"</p>
      <p>We'll get back to you within 24 hours.</p>
    `,
    variables: ['name', 'message']
  })
});

// 2. Set up autoresponder
const autoresponder = await fetch('/api/v1/email/autoresponders', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer ' + token,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    form_id: 'your-form-id',
    template_id: template.id,
    send_to_field: 'email',
    delay_minutes: 0,
    conditions: {
      field_conditions: [
        { field_name: 'email', operator: 'exists' }
      ]
    }
  })
});
```

### 2. Advanced Template with Conditions

```javascript
const smartAutoresponder = await fetch('/api/v1/email/autoresponders', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer ' + token,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    form_id: 'contact-form',
    template_id: 'welcome-template',
    send_to_field: 'email',
    delay_minutes: 5, // 5-minute delay
    conditions: {
      field_conditions: [
        { field_name: 'email', operator: 'exists' },
        { field_name: 'interest', operator: 'in', values: ['product', 'demo'] },
        { field_name: 'company_size', operator: 'greater_than', value: '10' }
      ],
      time_conditions: {
        start_time: '09:00',
        end_time: '17:00',
        days: ['monday', 'tuesday', 'wednesday', 'thursday', 'friday']
      },
      logical_operator: 'AND'
    }
  })
});
```

### 3. A/B Testing Implementation

```javascript
// Create two templates for testing
const templateA = await createTemplate({
  name: 'Subject A - Direct Approach',
  subject: 'Your form submission was received',
  html_content: '...'
});

const templateB = await createTemplate({
  name: 'Subject B - Personal Approach', 
  subject: 'Thanks for reaching out, {{name}}!',
  html_content: '...'
});

// Start A/B test
const abTest = await fetch('/api/v1/email/ab-tests', {
  method: 'POST',
  body: JSON.stringify({
    name: 'Subject Line Personalization Test',
    template_a_id: templateA.id,
    template_b_id: templateB.id,
    traffic_split: 50
  })
});

await fetch(`/api/v1/email/ab-tests/${abTest.id}/start`, {
  method: 'POST'
});
```

### 4. Multi-Provider Setup

```javascript
// Configure multiple email providers for redundancy
const providers = [
  {
    name: 'Primary SendGrid',
    type: 'sendgrid',
    config: {
      api_key: 'your-sendgrid-key',
      from_email: 'noreply@yourcompany.com'
    },
    is_default: true
  },
  {
    name: 'Backup SMTP',
    type: 'smtp',
    config: {
      host: 'smtp.gmail.com',
      port: 587,
      username: 'backup@yourcompany.com',
      password: 'password'
    }
  }
];

for (const provider of providers) {
  await fetch('/api/v1/email/providers', {
    method: 'POST',
    body: JSON.stringify(provider)
  });
}
```

## Error Handling

All endpoints return structured error responses:

```json
{
  "success": false,
  "error": "Template validation failed: HTML content contains script tags"
}
```

Common HTTP status codes:
- `200` - Success
- `201` - Created
- `400` - Bad Request (validation error)
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `429` - Rate Limited
- `500` - Internal Server Error

## Rate Limiting

API endpoints are rate limited to prevent abuse:
- **Template operations**: 100 requests per hour
- **Email sending**: Based on your plan
- **Analytics**: 500 requests per hour

## Webhooks

FormHub can send webhooks for email events:

```json
{
  "event": "email.opened",
  "timestamp": "2024-01-01T12:00:00Z",
  "data": {
    "queue_id": "uuid-queue-id",
    "template_id": "uuid-template-id",
    "recipient": "user@example.com",
    "user_agent": "Mozilla/5.0...",
    "ip_address": "192.168.1.1"
  }
}
```

Supported events:
- `email.sent`
- `email.delivered`
- `email.opened`
- `email.clicked`
- `email.bounced`
- `email.failed`

## Security Features

- **Authentication**: JWT-based authentication required
- **Input Validation**: All inputs are validated and sanitized
- **Rate Limiting**: Prevents abuse and spam
- **Content Security**: Templates are scanned for malicious content
- **Data Encryption**: Sensitive data encrypted at rest
- **Audit Logging**: All actions are logged for security

## Performance Optimization

- **Queue Processing**: Background processing with retry mechanisms
- **Template Caching**: Compiled templates are cached
- **CDN Integration**: Static assets served via CDN
- **Database Optimization**: Indexed queries for fast retrieval
- **Connection Pooling**: Efficient database connections

## Monitoring & Health Checks

Health check endpoint:
```
GET /health
```

Monitor email queue health:
```
GET /api/v1/email/queue/stats
```

This comprehensive email template system provides enterprise-grade functionality while maintaining ease of use, making it perfect for both simple form notifications and complex email marketing campaigns.