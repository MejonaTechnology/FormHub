# FormHub Comprehensive Spam Protection System

## Overview

FormHub now includes a state-of-the-art spam protection system that rivals and exceeds the capabilities of Web3Forms and other leading form services. Our system provides multi-layered protection with machine learning, behavioral analysis, and intelligent CAPTCHA integration.

## 🚀 Features

### 1. Multiple CAPTCHA Support
- **Google reCAPTCHA v2** - Traditional checkbox and image challenges
- **Google reCAPTCHA v3** - Invisible background analysis with scoring
- **hCaptcha** - Privacy-focused alternative to reCAPTCHA
- **Cloudflare Turnstile** - Modern, user-friendly CAPTCHA
- **Fallback CAPTCHA** - Math-based questions for ultimate compatibility

### 2. Advanced Spam Detection
- **Honeypot Fields** - Hidden fields that detect automated bots
- **Rate Limiting** - IP and form-based submission limits
- **Content Analysis** - Keyword filtering and pattern detection
- **Behavioral Analysis** - User interaction pattern analysis
- **IP Reputation** - Real-time IP risk assessment
- **Domain Blacklisting** - Block submissions from known spam domains

### 3. Machine Learning Classification
- **Naive Bayes Classifier** - Learns from spam patterns
- **Feature Extraction** - Word, pattern, and metadata analysis
- **Automatic Learning** - Improves from user feedback and honeypots
- **Real-time Prediction** - Sub-second spam classification

### 4. Behavioral Analysis
- **Typing Patterns** - Analyzes typing speed and rhythm
- **Mouse Movement** - Tracks natural mouse interactions
- **Interaction Timing** - Detects bot-like immediate interactions
- **Form Completion** - Analyzes natural form-filling patterns
- **Device Fingerprinting** - Browser and device consistency checks

### 5. Configuration Management
- **Per-Form Settings** - Individual spam protection levels
- **Global Policies** - System-wide protection rules
- **Custom Rules** - Regex-based content filtering
- **Whitelist/Blacklist** - IP and domain management
- **Threshold Control** - Fine-tune protection sensitivity

### 6. Admin Interface
- **Real-time Statistics** - Spam detection metrics and trends
- **Quarantine Management** - Review flagged submissions
- **IP Management** - Monitor and manage IP reputations
- **Webhook Monitoring** - Track notification delivery
- **Data Export** - Comprehensive reporting and analytics

### 7. Webhook Notifications
- **Real-time Alerts** - Instant notifications for blocked spam
- **Multiple Providers** - Support for various webhook endpoints
- **Retry Logic** - Automatic retry with exponential backoff
- **Custom Payloads** - Flexible notification formatting
- **Signature Verification** - Secure webhook authentication

## 📦 Installation & Setup

### 1. Database Migration

Run the spam protection database migration:

```sql
-- Run the migration script
source migrations/003_spam_protection_tables.sql
```

### 2. Environment Configuration

Add the following environment variables:

```env
# CAPTCHA Configuration
RECAPTCHA_V2_SECRET=your_recaptcha_v2_secret
RECAPTCHA_V3_SECRET=your_recaptcha_v3_secret
RECAPTCHA_V3_MIN_SCORE=0.5
HCAPTCHA_SECRET=your_hcaptcha_secret
TURNSTILE_SECRET=your_turnstile_secret

# Spam Protection
SPAM_PROTECTION_ENABLED=true
DEFAULT_SPAM_LEVEL=medium
BLOCK_THRESHOLD=0.8
QUARANTINE_THRESHOLD=0.6

# Machine Learning
ML_ENABLED=true
ML_THRESHOLD=0.7
ENABLE_LEARNING=true

# Behavioral Analysis
BEHAVIORAL_ENABLED=true
MIN_TYPING_TIME=2.0
MAX_TYPING_SPEED=200.0

# Webhooks
WEBHOOK_TIMEOUT=30
MAX_WEBHOOK_RETRIES=3
```

### 3. Service Initialization

The spam protection services are automatically initialized in `main.go`:

```go
// Initialize spam protection services
securityService := services.NewSecurityService(db, redis, "./quarantine")
spamService := services.NewSpamProtectionService(db, redis, securityService)
mlClassifier := services.NewNaiveBayesSpamClassifier(db, redis)
behavioralAnalyzer := services.NewBehavioralAnalyzer(db, redis)
webhookService := services.NewWebhookService(db, redis)
```

## 🔧 Configuration

### Global Spam Protection Configuration

```json
{
  "enabled": true,
  "default_level": "medium",
  "block_threshold": 0.8,
  "quarantine_threshold": 0.6,
  "captcha_config": {
    "recaptcha_v2_secret": "your_secret",
    "recaptcha_v3_secret": "your_secret",
    "recaptcha_v3_min_score": 0.5,
    "hcaptcha_secret": "your_secret",
    "turnstile_secret": "your_secret",
    "fallback_enabled": true,
    "timeout": 10
  },
  "require_captcha": false,
  "captcha_provider": "recaptcha_v3",
  "captcha_fallback": true,
  "rate_limit_enabled": true,
  "max_submissions_per_ip": 10,
  "rate_limit_window_minutes": 15,
  "honeypot_enabled": true,
  "honeypot_fields": ["_honeypot", "_hp", "_bot_check"],
  "content_filter_enabled": true,
  "blocked_keywords": ["viagra", "casino", "lottery", "spam"],
  "blocked_domains": ["spam.com", "tempmail.org"],
  "max_url_count": 3,
  "behavioral_enabled": true,
  "min_typing_time_seconds": 2.0,
  "max_typing_speed_wpm": 200.0,
  "ip_reputation_enabled": true,
  "ml_enabled": true,
  "ml_threshold": 0.7,
  "enable_learning": true,
  "notify_on_block": true,
  "notify_on_quarantine": true
}
```

### Per-Form Configuration

```json
{
  "form_id": "contact-form",
  "enabled": true,
  "level": "high",
  "custom_config": {
    "block_threshold": 0.9,
    "require_captcha": true,
    "captcha_provider": "hcaptcha"
  },
  "whitelist": ["192.168.1.0/24"],
  "blacklist": ["10.0.0.1"],
  "custom_rules": [
    {
      "id": "phone-spam",
      "name": "Phone Spam Detection",
      "pattern": "\\b\\d{3}-\\d{3}-\\d{4}\\b.*urgent",
      "field": "message",
      "action": "block",
      "score": 0.8
    }
  ]
}
```

## 🔌 API Endpoints

### Public Endpoints

#### Submit Form with Spam Protection
```http
POST /api/v1/submit
Content-Type: application/json

{
  "form_id": "contact-form",
  "name": "John Doe",
  "email": "john@example.com",
  "message": "Hello, this is a test message.",
  "_honeypot": "",
  "captcha_token": "03AGdBq25...",
  "captcha_provider": "recaptcha_v3",
  "_behavioral_data": "{...}"
}
```

Response (Normal):
```json
{
  "success": true,
  "message": "Form submitted successfully",
  "submission_id": "sub_123456"
}
```

Response (Spam Detected):
```json
{
  "error": "Request blocked",
  "message": "Spam detected (score: 0.85)",
  "blocked_by": "spam_analysis",
  "timestamp": "2025-08-29T10:30:00Z",
  "request_id": "req_789012"
}
```

Response (CAPTCHA Required):
```json
{
  "error": "CAPTCHA required",
  "message": "Please complete the CAPTCHA challenge",
  "captcha_required": true,
  "captcha_providers": ["recaptcha_v3", "hcaptcha", "turnstile", "fallback"],
  "challenge_reason": "suspicious_behavior"
}
```

### Admin Endpoints

All admin endpoints require authentication and admin privileges.

#### Get Global Configuration
```http
GET /api/v1/admin/spam/config
Authorization: Bearer your_token
```

#### Update Global Configuration
```http
PUT /api/v1/admin/spam/config
Authorization: Bearer your_token
Content-Type: application/json

{
  "enabled": true,
  "default_level": "high",
  "block_threshold": 0.85
}
```

#### Get Form-Specific Configuration
```http
GET /api/v1/admin/spam/forms/{formId}/config
Authorization: Bearer your_token
```

#### Get Spam Statistics
```http
GET /api/v1/admin/spam/statistics?form_id=contact-form&days=30
Authorization: Bearer your_token
```

#### Get Quarantined Submissions
```http
GET /api/v1/admin/spam/quarantined?status=pending&limit=50
Authorization: Bearer your_token
```

#### Review Quarantined Submission
```http
PUT /api/v1/admin/spam/quarantined/{submissionId}
Authorization: Bearer your_token
Content-Type: application/json

{
  "action": "approve",
  "notes": "False positive - legitimate submission"
}
```

#### Get Machine Learning Stats
```http
GET /api/v1/admin/spam/ml/stats
Authorization: Bearer your_token
```

#### Train ML Model
```http
POST /api/v1/admin/spam/ml/train
Authorization: Bearer your_token
Content-Type: application/json

{
  "use_latest_data": true,
  "min_samples": 100
}
```

#### Test Webhook
```http
POST /api/v1/admin/spam/webhooks/test
Authorization: Bearer your_token
Content-Type: application/json

{
  "url": "https://your-webhook-endpoint.com",
  "secret": "your_webhook_secret",
  "timeout": 10
}
```

#### Export Spam Data
```http
GET /api/v1/admin/spam/export?format=json&days=30&form_id=contact-form
Authorization: Bearer your_token
```

## 🎯 Integration Examples

### Frontend JavaScript Integration

```javascript
// Basic form submission with spam protection
async function submitForm(formData) {
  // Add honeypot field
  formData._honeypot = '';
  
  // Add behavioral data
  formData._behavioral_data = JSON.stringify(collectBehavioralData());
  
  // Add CAPTCHA token if available
  if (window.grecaptcha) {
    formData.captcha_token = await grecaptcha.execute('your_site_key');
    formData.captcha_provider = 'recaptcha_v3';
  }
  
  try {
    const response = await fetch('/api/v1/submit', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(formData)
    });
    
    const result = await response.json();
    
    if (response.ok) {
      showSuccessMessage('Form submitted successfully!');
    } else if (result.captcha_required) {
      // Show CAPTCHA challenge
      showCaptchaChallenge(result.captcha_providers);
    } else {
      showErrorMessage(result.message);
    }
  } catch (error) {
    showErrorMessage('Submission failed. Please try again.');
  }
}

// Collect behavioral data for analysis
function collectBehavioralData() {
  return {
    typing_time: getTypingTime(),
    typing_speed: calculateTypingSpeed(),
    mouse_movements: mouseMovements.length,
    mouse_distance: calculateMouseDistance(),
    scroll_events: scrollEvents.length,
    click_events: clickEvents.length,
    focus_events: focusEvents.length,
    tab_switches: tabSwitchCount,
    copy_paste_events: copyPasteCount,
    backspace_ratio: backspaceCount / keystrokeCount,
    typing_rhythm: getKeystrokeIntervals(),
    time_on_page: (Date.now() - pageLoadTime) / 1000,
    interaction_delay: (firstInteractionTime - pageLoadTime) / 1000
  };
}
```

### React Component Example

```jsx
import React, { useState, useEffect } from 'react';
import { useReCaptcha } from 'react-google-recaptcha-v3';

const SpamProtectedForm = () => {
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    message: ''
  });
  const [behavioralData, setBehavioralData] = useState({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { executeRecaptcha } = useReCaptcha();
  
  // Track behavioral data
  useEffect(() => {
    const tracker = new BehavioralTracker();
    tracker.start();
    
    return () => {
      setBehavioralData(tracker.getData());
      tracker.stop();
    };
  }, []);
  
  const handleSubmit = async (e) => {
    e.preventDefault();
    setIsSubmitting(true);
    
    try {
      // Get CAPTCHA token
      const captchaToken = await executeRecaptcha('submit');
      
      const submissionData = {
        ...formData,
        _honeypot: '', // Honeypot field
        captcha_token: captchaToken,
        captcha_provider: 'recaptcha_v3',
        _behavioral_data: JSON.stringify(behavioralData)
      };
      
      const response = await fetch('/api/v1/submit', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(submissionData)
      });
      
      const result = await response.json();
      
      if (response.ok) {
        alert('Form submitted successfully!');
        setFormData({ name: '', email: '', message: '' });
      } else if (result.captcha_required) {
        alert('Please complete the CAPTCHA challenge');
      } else {
        alert(result.message || 'Submission failed');
      }
    } catch (error) {
      alert('Network error. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  };
  
  return (
    <form onSubmit={handleSubmit}>
      <input
        type="text"
        placeholder="Name"
        value={formData.name}
        onChange={(e) => setFormData({...formData, name: e.target.value})}
        required
      />
      
      <input
        type="email"
        placeholder="Email"
        value={formData.email}
        onChange={(e) => setFormData({...formData, email: e.target.value})}
        required
      />
      
      <textarea
        placeholder="Message"
        value={formData.message}
        onChange={(e) => setFormData({...formData, message: e.target.value})}
        required
      />
      
      {/* Honeypot field - hidden with CSS */}
      <input
        type="text"
        name="_honeypot"
        style={{ display: 'none' }}
        tabIndex="-1"
        autoComplete="off"
      />
      
      <button type="submit" disabled={isSubmitting}>
        {isSubmitting ? 'Submitting...' : 'Submit'}
      </button>
    </form>
  );
};
```

### Webhook Handler Example

```javascript
// Express.js webhook handler
const crypto = require('crypto');

app.post('/webhook/formhub-spam', (req, res) => {
  // Verify webhook signature
  const signature = req.headers['x-formhub-signature'];
  const timestamp = req.headers['x-formhub-timestamp'];
  
  if (!verifySignature(req.body, signature, process.env.WEBHOOK_SECRET)) {
    return res.status(401).json({ error: 'Invalid signature' });
  }
  
  const event = req.body;
  
  switch (event.type) {
    case 'spam.blocked':
      handleSpamBlocked(event);
      break;
    case 'spam.quarantined':
      handleSpamQuarantined(event);
      break;
    case 'security.ip_flagged':
      handleIPFlagged(event);
      break;
  }
  
  res.json({ received: true });
});

function verifySignature(payload, signature, secret) {
  const expectedSignature = crypto
    .createHmac('sha256', secret)
    .update(JSON.stringify(payload))
    .digest('hex');
  
  return signature === `sha256=${expectedSignature}`;
}
```

## 📊 Monitoring & Analytics

### Key Metrics to Monitor

1. **Spam Detection Rate** - Percentage of submissions flagged as spam
2. **False Positive Rate** - Legitimate submissions incorrectly blocked
3. **CAPTCHA Success Rate** - Percentage of successful CAPTCHA completions
4. **Processing Time** - Average time for spam analysis
5. **IP Reputation Distribution** - Good vs. suspicious vs. malicious IPs
6. **ML Model Accuracy** - Precision, recall, and F1 scores
7. **Webhook Delivery Rate** - Successful webhook notifications

### Dashboard Queries

```sql
-- Daily spam statistics
SELECT 
  DATE(created_at) as date,
  COUNT(*) as total_submissions,
  SUM(CASE WHEN is_spam = 1 THEN 1 ELSE 0 END) as spam_detected,
  AVG(spam_score) as avg_spam_score,
  AVG(processing_time_ms) as avg_processing_time
FROM spam_analysis_logs 
WHERE created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)
GROUP BY DATE(created_at)
ORDER BY date DESC;

-- Top spam triggers
SELECT 
  JSON_UNQUOTE(JSON_EXTRACT(triggers, '$[*].type')) as trigger_type,
  COUNT(*) as trigger_count,
  AVG(spam_score) as avg_spam_score
FROM spam_analysis_logs 
WHERE is_spam = 1 
  AND created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
GROUP BY trigger_type
ORDER BY trigger_count DESC;

-- IP reputation distribution
SELECT 
  reputation,
  COUNT(*) as count,
  AVG(risk_score) as avg_risk_score
FROM ip_reputation 
GROUP BY reputation;
```

## 🛡️ Security Considerations

### Data Privacy
- All behavioral data is anonymized and encrypted
- IP addresses are hashed for storage
- Form data in quarantine is automatically expired
- GDPR compliance features available

### Performance Optimization
- Redis caching for IP reputation and ML models
- Asynchronous webhook processing
- Background ML model training
- Optimized database queries with proper indexing

### High Availability
- Graceful degradation when external services fail
- Fallback CAPTCHA when primary providers are down
- Circuit breakers for external API calls
- Health checks and monitoring

## 🧪 Testing

Run the comprehensive test suite:

```bash
# Install Python dependencies
pip install requests

# Run all tests
python test/spam_protection_test.py --url http://localhost:8080 --token your_auth_token --verbose

# Run specific test suites
python test/spam_protection_test.py --url http://localhost:8080 --suite captcha
python test/spam_protection_test.py --url http://localhost:8080 --suite behavioral

# Export test results
python test/spam_protection_test.py --output test_results.json
```

### Test Coverage
- ✅ CAPTCHA provider integration
- ✅ Honeypot field detection
- ✅ Rate limiting enforcement
- ✅ Content analysis and filtering
- ✅ Behavioral pattern analysis
- ✅ Machine learning classification
- ✅ IP reputation checking
- ✅ Webhook notifications
- ✅ Admin interface functionality
- ✅ Performance and load testing

## 📚 Best Practices

### For Form Owners
1. **Start with medium protection level** and adjust based on spam volume
2. **Monitor false positives** and whitelist legitimate users/IPs
3. **Use behavioral analysis** for better bot detection
4. **Enable webhooks** for real-time spam notifications
5. **Review quarantined submissions** regularly
6. **Update custom rules** based on spam patterns

### For Developers
1. **Implement honeypot fields** in all forms
2. **Collect behavioral data** for improved detection
3. **Handle CAPTCHA challenges** gracefully
4. **Implement proper error handling** for blocked submissions
5. **Use HTTPS** for all form submissions
6. **Validate webhook signatures** for security

### For System Administrators
1. **Monitor system performance** and resource usage
2. **Keep ML models updated** with fresh training data
3. **Review IP reputation** and update blacklists
4. **Set up proper alerting** for high spam volumes
5. **Regular backup** of configuration and models
6. **Monitor webhook delivery** and retry failures

## 🔄 Maintenance Tasks

### Daily
- Review quarantined submissions
- Monitor spam detection rates
- Check webhook delivery status

### Weekly
- Analyze spam trends and patterns
- Update custom rules if needed
- Review IP reputation changes
- Check ML model performance

### Monthly
- Retrain ML models with fresh data
- Update global configuration based on trends
- Audit admin access logs
- Performance optimization review

### Quarterly
- Comprehensive security review
- Update CAPTCHA provider configurations
- Review and update documentation
- Conduct penetration testing

## 🆘 Troubleshooting

### Common Issues

#### High False Positive Rate
- **Check behavioral analysis thresholds** - May be too strict
- **Review custom rules** - Overly broad patterns
- **Examine content filters** - Blocking legitimate keywords
- **Lower spam score thresholds** - Reduce quarantine/block limits

#### Performance Issues
- **Check database indexes** - Ensure proper indexing on spam tables
- **Monitor Redis usage** - May need more memory
- **Review ML model size** - Large models slow processing
- **Optimize custom rules** - Complex regex patterns are slow

#### CAPTCHA Not Working
- **Verify API keys** - Check reCAPTCHA/hCaptcha credentials
- **Check domain configuration** - Ensure domains are whitelisted
- **Network connectivity** - Firewall may block CAPTCHA requests
- **Enable fallback CAPTCHA** - Provides backup when services fail

#### Webhooks Failing
- **Check webhook URL** - Ensure endpoint is accessible
- **Verify SSL certificates** - HTTPS endpoints need valid certs
- **Review rate limits** - Webhook endpoint may be rate limited
- **Check signature verification** - Incorrect secret configuration

### Debug Mode

Enable debug logging for detailed spam analysis:

```go
// Add to main.go
if cfg.Environment == "development" {
    gin.SetMode(gin.DebugMode)
    
    // Enable detailed spam logging
    spamMiddleware.EnableDebugLogging()
}
```

### Performance Monitoring

Monitor key performance metrics:

```sql
-- Slow spam analysis queries
SELECT 
  form_id,
  AVG(processing_time_ms) as avg_time,
  COUNT(*) as count
FROM spam_analysis_logs 
WHERE processing_time_ms > 1000  -- Over 1 second
  AND created_at >= DATE_SUB(NOW(), INTERVAL 1 DAY)
GROUP BY form_id;
```

## 📈 Roadmap

### Planned Features
- **AI-powered image CAPTCHA** - Custom image recognition challenges  
- **Blockchain verification** - Decentralized identity verification
- **Advanced behavioral biometrics** - Keystroke dynamics analysis
- **Integration with threat intelligence** - Real-time IP/domain feeds
- **A/B testing framework** - Optimize protection without impacting UX
- **Mobile-specific detection** - Touch pattern analysis for mobile forms

### Performance Improvements
- **Distributed ML inference** - Scale ML processing across nodes
- **Edge computing integration** - Process at CDN level for speed
- **GraphQL admin interface** - More efficient admin queries
- **Real-time analytics** - WebSocket-based live monitoring
- **Automated threshold tuning** - AI-powered configuration optimization

---

## 📞 Support

For technical support or feature requests:
- Email: support@formhub.io
- Documentation: https://docs.formhub.io/spam-protection
- GitHub Issues: https://github.com/formhub/formhub/issues
- Discord Community: https://discord.gg/formhub

---

*FormHub Spam Protection System - Securing forms, one submission at a time.* 🛡️