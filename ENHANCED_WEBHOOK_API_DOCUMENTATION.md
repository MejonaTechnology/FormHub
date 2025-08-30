# FormHub Enhanced Webhook System API Documentation

## Overview

FormHub's Enhanced Webhook System provides comprehensive webhook and third-party integration capabilities with enterprise-grade features including:

- **Multiple webhook endpoints per form** with conditional triggering
- **Advanced retry logic** with exponential backoff and circuit breaker
- **Third-party integrations** (Google Sheets, Airtable, Notion, Slack, Discord, Telegram, Zapier)
- **Real-time analytics and monitoring** with health checks
- **Enterprise features** including load balancing, failover, and security
- **Integration marketplace** with pre-built templates

## Base URL

```
https://api.formhub.io/api/v1
```

## Authentication

All API requests require authentication using API keys:

```bash
Authorization: Bearer YOUR_API_KEY
```

## Webhook Endpoint Management

### Create Webhook Endpoint

Create a new webhook endpoint for a form.

```http
POST /forms/{formId}/webhooks/endpoints
```

**Request Body:**
```json
{
  "name": "My Webhook",
  "url": "https://example.com/webhook",
  "secret": "optional-secret-key",
  "events": ["form.submitted", "form.updated"],
  "headers": {
    "X-Custom-Header": "value"
  },
  "content_type": "application/json",
  "method": "POST",
  "timeout": 30,
  "max_retries": 3,
  "retry_delay": 5,
  "enabled": true,
  "rate_limit_enabled": true,
  "verify_ssl": true,
  "custom_payload": "{\"form_id\": \"{{.FormID}}\", \"data\": {{.Data}}}",
  "transform_config": {
    "field_mappings": {
      "email": "user_email",
      "name": "full_name"
    },
    "data_filters": [
      {
        "type": "exclude",
        "value": ["internal_field"]
      }
    ]
  },
  "conditional_rules": [
    {
      "field": "email",
      "operator": "contains",
      "value": "@company.com",
      "logic_op": "and"
    }
  ],
  "priority": 1,
  "tags": ["important", "customer"]
}
```

**Response:**
```json
{
  "success": true,
  "message": "Webhook endpoint created successfully",
  "endpoint": {
    "id": "endpoint-uuid",
    "name": "My Webhook",
    "url": "https://example.com/webhook",
    "events": ["form.submitted", "form.updated"],
    "enabled": true,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

### Get Webhook Endpoints

Retrieve all webhook endpoints for a form.

```http
GET /forms/{formId}/webhooks/endpoints
```

**Response:**
```json
{
  "success": true,
  "endpoints": [
    {
      "id": "endpoint-uuid",
      "name": "My Webhook",
      "url": "https://example.com/webhook",
      "events": ["form.submitted"],
      "enabled": true,
      "priority": 1,
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### Update Webhook Endpoint

Update an existing webhook endpoint.

```http
PUT /forms/{formId}/webhooks/endpoints/{endpointId}
```

### Delete Webhook Endpoint

Delete a webhook endpoint.

```http
DELETE /forms/{formId}/webhooks/endpoints/{endpointId}
```

### Test Webhook Endpoint

Test a webhook endpoint with a sample payload.

```http
POST /forms/{formId}/webhooks/endpoints/{endpointId}/test
```

**Response:**
```json
{
  "success": true,
  "result": {
    "endpoint_id": "endpoint-uuid",
    "success": true,
    "status_code": 200,
    "response_time": "250ms",
    "response": "OK",
    "tested_at": "2024-01-15T10:35:00Z"
  }
}
```

## Webhook Analytics

### Get Webhook Analytics

Retrieve comprehensive webhook analytics for a form.

```http
GET /forms/{formId}/webhooks/analytics?start=2024-01-01T00:00:00Z&end=2024-01-15T23:59:59Z
```

**Response:**
```json
{
  "success": true,
  "analytics": {
    "form_id": "form-uuid",
    "time_range": {
      "start": "2024-01-01T00:00:00Z",
      "end": "2024-01-15T23:59:59Z"
    },
    "total_webhooks": 1250,
    "successful_sent": 1198,
    "failed": 52,
    "success_rate": 95.84,
    "avg_response_time": "180ms",
    "endpoint_stats": [
      {
        "endpoint_id": "endpoint-uuid",
        "name": "My Webhook",
        "url": "https://example.com/webhook",
        "total_requests": 500,
        "successful_requests": 485,
        "success_rate": 97.0,
        "avg_response_time": "150ms"
      }
    ],
    "hourly_stats": [
      {
        "hour": "2024-01-15T10:00:00Z",
        "total_requests": 45,
        "successful_requests": 43,
        "failed_requests": 2,
        "avg_response_time": "165ms"
      }
    ],
    "error_breakdown": {
      "timeout": 25,
      "connection": 15,
      "server_error": 12
    },
    "response_codes": {
      "200": 1150,
      "404": 30,
      "500": 22
    },
    "performance_metrics": {
      "p50_response_time": "120ms",
      "p90_response_time": "280ms",
      "p95_response_time": "350ms",
      "p99_response_time": "500ms"
    }
  }
}
```

### Get Real-time Webhook Stats

Get current real-time webhook statistics.

```http
GET /forms/{formId}/webhooks/stats/realtime
```

**Response:**
```json
{
  "success": true,
  "stats": {
    "form_id": "form-uuid",
    "total_requests": 1250,
    "successful_requests": 1198,
    "failed_requests": 52,
    "success_rate": 95.84,
    "last_request": "2024-01-15T10:30:00Z",
    "queue_size": 3,
    "timestamp": "2024-01-15T10:35:00Z"
  }
}
```

## Webhook Monitoring

### Get Webhook Monitoring Data

Retrieve real-time monitoring information.

```http
GET /forms/{formId}/webhooks/monitoring
```

**Response:**
```json
{
  "success": true,
  "monitoring": {
    "form_id": "form-uuid",
    "status": "healthy",
    "active_endpoints": 3,
    "failing_endpoints": 0,
    "current_load": 5,
    "queue_size": 2,
    "recent_events": [
      {
        "type": "status_change",
        "timestamp": "2024-01-15T10:30:00Z",
        "endpoint_id": "endpoint-uuid",
        "severity": "info",
        "message": "Endpoint status changed from unhealthy to healthy"
      }
    ],
    "health_checks": [
      {
        "endpoint_id": "endpoint-uuid",
        "status": "healthy",
        "last_check": "2024-01-15T10:34:00Z",
        "response_time": "120ms",
        "success_rate": 98.5
      }
    ],
    "alerts": [
      {
        "id": "alert-uuid",
        "type": "high_failure_rate",
        "severity": "warning",
        "message": "High failure rate detected: 15%",
        "timestamp": "2024-01-15T09:45:00Z",
        "acknowledged": false
      }
    ]
  }
}
```

### WebSocket Real-time Monitoring

Connect to real-time monitoring via WebSocket.

```javascript
const ws = new WebSocket('wss://api.formhub.io/api/v1/forms/FORM_ID/webhooks/monitoring/ws');

ws.onmessage = function(event) {
  const monitorEvent = JSON.parse(event.data);
  console.log('Monitoring event:', monitorEvent);
};
```

## Third-Party Integrations

### List Available Integrations

Get all available third-party integrations.

```http
GET /integrations
```

**Response:**
```json
{
  "success": true,
  "integrations": [
    {
      "name": "slack",
      "schema": {
        "name": "Slack",
        "description": "Send notifications to Slack channels",
        "version": "2.1.0",
        "fields": [
          {
            "name": "webhook_url",
            "type": "string",
            "required": true,
            "description": "Slack webhook URL"
          },
          {
            "name": "channel",
            "type": "string",
            "required": true,
            "description": "Slack channel name"
          }
        ]
      }
    }
  ]
}
```

### Get Integration Schema

Get detailed schema for a specific integration.

```http
GET /integrations/{integrationName}/schema
```

### Test Integration Configuration

Test an integration configuration.

```http
POST /integrations/{integrationName}/test
```

**Request Body:**
```json
{
  "webhook_url": "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
  "channel": "#general",
  "username": "FormHub Bot"
}
```

### Send Test Event to Integration

Send a test event to an integration.

```http
POST /integrations/{integrationName}/send
```

**Request Body:**
```json
{
  "config": {
    "webhook_url": "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
    "channel": "#general"
  },
  "event": {
    "id": "test-event",
    "type": "test",
    "form_id": "form-uuid",
    "data": {
      "name": "John Doe",
      "email": "john@example.com"
    }
  }
}
```

## Integration Marketplace

### List Marketplace Integrations

Browse available integrations in the marketplace.

```http
GET /marketplace/integrations?category=communication&popular=true&limit=10
```

**Query Parameters:**
- `category`: Filter by category (communication, productivity, database, etc.)
- `popular`: Show only popular integrations (true/false)
- `featured`: Show only featured integrations (true/false)
- `min_rating`: Minimum rating filter (0-5)
- `sort_by`: Sort by (name, rating, downloads, created_at)
- `sort_order`: Sort order (asc, desc)
- `limit`: Number of results to return

**Response:**
```json
{
  "success": true,
  "integrations": [
    {
      "id": "slack",
      "name": "Slack",
      "description": "Send form notifications to Slack channels",
      "category": "communication",
      "version": "2.1.0",
      "author": "FormHub",
      "icon": "https://cdn.formhub.io/icons/slack.svg",
      "tags": ["slack", "notification", "communication"],
      "popular": true,
      "featured": true,
      "downloads": 12847,
      "rating": 4.7
    }
  ]
}
```

### Get Marketplace Categories

Get available integration categories.

```http
GET /marketplace/categories
```

**Response:**
```json
{
  "success": true,
  "categories": {
    "communication": ["slack", "discord", "telegram"],
    "productivity": ["google_sheets", "notion"],
    "database": ["airtable"],
    "automation": ["zapier"]
  }
}
```

### Install Marketplace Integration

Install a marketplace integration for a form.

```http
POST /forms/{formId}/marketplace/integrations/{integrationId}/install
```

**Request Body:**
```json
{
  "webhook_url": "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
  "channel": "#general",
  "username": "FormHub Bot"
}
```

## Event Types

FormHub supports various event types for webhook triggers:

### Form Events
- `form.created` - New form created
- `form.updated` - Form configuration updated
- `form.deleted` - Form deleted
- `form.published` - Form published
- `form.unpublished` - Form unpublished

### Submission Events
- `form.submitted` - New form submission received
- `submission.updated` - Submission data updated
- `submission.deleted` - Submission deleted
- `submission.approved` - Submission approved
- `submission.rejected` - Submission rejected

### System Events
- `webhook.test` - Webhook test event
- `integration.connected` - Third-party integration connected
- `integration.error` - Integration error occurred

## Webhook Payload Structure

All webhook payloads follow this structure:

```json
{
  "id": "event-uuid",
  "type": "form.submitted",
  "timestamp": "2024-01-15T10:30:00Z",
  "form_id": "form-uuid",
  "submission_id": "submission-uuid",
  "user_id": "user-uuid",
  "data": {
    "name": "John Doe",
    "email": "john@example.com",
    "message": "Hello world"
  },
  "metadata": {
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0...",
    "referrer": "https://example.com/contact"
  },
  "source": "form_submission",
  "version": "2.0",
  "event_sequence": 12345,
  "correlation_id": "corr-uuid",
  "environment": "production",
  "geolocation": {
    "country": "US",
    "region": "CA",
    "city": "San Francisco"
  },
  "device_info": {
    "browser": "Chrome",
    "os": "macOS",
    "is_mobile": false
  }
}
```

## Webhook Security

### Signature Verification

All webhook payloads are signed using HMAC-SHA256:

```javascript
const crypto = require('crypto');

function verifyWebhookSignature(payload, signature, secret) {
  const expectedSignature = crypto
    .createHmac('sha256', secret)
    .update(payload, 'utf8')
    .digest('hex');
  
  return signature === `sha256=${expectedSignature}`;
}
```

**Headers:**
```
X-FormHub-Signature-256: sha256=abc123def456...
X-FormHub-Timestamp: 1642248600
X-FormHub-Endpoint-ID: endpoint-uuid
```

### IP Whitelisting

Configure IP restrictions for webhooks:

```http
PUT /forms/{formId}/webhooks/security
```

**Request Body:**
```json
{
  "allowed_ips": ["192.168.1.0/24", "10.0.0.1"],
  "blocked_ips": ["1.2.3.4"],
  "require_signature": true,
  "max_payload_size": 5242880,
  "rate_limit_enabled": true,
  "rate_limit_requests": 100,
  "rate_limit_window": 3600
}
```

## Error Handling

### Error Response Format

```json
{
  "success": false,
  "error": "Invalid webhook configuration",
  "details": "URL is required",
  "code": "VALIDATION_ERROR",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### HTTP Status Codes

- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `409` - Conflict
- `429` - Too Many Requests
- `500` - Internal Server Error

## Rate Limits

API endpoints are rate-limited:

- **Webhook Management:** 1000 requests/hour per API key
- **Analytics:** 500 requests/hour per API key
- **Testing:** 100 requests/hour per endpoint
- **Webhook Delivery:** 10,000 requests/hour per form

Rate limit headers:
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 995
X-RateLimit-Reset: 1642252200
```

## SDKs and Examples

### JavaScript SDK

```bash
npm install @formhub/webhook-sdk
```

```javascript
import { FormHubWebhooks } from '@formhub/webhook-sdk';

const webhooks = new FormHubWebhooks({
  apiKey: 'your-api-key',
  baseURL: 'https://api.formhub.io'
});

// Create webhook endpoint
const endpoint = await webhooks.createEndpoint('form-id', {
  name: 'My Webhook',
  url: 'https://example.com/webhook',
  events: ['form.submitted']
});

// Get analytics
const analytics = await webhooks.getAnalytics('form-id', {
  start: new Date('2024-01-01'),
  end: new Date('2024-01-15')
});
```

### Python SDK

```bash
pip install formhub-webhooks
```

```python
from formhub_webhooks import WebhookClient

client = WebhookClient(api_key='your-api-key')

# Create webhook endpoint
endpoint = client.create_endpoint(
    form_id='form-id',
    name='My Webhook',
    url='https://example.com/webhook',
    events=['form.submitted']
)

# Get real-time stats
stats = client.get_realtime_stats('form-id')
print(f"Success rate: {stats['success_rate']:.1f}%")
```

### Go SDK

```go
import "github.com/formhub/webhook-go"

client := webhook.NewClient("your-api-key")

endpoint := &webhook.Endpoint{
    Name: "My Webhook",
    URL:  "https://example.com/webhook",
    Events: []string{"form.submitted"},
}

result, err := client.CreateEndpoint("form-id", endpoint)
if err != nil {
    log.Fatal(err)
}
```

## Best Practices

### Webhook Endpoint Implementation

1. **Idempotency**: Handle duplicate webhook deliveries
2. **Fast Response**: Return 200 OK quickly, process asynchronously
3. **Error Handling**: Return appropriate HTTP status codes
4. **Logging**: Log all webhook attempts for debugging
5. **Security**: Always verify webhook signatures

```javascript
// Express.js webhook handler example
app.post('/webhook', express.raw({type: 'application/json'}), (req, res) => {
  const signature = req.get('X-FormHub-Signature-256');
  const payload = req.body;
  
  // Verify signature
  if (!verifySignature(payload, signature, process.env.WEBHOOK_SECRET)) {
    return res.status(401).send('Unauthorized');
  }
  
  // Respond quickly
  res.status(200).send('OK');
  
  // Process asynchronously
  setImmediate(() => {
    processWebhookPayload(JSON.parse(payload));
  });
});
```

### Integration Configuration

1. **Test First**: Always test integrations before enabling
2. **Monitor Health**: Set up alerts for integration failures
3. **Backup Plans**: Configure multiple endpoints for critical workflows
4. **Rate Limiting**: Respect third-party API rate limits
5. **Error Recovery**: Implement proper retry and fallback logic

### Performance Optimization

1. **Conditional Webhooks**: Use rules to reduce unnecessary calls
2. **Payload Optimization**: Only include necessary data
3. **Batch Processing**: Group multiple events when possible
4. **Caching**: Cache integration responses when appropriate
5. **Load Balancing**: Distribute load across multiple endpoints

## Troubleshooting

### Common Issues

**Webhook Not Firing:**
- Check webhook endpoint configuration
- Verify event type matches
- Review conditional rules
- Check form submission process

**Integration Failures:**
- Verify API credentials
- Check rate limiting status
- Review integration logs
- Test with simple payload

**Performance Issues:**
- Monitor response times
- Check queue sizes
- Review analytics data
- Scale infrastructure if needed

### Debug Tools

Use the built-in testing tools to debug webhook issues:

```bash
# Test webhook endpoint
curl -X POST "https://api.formhub.io/api/v1/forms/FORM_ID/webhooks/endpoints/ENDPOINT_ID/test" \
  -H "Authorization: Bearer YOUR_API_KEY"

# Check webhook logs
curl "https://api.formhub.io/api/v1/forms/FORM_ID/webhooks/logs?limit=50" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Support

For additional support:
- **Documentation**: https://docs.formhub.io/webhooks
- **Community**: https://community.formhub.io
- **Support**: support@formhub.io
- **Status Page**: https://status.formhub.io

## Changelog

### Version 2.0.0 (Current)
- Enhanced webhook system with multiple endpoints
- Third-party integration marketplace
- Advanced analytics and monitoring
- Enterprise security features
- Circuit breaker and load balancing

### Version 1.0.0
- Basic webhook functionality
- Simple retry logic
- Basic analytics

---

*This documentation covers FormHub's Enhanced Webhook System API. For the latest updates and examples, visit our [developer portal](https://developers.formhub.io).*