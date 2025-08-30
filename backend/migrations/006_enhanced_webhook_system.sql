-- Enhanced Webhook System Migration
-- This migration adds comprehensive webhook support with enterprise features

-- Webhook logs for detailed tracking
CREATE TABLE IF NOT EXISTS webhook_logs (
    id VARCHAR(36) PRIMARY KEY,
    endpoint_id VARCHAR(36) NOT NULL,
    form_id VARCHAR(36) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    url TEXT NOT NULL,
    status_code INT,
    response_time_ms BIGINT,
    attempts INT DEFAULT 1,
    success BOOLEAN DEFAULT FALSE,
    error_message TEXT,
    request_payload JSON,
    response_body TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_webhook_logs_form_id (form_id),
    INDEX idx_webhook_logs_endpoint_id (endpoint_id),
    INDEX idx_webhook_logs_created_at (created_at),
    INDEX idx_webhook_logs_success (success),
    INDEX idx_webhook_logs_event_type (event_type)
);

-- Webhook health checks for monitoring
CREATE TABLE IF NOT EXISTS webhook_health_checks (
    id VARCHAR(36) PRIMARY KEY,
    form_id VARCHAR(36) NOT NULL,
    endpoint_id VARCHAR(36) NOT NULL,
    status ENUM('healthy', 'unhealthy', 'unknown') DEFAULT 'unknown',
    response_time_ms BIGINT,
    success_rate DECIMAL(5,2),
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_endpoint_health (form_id, endpoint_id),
    INDEX idx_health_checks_form_id (form_id),
    INDEX idx_health_checks_status (status),
    INDEX idx_health_checks_updated_at (updated_at)
);

-- Webhook alerts for monitoring and notifications
CREATE TABLE IF NOT EXISTS webhook_alerts (
    id VARCHAR(36) PRIMARY KEY,
    form_id VARCHAR(36) NOT NULL,
    endpoint_id VARCHAR(36),
    type VARCHAR(50) NOT NULL, -- high_failure_rate, endpoint_down, etc.
    severity ENUM('info', 'warning', 'critical') DEFAULT 'info',
    message TEXT NOT NULL,
    data JSON, -- Additional alert data
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_at TIMESTAMP NULL,
    acknowledged_by VARCHAR(36),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_webhook_alerts_form_id (form_id),
    INDEX idx_webhook_alerts_type (type),
    INDEX idx_webhook_alerts_severity (severity),
    INDEX idx_webhook_alerts_acknowledged (acknowledged),
    INDEX idx_webhook_alerts_created_at (created_at)
);

-- Webhook monitor events for activity tracking
CREATE TABLE IF NOT EXISTS webhook_monitor_events (
    id VARCHAR(36) PRIMARY KEY,
    form_id VARCHAR(36) NOT NULL,
    endpoint_id VARCHAR(36),
    type VARCHAR(50) NOT NULL, -- status_change, test, etc.
    severity ENUM('info', 'warning', 'critical') DEFAULT 'info',
    message TEXT NOT NULL,
    data JSON, -- Event-specific data
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_monitor_events_form_id (form_id),
    INDEX idx_monitor_events_type (type),
    INDEX idx_monitor_events_severity (severity),
    INDEX idx_monitor_events_created_at (created_at)
);

-- OAuth tokens for third-party integrations
CREATE TABLE IF NOT EXISTS webhook_oauth_tokens (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    service_name VARCHAR(50) NOT NULL, -- google, slack, etc.
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_type VARCHAR(20) DEFAULT 'Bearer',
    expires_at TIMESTAMP,
    scope TEXT, -- Comma-separated scopes
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_user_service (user_id, service_name),
    INDEX idx_oauth_tokens_user_id (user_id),
    INDEX idx_oauth_tokens_service (service_name),
    INDEX idx_oauth_tokens_expires_at (expires_at)
);

-- Integration configurations for forms
CREATE TABLE IF NOT EXISTS form_integrations (
    id VARCHAR(36) PRIMARY KEY,
    form_id VARCHAR(36) NOT NULL,
    integration_type VARCHAR(50) NOT NULL, -- google_sheets, slack, etc.
    integration_name VARCHAR(100) NOT NULL,
    configuration JSON NOT NULL, -- Integration-specific config
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_form_integrations_form_id (form_id),
    INDEX idx_form_integrations_type (integration_type),
    INDEX idx_form_integrations_enabled (enabled),
    FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE
);

-- Webhook templates for reusable configurations
CREATE TABLE IF NOT EXISTS webhook_templates (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    category VARCHAR(50), -- notification, integration, custom
    template_data JSON NOT NULL, -- Template configuration
    is_public BOOLEAN DEFAULT FALSE,
    usage_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_webhook_templates_user_id (user_id),
    INDEX idx_webhook_templates_category (category),
    INDEX idx_webhook_templates_public (is_public),
    INDEX idx_webhook_templates_usage (usage_count)
);

-- Webhook rate limiting tracking
CREATE TABLE IF NOT EXISTS webhook_rate_limits (
    id VARCHAR(36) PRIMARY KEY,
    endpoint_url VARCHAR(500) NOT NULL,
    ip_address VARCHAR(45),
    user_id VARCHAR(36),
    form_id VARCHAR(36),
    request_count INT DEFAULT 1,
    window_start TIMESTAMP NOT NULL,
    window_end TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_rate_limits_endpoint (endpoint_url(100)),
    INDEX idx_rate_limits_ip (ip_address),
    INDEX idx_rate_limits_user (user_id),
    INDEX idx_rate_limits_window (window_start, window_end)
);

-- Webhook security settings
CREATE TABLE IF NOT EXISTS webhook_security_settings (
    id VARCHAR(36) PRIMARY KEY,
    form_id VARCHAR(36) NOT NULL,
    allowed_ips JSON, -- Array of allowed IP addresses
    blocked_ips JSON, -- Array of blocked IP addresses
    allowed_domains JSON, -- Array of allowed domains
    blocked_domains JSON, -- Array of blocked domains
    require_signature BOOLEAN DEFAULT TRUE,
    signature_algorithm VARCHAR(20) DEFAULT 'sha256',
    max_payload_size INT DEFAULT 5242880, -- 5MB
    rate_limit_enabled BOOLEAN DEFAULT TRUE,
    rate_limit_requests INT DEFAULT 100,
    rate_limit_window INT DEFAULT 3600, -- 1 hour in seconds
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_form_security (form_id),
    FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE
);

-- Webhook analytics aggregated data
CREATE TABLE IF NOT EXISTS webhook_analytics (
    id VARCHAR(36) PRIMARY KEY,
    form_id VARCHAR(36) NOT NULL,
    endpoint_id VARCHAR(36),
    date DATE NOT NULL,
    hour TINYINT, -- NULL for daily aggregates, 0-23 for hourly
    total_requests INT DEFAULT 0,
    successful_requests INT DEFAULT 0,
    failed_requests INT DEFAULT 0,
    avg_response_time_ms BIGINT DEFAULT 0,
    min_response_time_ms BIGINT DEFAULT 0,
    max_response_time_ms BIGINT DEFAULT 0,
    p95_response_time_ms BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_analytics (form_id, endpoint_id, date, hour),
    INDEX idx_analytics_form_date (form_id, date),
    INDEX idx_analytics_endpoint_date (endpoint_id, date),
    INDEX idx_analytics_date (date)
);

-- Integration marketplace (for pre-built integrations)
CREATE TABLE IF NOT EXISTS integration_marketplace (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL,
    version VARCHAR(20) NOT NULL,
    author VARCHAR(100),
    icon_url TEXT,
    tags JSON, -- Array of tags
    configuration_schema JSON NOT NULL, -- JSON schema for configuration
    template JSON NOT NULL, -- Integration template
    is_popular BOOLEAN DEFAULT FALSE,
    is_featured BOOLEAN DEFAULT FALSE,
    downloads INT DEFAULT 0,
    rating DECIMAL(3,2) DEFAULT 0.0,
    rating_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_marketplace_category (category),
    INDEX idx_marketplace_popular (is_popular),
    INDEX idx_marketplace_featured (is_featured),
    INDEX idx_marketplace_rating (rating),
    INDEX idx_marketplace_downloads (downloads)
);

-- Integration reviews and ratings
CREATE TABLE IF NOT EXISTS integration_reviews (
    id VARCHAR(36) PRIMARY KEY,
    integration_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    rating TINYINT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    review_text TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_user_review (integration_id, user_id),
    INDEX idx_reviews_integration (integration_id),
    INDEX idx_reviews_user (user_id),
    INDEX idx_reviews_rating (rating),
    FOREIGN KEY (integration_id) REFERENCES integration_marketplace(id) ON DELETE CASCADE
);

-- Webhook retry queue
CREATE TABLE IF NOT EXISTS webhook_retry_queue (
    id VARCHAR(36) PRIMARY KEY,
    webhook_log_id VARCHAR(36) NOT NULL,
    endpoint_id VARCHAR(36) NOT NULL,
    form_id VARCHAR(36) NOT NULL,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    next_retry_at TIMESTAMP NOT NULL,
    payload JSON NOT NULL,
    status ENUM('pending', 'processing', 'completed', 'failed') DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_retry_queue_next_retry (next_retry_at),
    INDEX idx_retry_queue_status (status),
    INDEX idx_retry_queue_form_id (form_id),
    FOREIGN KEY (webhook_log_id) REFERENCES webhook_logs(id) ON DELETE CASCADE
);

-- Circuit breaker states
CREATE TABLE IF NOT EXISTS webhook_circuit_breakers (
    id VARCHAR(36) PRIMARY KEY,
    endpoint_id VARCHAR(36) NOT NULL,
    form_id VARCHAR(36) NOT NULL,
    state ENUM('closed', 'open', 'half_open') DEFAULT 'closed',
    failure_count INT DEFAULT 0,
    failure_threshold INT DEFAULT 5,
    last_failure_at TIMESTAMP NULL,
    next_reset_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_endpoint_breaker (endpoint_id),
    INDEX idx_circuit_breakers_form_id (form_id),
    INDEX idx_circuit_breakers_state (state),
    INDEX idx_circuit_breakers_next_reset (next_reset_at)
);

-- Insert default marketplace integrations
INSERT INTO integration_marketplace (id, name, description, category, version, author, tags, configuration_schema, template, is_popular, is_featured, downloads, rating) VALUES
(
    'google-sheets',
    'Google Sheets',
    'Send form submissions directly to Google Sheets spreadsheets',
    'productivity',
    '1.0.0',
    'FormHub',
    '["sheets", "google", "productivity", "data"]',
    '{
        "type": "object",
        "properties": {
            "spreadsheet_id": {"type": "string", "description": "Google Sheets spreadsheet ID"},
            "worksheet_name": {"type": "string", "description": "Worksheet name (optional)"},
            "credentials_json": {"type": "string", "description": "Service account credentials JSON"},
            "field_mappings": {"type": "object", "description": "Field mappings"}
        },
        "required": ["spreadsheet_id", "credentials_json"]
    }',
    '{"type": "google_sheets", "config": {}}',
    TRUE,
    TRUE,
    15420,
    4.8
),
(
    'slack',
    'Slack',
    'Get notified in Slack channels when forms are submitted',
    'communication',
    '2.1.0',
    'FormHub',
    '["slack", "notification", "communication", "team"]',
    '{
        "type": "object",
        "properties": {
            "webhook_url": {"type": "string", "description": "Slack webhook URL"},
            "channel": {"type": "string", "description": "Slack channel"},
            "username": {"type": "string", "description": "Bot username"},
            "icon_emoji": {"type": "string", "description": "Bot emoji icon"}
        },
        "required": ["webhook_url", "channel"]
    }',
    '{"type": "slack", "config": {}}',
    TRUE,
    TRUE,
    12847,
    4.7
),
(
    'airtable',
    'Airtable',
    'Organize form submissions in Airtable bases',
    'database',
    '1.5.0',
    'FormHub',
    '["airtable", "database", "organization", "crm"]',
    '{
        "type": "object",
        "properties": {
            "api_key": {"type": "string", "description": "Airtable API key"},
            "base_id": {"type": "string", "description": "Airtable base ID"},
            "table_name": {"type": "string", "description": "Table name"},
            "field_mappings": {"type": "object", "description": "Field mappings"}
        },
        "required": ["api_key", "base_id", "table_name"]
    }',
    '{"type": "airtable", "config": {}}',
    TRUE,
    FALSE,
    8932,
    4.6
),
(
    'zapier',
    'Zapier',
    'Connect to thousands of apps through Zapier webhooks',
    'automation',
    '2.0.0',
    'FormHub',
    '["zapier", "automation", "integration", "workflow"]',
    '{
        "type": "object",
        "properties": {
            "webhook_url": {"type": "string", "description": "Zapier webhook URL"},
            "custom_fields": {"type": "object", "description": "Custom field mappings"}
        },
        "required": ["webhook_url"]
    }',
    '{"type": "zapier", "config": {}}',
    TRUE,
    TRUE,
    18932,
    4.9
);

-- Create indexes for better performance
CREATE INDEX idx_forms_webhook_config ON forms(webhook_config(100));

-- Add webhook configuration column to forms table if not exists
ALTER TABLE forms ADD COLUMN IF NOT EXISTS webhook_config JSON AFTER description;

-- Update existing webhook notifications table structure
ALTER TABLE webhook_notifications ADD COLUMN IF NOT EXISTS archived BOOLEAN DEFAULT FALSE;
ALTER TABLE webhook_notifications ADD COLUMN IF NOT EXISTS archived_at TIMESTAMP NULL;
ALTER TABLE webhook_notifications ADD COLUMN IF NOT EXISTS webhook_endpoint_id VARCHAR(36);

-- Add indexes to existing webhook_notifications table
CREATE INDEX IF NOT EXISTS idx_webhook_notifications_archived ON webhook_notifications(archived);
CREATE INDEX IF NOT EXISTS idx_webhook_notifications_endpoint ON webhook_notifications(webhook_endpoint_id);

-- Create trigger to update marketplace integration ratings
DELIMITER //
CREATE TRIGGER IF NOT EXISTS update_integration_rating 
AFTER INSERT ON integration_reviews
FOR EACH ROW
BEGIN
    UPDATE integration_marketplace 
    SET rating = (
        SELECT AVG(rating) 
        FROM integration_reviews 
        WHERE integration_id = NEW.integration_id
    ),
    rating_count = (
        SELECT COUNT(*) 
        FROM integration_reviews 
        WHERE integration_id = NEW.integration_id
    )
    WHERE id = NEW.integration_id;
END //
DELIMITER ;

-- Create trigger to clean up old webhook logs (keep only last 90 days)
DELIMITER //
CREATE EVENT IF NOT EXISTS cleanup_old_webhook_logs
ON SCHEDULE EVERY 1 DAY
STARTS CURRENT_TIMESTAMP
DO
BEGIN
    DELETE FROM webhook_logs 
    WHERE created_at < DATE_SUB(NOW(), INTERVAL 90 DAY) 
    AND archived = TRUE
    LIMIT 10000;
    
    DELETE FROM webhook_monitor_events 
    WHERE created_at < DATE_SUB(NOW(), INTERVAL 30 DAY)
    LIMIT 5000;
    
    DELETE FROM webhook_rate_limits 
    WHERE window_end < DATE_SUB(NOW(), INTERVAL 7 DAY)
    LIMIT 5000;
END //
DELIMITER ;

-- Create stored procedure for webhook analytics aggregation
DELIMITER //
CREATE PROCEDURE IF NOT EXISTS AggregateWebhookAnalytics(IN target_date DATE)
BEGIN
    -- Daily aggregation
    INSERT INTO webhook_analytics (
        id, form_id, endpoint_id, date, hour, 
        total_requests, successful_requests, failed_requests, 
        avg_response_time_ms, min_response_time_ms, max_response_time_ms
    )
    SELECT 
        UUID() as id,
        form_id,
        endpoint_id,
        DATE(created_at) as date,
        NULL as hour,
        COUNT(*) as total_requests,
        SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as successful_requests,
        SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) as failed_requests,
        AVG(response_time_ms) as avg_response_time_ms,
        MIN(response_time_ms) as min_response_time_ms,
        MAX(response_time_ms) as max_response_time_ms
    FROM webhook_logs
    WHERE DATE(created_at) = target_date
    GROUP BY form_id, endpoint_id, DATE(created_at)
    ON DUPLICATE KEY UPDATE
        total_requests = VALUES(total_requests),
        successful_requests = VALUES(successful_requests),
        failed_requests = VALUES(failed_requests),
        avg_response_time_ms = VALUES(avg_response_time_ms),
        min_response_time_ms = VALUES(min_response_time_ms),
        max_response_time_ms = VALUES(max_response_time_ms);
    
    -- Hourly aggregation
    INSERT INTO webhook_analytics (
        id, form_id, endpoint_id, date, hour,
        total_requests, successful_requests, failed_requests,
        avg_response_time_ms, min_response_time_ms, max_response_time_ms
    )
    SELECT 
        UUID() as id,
        form_id,
        endpoint_id,
        DATE(created_at) as date,
        HOUR(created_at) as hour,
        COUNT(*) as total_requests,
        SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as successful_requests,
        SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) as failed_requests,
        AVG(response_time_ms) as avg_response_time_ms,
        MIN(response_time_ms) as min_response_time_ms,
        MAX(response_time_ms) as max_response_time_ms
    FROM webhook_logs
    WHERE DATE(created_at) = target_date
    GROUP BY form_id, endpoint_id, DATE(created_at), HOUR(created_at)
    ON DUPLICATE KEY UPDATE
        total_requests = VALUES(total_requests),
        successful_requests = VALUES(successful_requests),
        failed_requests = VALUES(failed_requests),
        avg_response_time_ms = VALUES(avg_response_time_ms),
        min_response_time_ms = VALUES(min_response_time_ms),
        max_response_time_ms = VALUES(max_response_time_ms);
END //
DELIMITER ;

-- Create event to run daily analytics aggregation
DELIMITER //
CREATE EVENT IF NOT EXISTS daily_webhook_analytics_aggregation
ON SCHEDULE EVERY 1 DAY
STARTS CURRENT_TIMESTAMP
DO
BEGIN
    CALL AggregateWebhookAnalytics(DATE_SUB(CURDATE(), INTERVAL 1 DAY));
END //
DELIMITER ;

-- Enable event scheduler
SET GLOBAL event_scheduler = ON;