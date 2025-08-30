-- Form Analytics System Migration
-- This migration adds comprehensive form analytics, submission tracking, and business intelligence support

-- Form Analytics Events Table
CREATE TABLE form_analytics_events (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    form_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    session_id VARCHAR(255) NOT NULL,
    event_type ENUM(
        'form_view', 'form_start', 'field_focus', 'field_blur', 'field_change',
        'form_submit', 'form_complete', 'form_abandon', 'validation_error',
        'file_upload_start', 'file_upload_complete', 'recaptcha_solve'
    ) NOT NULL,
    field_name VARCHAR(255) NULL,
    field_value_length INT NULL,
    field_validation_error TEXT NULL,
    page_url TEXT NOT NULL,
    referrer TEXT NULL,
    utm_source VARCHAR(255) NULL,
    utm_medium VARCHAR(255) NULL,
    utm_campaign VARCHAR(255) NULL,
    utm_term VARCHAR(255) NULL,
    utm_content VARCHAR(255) NULL,
    device_type ENUM('desktop', 'tablet', 'mobile', 'unknown') NOT NULL DEFAULT 'unknown',
    browser_name VARCHAR(100) NULL,
    browser_version VARCHAR(50) NULL,
    os_name VARCHAR(100) NULL,
    os_version VARCHAR(50) NULL,
    screen_resolution VARCHAR(20) NULL,
    viewport_size VARCHAR(20) NULL,
    ip_address VARCHAR(45) NOT NULL,
    country_code CHAR(2) NULL,
    country_name VARCHAR(100) NULL,
    region VARCHAR(100) NULL,
    city VARCHAR(100) NULL,
    latitude DECIMAL(10, 8) NULL,
    longitude DECIMAL(11, 8) NULL,
    timezone VARCHAR(100) NULL,
    user_agent TEXT NULL,
    event_data JSON NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_form_analytics_events_form_id (form_id),
    INDEX idx_form_analytics_events_user_id (user_id),
    INDEX idx_form_analytics_events_session_id (session_id),
    INDEX idx_form_analytics_events_type (event_type),
    INDEX idx_form_analytics_events_created_at (created_at),
    INDEX idx_form_analytics_events_utm_source (utm_source),
    INDEX idx_form_analytics_events_device_type (device_type),
    INDEX idx_form_analytics_events_country (country_code),
    INDEX idx_form_analytics_events_field (field_name)
);

-- Form Conversion Funnels Table
CREATE TABLE form_conversion_funnels (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    form_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    date DATE NOT NULL,
    total_views INT DEFAULT 0,
    total_starts INT DEFAULT 0,
    total_submits INT DEFAULT 0,
    total_completes INT DEFAULT 0,
    total_abandons INT DEFAULT 0,
    abandonment_points JSON NULL, -- {"field_name": abandon_count}
    conversion_rate DECIMAL(5, 2) DEFAULT 0.00,
    completion_rate DECIMAL(5, 2) DEFAULT 0.00,
    average_time_to_submit INT NULL, -- in seconds
    average_time_to_abandon INT NULL, -- in seconds
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY unique_funnel_date (form_id, date),
    INDEX idx_form_conversion_funnels_form_id (form_id),
    INDEX idx_form_conversion_funnels_user_id (user_id),
    INDEX idx_form_conversion_funnels_date (date),
    INDEX idx_form_conversion_funnels_conversion_rate (conversion_rate)
);

-- Submission Lifecycle Tracking Table
CREATE TABLE submission_lifecycle (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    submission_id CHAR(36) NOT NULL UNIQUE,
    form_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    tracking_id VARCHAR(255) NOT NULL UNIQUE,
    status ENUM(
        'received', 'processing', 'validated', 'spam_flagged', 
        'email_sent', 'webhook_sent', 'completed', 'failed', 
        'responded', 'archived'
    ) NOT NULL DEFAULT 'received',
    processing_time_ms INT NULL,
    validation_errors JSON NULL,
    spam_detection_score DECIMAL(3, 2) NULL,
    spam_detection_reasons JSON NULL,
    email_delivery_status ENUM('pending', 'sent', 'delivered', 'bounced', 'failed') NULL,
    email_delivery_time_ms INT NULL,
    webhook_delivery_status ENUM('pending', 'sent', 'success', 'failed', 'retry') NULL,
    webhook_delivery_time_ms INT NULL,
    webhook_response_code INT NULL,
    response_time TIMESTAMP NULL,
    response_method ENUM('email', 'phone', 'in_person', 'other') NULL,
    notes TEXT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (submission_id) REFERENCES submissions(id) ON DELETE CASCADE,
    FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_submission_lifecycle_submission_id (submission_id),
    INDEX idx_submission_lifecycle_form_id (form_id),
    INDEX idx_submission_lifecycle_user_id (user_id),
    INDEX idx_submission_lifecycle_tracking_id (tracking_id),
    INDEX idx_submission_lifecycle_status (status),
    INDEX idx_submission_lifecycle_created_at (created_at)
);

-- User Sessions Table for Journey Tracking
CREATE TABLE user_sessions (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    session_id VARCHAR(255) NOT NULL UNIQUE,
    user_id CHAR(36) NULL,
    ip_address VARCHAR(45) NOT NULL,
    user_agent TEXT NULL,
    device_type ENUM('desktop', 'tablet', 'mobile', 'unknown') NOT NULL DEFAULT 'unknown',
    browser_name VARCHAR(100) NULL,
    browser_version VARCHAR(50) NULL,
    os_name VARCHAR(100) NULL,
    os_version VARCHAR(50) NULL,
    country_code CHAR(2) NULL,
    country_name VARCHAR(100) NULL,
    region VARCHAR(100) NULL,
    city VARCHAR(100) NULL,
    timezone VARCHAR(100) NULL,
    referrer TEXT NULL,
    landing_page TEXT NULL,
    utm_source VARCHAR(255) NULL,
    utm_medium VARCHAR(255) NULL,
    utm_campaign VARCHAR(255) NULL,
    utm_term VARCHAR(255) NULL,
    utm_content VARCHAR(255) NULL,
    total_forms_viewed INT DEFAULT 0,
    total_forms_started INT DEFAULT 0,
    total_forms_submitted INT DEFAULT 0,
    total_session_time INT DEFAULT 0, -- in seconds
    is_bot BOOLEAN DEFAULT FALSE,
    bot_detection_score DECIMAL(3, 2) DEFAULT 0.00,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP NULL,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_user_sessions_session_id (session_id),
    INDEX idx_user_sessions_user_id (user_id),
    INDEX idx_user_sessions_ip_address (ip_address),
    INDEX idx_user_sessions_started_at (started_at),
    INDEX idx_user_sessions_country (country_code),
    INDEX idx_user_sessions_utm_source (utm_source),
    INDEX idx_user_sessions_is_bot (is_bot)
);

-- Form A/B Test Variants Table
CREATE TABLE form_ab_test_variants (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    form_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    test_name VARCHAR(255) NOT NULL,
    variant_name VARCHAR(255) NOT NULL,
    variant_config JSON NOT NULL, -- Form configuration changes
    traffic_percentage INT NOT NULL DEFAULT 50,
    is_active BOOLEAN DEFAULT TRUE,
    status ENUM('draft', 'active', 'paused', 'completed') DEFAULT 'draft',
    started_at TIMESTAMP NULL,
    ended_at TIMESTAMP NULL,
    total_views INT DEFAULT 0,
    total_submissions INT DEFAULT 0,
    conversion_rate DECIMAL(5, 2) DEFAULT 0.00,
    confidence_level DECIMAL(3, 2) NULL, -- Statistical confidence
    is_winner BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_form_ab_test_variants_form_id (form_id),
    INDEX idx_form_ab_test_variants_user_id (user_id),
    INDEX idx_form_ab_test_variants_test_name (test_name),
    INDEX idx_form_ab_test_variants_status (status),
    INDEX idx_form_ab_test_variants_is_winner (is_winner),
    UNIQUE KEY unique_form_test_variant (form_id, test_name, variant_name)
);

-- Geographic Analytics Aggregation Table
CREATE TABLE form_geographic_analytics (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    form_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    date DATE NOT NULL,
    country_code CHAR(2) NOT NULL,
    country_name VARCHAR(100) NOT NULL,
    region VARCHAR(100) NULL,
    city VARCHAR(100) NULL,
    total_views INT DEFAULT 0,
    total_submissions INT DEFAULT 0,
    conversion_rate DECIMAL(5, 2) DEFAULT 0.00,
    bounce_rate DECIMAL(5, 2) DEFAULT 0.00,
    average_session_time INT DEFAULT 0, -- in seconds
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY unique_geographic_analytics (form_id, date, country_code, region, city),
    INDEX idx_form_geographic_analytics_form_id (form_id),
    INDEX idx_form_geographic_analytics_user_id (user_id),
    INDEX idx_form_geographic_analytics_date (date),
    INDEX idx_form_geographic_analytics_country (country_code),
    INDEX idx_form_geographic_analytics_conversion_rate (conversion_rate)
);

-- Device & Browser Analytics Table
CREATE TABLE form_device_analytics (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    form_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    date DATE NOT NULL,
    device_type ENUM('desktop', 'tablet', 'mobile', 'unknown') NOT NULL,
    browser_name VARCHAR(100) NOT NULL,
    browser_version VARCHAR(50) NULL,
    os_name VARCHAR(100) NOT NULL,
    os_version VARCHAR(50) NULL,
    total_views INT DEFAULT 0,
    total_submissions INT DEFAULT 0,
    conversion_rate DECIMAL(5, 2) DEFAULT 0.00,
    bounce_rate DECIMAL(5, 2) DEFAULT 0.00,
    average_completion_time INT DEFAULT 0, -- in seconds
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY unique_device_analytics (form_id, date, device_type, browser_name, os_name),
    INDEX idx_form_device_analytics_form_id (form_id),
    INDEX idx_form_device_analytics_user_id (user_id),
    INDEX idx_form_device_analytics_date (date),
    INDEX idx_form_device_analytics_device_type (device_type),
    INDEX idx_form_device_analytics_browser (browser_name),
    INDEX idx_form_device_analytics_conversion_rate (conversion_rate)
);

-- Field-Level Analytics Table
CREATE TABLE form_field_analytics (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    form_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    field_name VARCHAR(255) NOT NULL,
    date DATE NOT NULL,
    total_interactions INT DEFAULT 0,
    total_focus_events INT DEFAULT 0,
    total_blur_events INT DEFAULT 0,
    total_changes INT DEFAULT 0,
    total_validation_errors INT DEFAULT 0,
    average_time_to_fill INT DEFAULT 0, -- in seconds
    abandonment_rate DECIMAL(5, 2) DEFAULT 0.00,
    error_rate DECIMAL(5, 2) DEFAULT 0.00,
    common_errors JSON NULL, -- {"error_type": count}
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY unique_field_analytics (form_id, field_name, date),
    INDEX idx_form_field_analytics_form_id (form_id),
    INDEX idx_form_field_analytics_user_id (user_id),
    INDEX idx_form_field_analytics_field_name (field_name),
    INDEX idx_form_field_analytics_date (date),
    INDEX idx_form_field_analytics_error_rate (error_rate),
    INDEX idx_form_field_analytics_abandonment_rate (abandonment_rate)
);

-- Automated Reports Configuration Table
CREATE TABLE automated_reports (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id CHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NULL,
    report_type ENUM(
        'daily_summary', 'weekly_summary', 'monthly_summary',
        'conversion_analysis', 'geographic_breakdown', 'device_analysis',
        'field_performance', 'spam_analysis', 'custom'
    ) NOT NULL,
    forms_included JSON NULL, -- Array of form IDs, NULL for all forms
    frequency ENUM('daily', 'weekly', 'monthly', 'quarterly') NOT NULL,
    email_recipients JSON NOT NULL, -- Array of email addresses
    report_format ENUM('pdf', 'html', 'csv', 'json') NOT NULL DEFAULT 'pdf',
    custom_config JSON NULL, -- Custom report parameters
    timezone VARCHAR(100) DEFAULT 'UTC',
    send_time TIME DEFAULT '09:00:00',
    is_active BOOLEAN DEFAULT TRUE,
    last_sent_at TIMESTAMP NULL,
    next_send_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_automated_reports_user_id (user_id),
    INDEX idx_automated_reports_type (report_type),
    INDEX idx_automated_reports_frequency (frequency),
    INDEX idx_automated_reports_next_send (next_send_at),
    INDEX idx_automated_reports_is_active (is_active)
);

-- Real-time Monitoring Alerts Table
CREATE TABLE monitoring_alerts (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id CHAR(36) NOT NULL,
    alert_name VARCHAR(255) NOT NULL,
    alert_type ENUM(
        'high_spam_rate', 'low_conversion_rate', 'high_abandonment_rate',
        'unusual_traffic', 'form_errors', 'webhook_failures', 'email_delivery_issues',
        'suspicious_activity', 'form_downtime', 'custom'
    ) NOT NULL,
    form_ids JSON NULL, -- Array of form IDs to monitor, NULL for all forms
    conditions JSON NOT NULL, -- Alert conditions and thresholds
    notification_methods JSON NOT NULL, -- ["email", "webhook", "slack"]
    notification_config JSON NOT NULL, -- Email addresses, webhook URLs, etc.
    cooldown_minutes INT DEFAULT 60, -- Minimum time between alerts
    is_active BOOLEAN DEFAULT TRUE,
    last_triggered_at TIMESTAMP NULL,
    trigger_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_monitoring_alerts_user_id (user_id),
    INDEX idx_monitoring_alerts_type (alert_type),
    INDEX idx_monitoring_alerts_is_active (is_active),
    INDEX idx_monitoring_alerts_last_triggered (last_triggered_at)
);

-- Alert Trigger History Table
CREATE TABLE alert_trigger_history (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    alert_id CHAR(36) NOT NULL,
    form_id CHAR(36) NULL,
    trigger_reason TEXT NOT NULL,
    trigger_data JSON NOT NULL,
    severity ENUM('low', 'medium', 'high', 'critical') NOT NULL,
    is_resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP NULL,
    resolved_by CHAR(36) NULL,
    resolution_notes TEXT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (alert_id) REFERENCES monitoring_alerts(id) ON DELETE CASCADE,
    FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE,
    FOREIGN KEY (resolved_by) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_alert_trigger_history_alert_id (alert_id),
    INDEX idx_alert_trigger_history_form_id (form_id),
    INDEX idx_alert_trigger_history_created_at (created_at),
    INDEX idx_alert_trigger_history_severity (severity),
    INDEX idx_alert_trigger_history_is_resolved (is_resolved)
);

-- Performance Metrics Table for API monitoring
CREATE TABLE api_performance_metrics (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    endpoint_path VARCHAR(255) NOT NULL,
    http_method ENUM('GET', 'POST', 'PUT', 'DELETE', 'PATCH') NOT NULL,
    response_time_ms INT NOT NULL,
    status_code INT NOT NULL,
    user_id CHAR(36) NULL,
    form_id CHAR(36) NULL,
    ip_address VARCHAR(45) NOT NULL,
    user_agent TEXT NULL,
    request_size_bytes INT NULL,
    response_size_bytes INT NULL,
    error_message TEXT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE SET NULL,
    INDEX idx_api_performance_metrics_endpoint (endpoint_path),
    INDEX idx_api_performance_metrics_method (http_method),
    INDEX idx_api_performance_metrics_created_at (created_at),
    INDEX idx_api_performance_metrics_status_code (status_code),
    INDEX idx_api_performance_metrics_user_id (user_id),
    INDEX idx_api_performance_metrics_response_time (response_time_ms)
);

-- Create aggregated analytics views for better performance
CREATE VIEW v_form_daily_stats AS
SELECT 
    f.id as form_id,
    f.user_id,
    DATE(s.created_at) as date,
    COUNT(*) as total_submissions,
    COUNT(CASE WHEN s.is_spam = FALSE THEN 1 END) as valid_submissions,
    COUNT(CASE WHEN s.is_spam = TRUE THEN 1 END) as spam_submissions,
    AVG(CASE WHEN s.is_spam = FALSE THEN s.spam_score END) as avg_spam_score,
    COUNT(CASE WHEN s.email_sent = TRUE THEN 1 END) as emails_sent,
    COUNT(CASE WHEN s.webhook_sent = TRUE THEN 1 END) as webhooks_sent
FROM forms f
LEFT JOIN submissions s ON f.id = s.form_id
GROUP BY f.id, f.user_id, DATE(s.created_at);

-- Create a view for real-time form analytics
CREATE VIEW v_form_realtime_stats AS
SELECT 
    f.id as form_id,
    f.user_id,
    f.name as form_name,
    COUNT(s.id) as total_submissions_today,
    COUNT(CASE WHEN s.created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR) THEN 1 END) as submissions_last_hour,
    COUNT(CASE WHEN s.created_at >= DATE_SUB(NOW(), INTERVAL 15 MINUTE) THEN 1 END) as submissions_last_15min,
    COUNT(CASE WHEN s.is_spam = TRUE AND DATE(s.created_at) = CURDATE() THEN 1 END) as spam_today,
    AVG(CASE WHEN DATE(s.created_at) = CURDATE() AND s.is_spam = FALSE THEN s.spam_score END) as avg_spam_score_today,
    MAX(s.created_at) as last_submission_at
FROM forms f
LEFT JOIN submissions s ON f.id = s.form_id AND DATE(s.created_at) = CURDATE()
WHERE f.is_active = TRUE
GROUP BY f.id, f.user_id, f.name;

-- Create trigger to update form submission count
DELIMITER //
CREATE TRIGGER update_form_submission_count_insert 
AFTER INSERT ON submissions
FOR EACH ROW
BEGIN
    UPDATE forms 
    SET submission_count = submission_count + 1 
    WHERE id = NEW.form_id;
END;//

CREATE TRIGGER update_form_submission_count_delete
AFTER DELETE ON submissions
FOR EACH ROW
BEGIN
    UPDATE forms 
    SET submission_count = GREATEST(0, submission_count - 1) 
    WHERE id = OLD.form_id;
END;//
DELIMITER ;

-- Insert default monitoring alerts for new users (will be handled by application)
-- This will be managed through the application when users sign up

-- Create indexes for time-series data partitioning (MySQL 8.0+)
-- These would be created programmatically based on MySQL version

-- Add analytics-related columns to forms table
ALTER TABLE forms 
ADD COLUMN analytics_enabled BOOLEAN DEFAULT TRUE AFTER is_active,
ADD COLUMN conversion_tracking BOOLEAN DEFAULT TRUE AFTER analytics_enabled,
ADD COLUMN heat_map_enabled BOOLEAN DEFAULT FALSE AFTER conversion_tracking,
ADD COLUMN session_recording BOOLEAN DEFAULT FALSE AFTER heat_map_enabled;

-- Add tracking fields to submissions table for enhanced analytics
ALTER TABLE submissions
ADD COLUMN session_id VARCHAR(255) NULL AFTER created_at,
ADD COLUMN form_variant_id CHAR(36) NULL AFTER session_id,
ADD COLUMN conversion_value DECIMAL(10, 2) NULL AFTER form_variant_id,
ADD COLUMN conversion_currency CHAR(3) NULL AFTER conversion_value,
ADD COLUMN source_attribution JSON NULL AFTER conversion_currency;

-- Create indexes for new fields
CREATE INDEX idx_submissions_session_id ON submissions(session_id);
CREATE INDEX idx_submissions_form_variant_id ON submissions(form_variant_id);
CREATE INDEX idx_submissions_conversion_value ON submissions(conversion_value);