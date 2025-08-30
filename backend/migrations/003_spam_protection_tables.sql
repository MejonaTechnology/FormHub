-- Migration: 003_spam_protection_tables.sql
-- Description: Create tables for comprehensive spam protection system
-- Version: 1.0
-- Date: 2025-08-29

-- Table for global spam protection configuration
CREATE TABLE IF NOT EXISTS spam_protection_config (
    id VARCHAR(50) PRIMARY KEY,
    config JSON NOT NULL,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table for per-form spam protection settings
CREATE TABLE IF NOT EXISTS form_spam_configs (
    id VARCHAR(36) PRIMARY KEY DEFAULT (UUID()),
    form_id VARCHAR(255) NOT NULL,
    config JSON NOT NULL,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY idx_form_id (form_id),
    KEY idx_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table for spam analysis logs
CREATE TABLE IF NOT EXISTS spam_analysis_logs (
    id VARCHAR(36) PRIMARY KEY,
    form_id VARCHAR(255) NOT NULL,
    submission_id VARCHAR(36),
    client_ip VARCHAR(45),
    user_agent TEXT,
    spam_score DECIMAL(4,3) NOT NULL DEFAULT 0.000,
    confidence DECIMAL(4,3) NOT NULL DEFAULT 0.000,
    action ENUM('allow', 'block', 'quarantine') NOT NULL,
    triggers_count INT NOT NULL DEFAULT 0,
    triggers JSON,
    is_spam BOOLEAN DEFAULT false,
    processing_time_ms INT,
    metadata JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_form_id (form_id),
    INDEX idx_client_ip (client_ip),
    INDEX idx_action (action),
    INDEX idx_is_spam (is_spam),
    INDEX idx_created_at (created_at),
    INDEX idx_spam_score (spam_score),
    INDEX idx_confidence (confidence)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table for security logs and events
CREATE TABLE IF NOT EXISTS security_logs (
    id VARCHAR(36) PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    severity ENUM('low', 'medium', 'high', 'critical') DEFAULT 'medium',
    client_ip VARCHAR(45),
    user_agent TEXT,
    form_id VARCHAR(255),
    submission_id VARCHAR(36),
    details JSON,
    resolved BOOLEAN DEFAULT false,
    resolved_at TIMESTAMP NULL,
    resolved_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_event_type (event_type),
    INDEX idx_severity (severity),
    INDEX idx_client_ip (client_ip),
    INDEX idx_form_id (form_id),
    INDEX idx_resolved (resolved),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table for IP reputation and analysis
CREATE TABLE IF NOT EXISTS ip_reputation (
    ip VARCHAR(45) PRIMARY KEY,
    reputation ENUM('good', 'neutral', 'suspicious', 'malicious') DEFAULT 'neutral',
    risk_score DECIMAL(4,3) NOT NULL DEFAULT 0.000,
    country_code CHAR(2),
    asn VARCHAR(20),
    is_vpn BOOLEAN DEFAULT false,
    is_proxy BOOLEAN DEFAULT false,
    is_tor BOOLEAN DEFAULT false,
    submission_count INT DEFAULT 0,
    block_count INT DEFAULT 0,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    metadata JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_reputation (reputation),
    INDEX idx_risk_score (risk_score),
    INDEX idx_country_code (country_code),
    INDEX idx_last_seen (last_seen),
    INDEX idx_is_vpn (is_vpn),
    INDEX idx_is_proxy (is_proxy)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table for machine learning models
CREATE TABLE IF NOT EXISTS ml_models (
    id VARCHAR(50) PRIMARY KEY,
    model_type VARCHAR(50) NOT NULL,
    model_data LONGTEXT NOT NULL,
    version VARCHAR(20) NOT NULL,
    accuracy DECIMAL(5,4),
    precision_score DECIMAL(5,4),
    recall_score DECIMAL(5,4),
    f1_score DECIMAL(5,4),
    training_samples INT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_model_type (model_type),
    INDEX idx_version (version),
    INDEX idx_is_active (is_active),
    INDEX idx_updated_at (updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table for behavioral analysis models
CREATE TABLE IF NOT EXISTS behavioral_models (
    model_name VARCHAR(50) PRIMARY KEY,
    model_data JSON NOT NULL,
    sample_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_updated_at (updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table for manual spam labels (training data)
CREATE TABLE IF NOT EXISTS manual_spam_labels (
    id VARCHAR(36) PRIMARY KEY DEFAULT (UUID()),
    submission_id VARCHAR(36),
    form_id VARCHAR(255) NOT NULL,
    submission_data JSON NOT NULL,
    metadata JSON,
    is_spam BOOLEAN NOT NULL,
    labeled_by VARCHAR(100) NOT NULL,
    labeling_source ENUM('admin', 'user_feedback', 'honeypot', 'automated') DEFAULT 'admin',
    confidence ENUM('low', 'medium', 'high') DEFAULT 'medium',
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_form_id (form_id),
    INDEX idx_is_spam (is_spam),
    INDEX idx_labeled_by (labeled_by),
    INDEX idx_labeling_source (labeling_source),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table for user feedback on spam detection
CREATE TABLE IF NOT EXISTS user_feedback (
    id VARCHAR(36) PRIMARY KEY DEFAULT (UUID()),
    submission_id VARCHAR(36),
    form_id VARCHAR(255) NOT NULL,
    submission_data JSON,
    metadata JSON,
    original_spam_score DECIMAL(4,3),
    original_action VARCHAR(20),
    feedback_spam BOOLEAN,
    feedback_source ENUM('form_owner', 'end_user', 'admin') NOT NULL,
    feedback_reason TEXT,
    user_id VARCHAR(36),
    client_ip VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_form_id (form_id),
    INDEX idx_feedback_spam (feedback_spam),
    INDEX idx_feedback_source (feedback_source),
    INDEX idx_user_id (user_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table for CAPTCHA challenges and results
CREATE TABLE IF NOT EXISTS captcha_challenges (
    id VARCHAR(36) PRIMARY KEY DEFAULT (UUID()),
    challenge_token VARCHAR(255) NOT NULL,
    provider ENUM('recaptcha_v2', 'recaptcha_v3', 'hcaptcha', 'turnstile', 'fallback') NOT NULL,
    client_ip VARCHAR(45) NOT NULL,
    user_agent TEXT,
    form_id VARCHAR(255),
    challenge_data JSON,
    is_verified BOOLEAN DEFAULT false,
    verification_score DECIMAL(4,3),
    verification_metadata JSON,
    expires_at TIMESTAMP NOT NULL,
    verified_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY idx_challenge_token (challenge_token),
    INDEX idx_client_ip (client_ip),
    INDEX idx_form_id (form_id),
    INDEX idx_provider (provider),
    INDEX idx_is_verified (is_verified),
    INDEX idx_expires_at (expires_at),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table for honeypot field tracking
CREATE TABLE IF NOT EXISTS honeypot_violations (
    id VARCHAR(36) PRIMARY KEY DEFAULT (UUID()),
    client_ip VARCHAR(45) NOT NULL,
    user_agent TEXT,
    form_id VARCHAR(255) NOT NULL,
    field_name VARCHAR(100) NOT NULL,
    field_value TEXT,
    violation_count INT DEFAULT 1,
    first_violation TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_violation TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_flagged BOOLEAN DEFAULT false,
    flagged_at TIMESTAMP NULL,
    metadata JSON,
    INDEX idx_client_ip (client_ip),
    INDEX idx_form_id (form_id),
    INDEX idx_field_name (field_name),
    INDEX idx_is_flagged (is_flagged),
    INDEX idx_last_violation (last_violation),
    UNIQUE KEY idx_ip_form_field (client_ip, form_id, field_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table for rate limiting tracking
CREATE TABLE IF NOT EXISTS rate_limit_violations (
    id VARCHAR(36) PRIMARY KEY DEFAULT (UUID()),
    client_ip VARCHAR(45) NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    user_agent TEXT,
    violation_count INT DEFAULT 1,
    window_start TIMESTAMP NOT NULL,
    window_end TIMESTAMP NOT NULL,
    limit_exceeded INT NOT NULL,
    current_limit INT NOT NULL,
    is_blocked BOOLEAN DEFAULT false,
    blocked_until TIMESTAMP NULL,
    metadata JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_client_ip (client_ip),
    INDEX idx_endpoint (endpoint),
    INDEX idx_is_blocked (is_blocked),
    INDEX idx_blocked_until (blocked_until),
    INDEX idx_window_end (window_end),
    UNIQUE KEY idx_ip_endpoint_window (client_ip, endpoint, window_start)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table for quarantined submissions
CREATE TABLE IF NOT EXISTS quarantined_submissions (
    id VARCHAR(36) PRIMARY KEY DEFAULT (UUID()),
    original_submission_id VARCHAR(36),
    form_id VARCHAR(255) NOT NULL,
    client_ip VARCHAR(45) NOT NULL,
    user_agent TEXT,
    submission_data JSON NOT NULL,
    metadata JSON,
    spam_score DECIMAL(4,3) NOT NULL,
    confidence DECIMAL(4,3) NOT NULL,
    triggers JSON,
    quarantine_reason TEXT,
    review_status ENUM('pending', 'approved', 'rejected', 'spam') DEFAULT 'pending',
    reviewed_by VARCHAR(100),
    reviewed_at TIMESTAMP NULL,
    review_notes TEXT,
    auto_expire_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_form_id (form_id),
    INDEX idx_client_ip (client_ip),
    INDEX idx_review_status (review_status),
    INDEX idx_spam_score (spam_score),
    INDEX idx_auto_expire_at (auto_expire_at),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table for webhook notifications
CREATE TABLE IF NOT EXISTS webhook_notifications (
    id VARCHAR(36) PRIMARY KEY DEFAULT (UUID()),
    webhook_url VARCHAR(500) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    form_id VARCHAR(255),
    submission_id VARCHAR(36),
    payload JSON NOT NULL,
    status ENUM('pending', 'sent', 'failed', 'retrying') DEFAULT 'pending',
    response_code INT,
    response_body TEXT,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    next_retry_at TIMESTAMP NULL,
    sent_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_event_type (event_type),
    INDEX idx_form_id (form_id),
    INDEX idx_next_retry_at (next_retry_at),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table for spam detection statistics
CREATE TABLE IF NOT EXISTS spam_detection_stats (
    id VARCHAR(36) PRIMARY KEY DEFAULT (UUID()),
    date_key DATE NOT NULL,
    form_id VARCHAR(255),
    total_submissions INT DEFAULT 0,
    spam_detected INT DEFAULT 0,
    blocked INT DEFAULT 0,
    quarantined INT DEFAULT 0,
    challenged INT DEFAULT 0,
    false_positives INT DEFAULT 0,
    true_positives INT DEFAULT 0,
    avg_spam_score DECIMAL(5,4) DEFAULT 0.0000,
    avg_processing_time_ms INT DEFAULT 0,
    top_triggers JSON,
    hourly_distribution JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY idx_date_form (date_key, form_id),
    INDEX idx_date_key (date_key),
    INDEX idx_form_id (form_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table for custom spam rules
CREATE TABLE IF NOT EXISTS custom_spam_rules (
    id VARCHAR(36) PRIMARY KEY DEFAULT (UUID()),
    form_id VARCHAR(255),
    rule_name VARCHAR(100) NOT NULL,
    rule_description TEXT,
    pattern VARCHAR(500) NOT NULL,
    field_target VARCHAR(100) DEFAULT '*',
    action ENUM('block', 'quarantine', 'flag') DEFAULT 'flag',
    score DECIMAL(4,3) NOT NULL DEFAULT 0.500,
    enabled BOOLEAN DEFAULT true,
    created_by VARCHAR(100) NOT NULL,
    hit_count INT DEFAULT 0,
    last_triggered TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_form_id (form_id),
    INDEX idx_enabled (enabled),
    INDEX idx_action (action),
    INDEX idx_created_by (created_by),
    INDEX idx_hit_count (hit_count)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Insert default global configuration
INSERT INTO spam_protection_config (id, config, enabled) 
VALUES ('global', JSON_OBJECT(
    'enabled', true,
    'default_level', 'medium',
    'block_threshold', 0.8,
    'quarantine_threshold', 0.6,
    'require_captcha', false,
    'captcha_provider', 'recaptcha_v3',
    'captcha_fallback', true,
    'rate_limit_enabled', true,
    'max_submissions_per_ip', 10,
    'rate_limit_window_minutes', 15,
    'honeypot_enabled', true,
    'honeypot_fields', JSON_ARRAY('_honeypot', '_hp', '_bot_check'),
    'content_filter_enabled', true,
    'blocked_keywords', JSON_ARRAY('viagra', 'casino', 'lottery', 'spam'),
    'max_url_count', 3,
    'behavioral_enabled', true,
    'min_typing_time_seconds', 2.0,
    'max_typing_speed_wpm', 200.0,
    'ip_reputation_enabled', true,
    'ml_enabled', true,
    'ml_threshold', 0.7,
    'enable_learning', true,
    'notify_on_block', true,
    'notify_on_quarantine', true
), true)
ON DUPLICATE KEY UPDATE 
    config = VALUES(config),
    updated_at = CURRENT_TIMESTAMP;

-- Insert initial behavioral models (empty)
INSERT INTO behavioral_models (model_name, model_data) 
VALUES 
    ('typing_speed', JSON_OBJECT('mean', 45.0, 'standard_deviation', 15.0, 'sample_count', 0)),
    ('interaction', JSON_OBJECT('mean', 3.0, 'standard_deviation', 2.0, 'sample_count', 0)),
    ('mouse_movement', JSON_OBJECT('mean', 25.0, 'standard_deviation', 10.0, 'sample_count', 0)),
    ('keystroke', JSON_OBJECT('mean', 150.0, 'standard_deviation', 50.0, 'sample_count', 0))
ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP;

-- Create triggers for automatic statistics updates
DELIMITER //

CREATE TRIGGER IF NOT EXISTS update_spam_stats_after_log
    AFTER INSERT ON spam_analysis_logs
    FOR EACH ROW
BEGIN
    INSERT INTO spam_detection_stats 
        (date_key, form_id, total_submissions, spam_detected, blocked, quarantined)
    VALUES 
        (DATE(NEW.created_at), NEW.form_id, 1, 
         CASE WHEN NEW.is_spam THEN 1 ELSE 0 END,
         CASE WHEN NEW.action = 'block' THEN 1 ELSE 0 END,
         CASE WHEN NEW.action = 'quarantine' THEN 1 ELSE 0 END)
    ON DUPLICATE KEY UPDATE
        total_submissions = total_submissions + 1,
        spam_detected = spam_detected + CASE WHEN NEW.is_spam THEN 1 ELSE 0 END,
        blocked = blocked + CASE WHEN NEW.action = 'block' THEN 1 ELSE 0 END,
        quarantined = quarantined + CASE WHEN NEW.action = 'quarantine' THEN 1 ELSE 0 END,
        avg_spam_score = (avg_spam_score * (total_submissions - 1) + NEW.spam_score) / total_submissions,
        updated_at = CURRENT_TIMESTAMP;
END //

CREATE TRIGGER IF NOT EXISTS update_custom_rule_hits
    AFTER INSERT ON spam_analysis_logs
    FOR EACH ROW
BEGIN
    -- Update hit counts for triggered custom rules
    IF NEW.triggers IS NOT NULL THEN
        UPDATE custom_spam_rules 
        SET hit_count = hit_count + 1,
            last_triggered = NEW.created_at
        WHERE enabled = true 
        AND JSON_CONTAINS(NEW.triggers, JSON_OBJECT('type', 'custom_rule'));
    END IF;
END //

DELIMITER ;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_spam_analysis_composite ON spam_analysis_logs (form_id, created_at, action, is_spam);
CREATE INDEX IF NOT EXISTS idx_security_logs_composite ON security_logs (event_type, severity, created_at);
CREATE INDEX IF NOT EXISTS idx_ip_reputation_composite ON ip_reputation (reputation, risk_score, last_seen);
CREATE INDEX IF NOT EXISTS idx_quarantine_composite ON quarantined_submissions (review_status, auto_expire_at, created_at);

-- Create a view for spam detection summary
CREATE OR REPLACE VIEW spam_detection_summary AS
SELECT 
    DATE(created_at) as date,
    COUNT(*) as total_requests,
    SUM(CASE WHEN is_spam = 1 THEN 1 ELSE 0 END) as spam_detected,
    SUM(CASE WHEN action = 'block' THEN 1 ELSE 0 END) as blocked,
    SUM(CASE WHEN action = 'quarantine' THEN 1 ELSE 0 END) as quarantined,
    AVG(spam_score) as avg_spam_score,
    AVG(processing_time_ms) as avg_processing_time,
    (SUM(CASE WHEN is_spam = 1 THEN 1 ELSE 0 END) / COUNT(*)) * 100 as spam_percentage
FROM spam_analysis_logs 
WHERE created_at >= DATE_SUB(CURRENT_DATE(), INTERVAL 30 DAY)
GROUP BY DATE(created_at)
ORDER BY date DESC;

-- Create a view for top spam triggers
CREATE OR REPLACE VIEW top_spam_triggers AS
SELECT 
    JSON_UNQUOTE(JSON_EXTRACT(trigger_data.value, '$.type')) as trigger_type,
    JSON_UNQUOTE(JSON_EXTRACT(trigger_data.value, '$.rule')) as rule_name,
    COUNT(*) as trigger_count,
    AVG(spam_score) as avg_spam_score,
    DATE(created_at) as date
FROM spam_analysis_logs,
     JSON_TABLE(triggers, '$[*]' COLUMNS (value JSON PATH '$')) as trigger_data
WHERE created_at >= DATE_SUB(CURRENT_DATE(), INTERVAL 7 DAY)
AND is_spam = 1
GROUP BY trigger_type, rule_name, DATE(created_at)
ORDER BY trigger_count DESC, date DESC;

-- Grant necessary permissions (adjust as needed for your setup)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON formhub.spam_protection_config TO 'formhub_user'@'%';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON formhub.form_spam_configs TO 'formhub_user'@'%';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON formhub.spam_analysis_logs TO 'formhub_user'@'%';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON formhub.security_logs TO 'formhub_user'@'%';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON formhub.ip_reputation TO 'formhub_user'@'%';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON formhub.ml_models TO 'formhub_user'@'%';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON formhub.behavioral_models TO 'formhub_user'@'%';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON formhub.manual_spam_labels TO 'formhub_user'@'%';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON formhub.user_feedback TO 'formhub_user'@'%';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON formhub.captcha_challenges TO 'formhub_user'@'%';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON formhub.honeypot_violations TO 'formhub_user'@'%';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON formhub.rate_limit_violations TO 'formhub_user'@'%';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON formhub.quarantined_submissions TO 'formhub_user'@'%';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON formhub.webhook_notifications TO 'formhub_user'@'%';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON formhub.spam_detection_stats TO 'formhub_user'@'%';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON formhub.custom_spam_rules TO 'formhub_user'@'%';
-- GRANT SELECT ON formhub.spam_detection_summary TO 'formhub_user'@'%';
-- GRANT SELECT ON formhub.top_spam_triggers TO 'formhub_user'@'%';

-- Migration complete