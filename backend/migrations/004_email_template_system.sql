-- Email Template System Migration
-- This migration adds comprehensive email template, provider, and analytics support

-- Email Providers Table
CREATE TABLE email_providers (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id CHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    type ENUM('smtp', 'sendgrid', 'mailgun', 'aws_ses', 'postmark', 'mailjet') NOT NULL,
    config JSON NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_email_providers_user_id (user_id),
    INDEX idx_email_providers_type (type),
    INDEX idx_email_providers_is_default (is_default, user_id)
);

-- Email Template Categories Table
CREATE TABLE email_template_categories (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id CHAR(36) NULL, -- NULL for system categories
    name VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(7) DEFAULT '#6366f1', -- Hex color
    icon VARCHAR(50) DEFAULT 'mail',
    category_order INT DEFAULT 0,
    is_system BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_email_template_categories_user_id (user_id),
    INDEX idx_email_template_categories_system (is_system),
    UNIQUE KEY unique_category_name_user (name, user_id)
);

-- Email Templates Table
CREATE TABLE email_templates (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id CHAR(36) NOT NULL,
    form_id CHAR(36) NULL, -- Optional: specific to a form
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type ENUM('notification', 'autoresponder', 'welcome', 'confirmation', 'follow_up', 'custom') NOT NULL,
    language VARCHAR(10) DEFAULT 'en',
    subject VARCHAR(255) NOT NULL,
    html_content LONGTEXT NOT NULL,
    text_content TEXT,
    variables JSON, -- Array of available variables
    parent_id CHAR(36) NULL, -- For template inheritance
    is_active BOOLEAN DEFAULT TRUE,
    is_default BOOLEAN DEFAULT FALSE,
    version INT DEFAULT 1,
    tags JSON, -- Array of tags for categorization
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES email_templates(id) ON DELETE SET NULL,
    INDEX idx_email_templates_user_id (user_id),
    INDEX idx_email_templates_form_id (form_id),
    INDEX idx_email_templates_type (type),
    INDEX idx_email_templates_language (language),
    INDEX idx_email_templates_parent_id (parent_id),
    FULLTEXT KEY ft_email_templates_content (name, description, subject, html_content, text_content)
);

-- Email Autoresponders Table
CREATE TABLE email_autoresponders (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id CHAR(36) NOT NULL,
    form_id CHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    template_id CHAR(36) NOT NULL,
    provider_id CHAR(36) NULL, -- NULL uses default provider
    is_enabled BOOLEAN DEFAULT TRUE,
    delay_minutes INT DEFAULT 0, -- 0 for immediate
    conditions JSON, -- AutoresponderConditions as JSON
    send_to_field VARCHAR(255) NOT NULL, -- Form field containing recipient email
    cc_emails JSON, -- Array of CC emails
    bcc_emails JSON, -- Array of BCC emails
    reply_to VARCHAR(255),
    track_opens BOOLEAN DEFAULT FALSE,
    track_clicks BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE,
    FOREIGN KEY (template_id) REFERENCES email_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (provider_id) REFERENCES email_providers(id) ON DELETE SET NULL,
    INDEX idx_email_autoresponders_user_id (user_id),
    INDEX idx_email_autoresponders_form_id (form_id),
    INDEX idx_email_autoresponders_template_id (template_id),
    INDEX idx_email_autoresponders_enabled (is_enabled)
);

-- Email Queue Table
CREATE TABLE email_queue (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id CHAR(36) NOT NULL,
    form_id CHAR(36) NULL,
    submission_id CHAR(36) NULL,
    template_id CHAR(36) NOT NULL,
    provider_id CHAR(36) NULL,
    to_emails JSON NOT NULL, -- Array of recipient emails
    cc_emails JSON, -- Array of CC emails
    bcc_emails JSON, -- Array of BCC emails
    subject VARCHAR(255) NOT NULL,
    html_content LONGTEXT NOT NULL,
    text_content TEXT,
    variables JSON, -- Template variables as JSON
    scheduled_at TIMESTAMP NOT NULL,
    sent_at TIMESTAMP NULL,
    status ENUM('pending', 'sending', 'sent', 'failed', 'cancelled', 'scheduled') DEFAULT 'pending',
    attempts INT DEFAULT 0,
    last_error TEXT,
    priority INT DEFAULT 0, -- Higher number = higher priority
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE,
    FOREIGN KEY (submission_id) REFERENCES submissions(id) ON DELETE CASCADE,
    FOREIGN KEY (template_id) REFERENCES email_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (provider_id) REFERENCES email_providers(id) ON DELETE SET NULL,
    INDEX idx_email_queue_user_id (user_id),
    INDEX idx_email_queue_status (status),
    INDEX idx_email_queue_scheduled (scheduled_at, status),
    INDEX idx_email_queue_priority (priority DESC, scheduled_at ASC),
    INDEX idx_email_queue_template_id (template_id)
);

-- Email Analytics Table
CREATE TABLE email_analytics (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    queue_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    form_id CHAR(36) NULL,
    template_id CHAR(36) NOT NULL,
    email_address VARCHAR(255) NOT NULL,
    delivered_at TIMESTAMP NULL,
    opened_at TIMESTAMP NULL,
    first_clicked_at TIMESTAMP NULL,
    open_count INT DEFAULT 0,
    click_count INT DEFAULT 0,
    links JSON, -- Array of LinkClick objects
    user_agent TEXT,
    ip_address VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (queue_id) REFERENCES email_queue(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE,
    FOREIGN KEY (template_id) REFERENCES email_templates(id) ON DELETE CASCADE,
    INDEX idx_email_analytics_queue_id (queue_id),
    INDEX idx_email_analytics_user_id (user_id),
    INDEX idx_email_analytics_template_id (template_id),
    INDEX idx_email_analytics_email (email_address),
    INDEX idx_email_analytics_opened (opened_at),
    INDEX idx_email_analytics_clicked (first_clicked_at)
);

-- Email A/B Tests Table
CREATE TABLE email_ab_tests (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id CHAR(36) NOT NULL,
    form_id CHAR(36) NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    template_a_id CHAR(36) NOT NULL,
    template_b_id CHAR(36) NOT NULL,
    traffic_split INT DEFAULT 50, -- Percentage for template A (0-100)
    status ENUM('draft', 'active', 'paused', 'ended') DEFAULT 'draft',
    started_at TIMESTAMP NULL,
    ended_at TIMESTAMP NULL,
    winner CHAR(36) NULL, -- Template ID of the winner
    stats_sent_a INT DEFAULT 0,
    stats_sent_b INT DEFAULT 0,
    stats_open_a INT DEFAULT 0,
    stats_open_b INT DEFAULT 0,
    stats_click_a INT DEFAULT 0,
    stats_click_b INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE,
    FOREIGN KEY (template_a_id) REFERENCES email_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (template_b_id) REFERENCES email_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (winner) REFERENCES email_templates(id) ON DELETE SET NULL,
    INDEX idx_email_ab_tests_user_id (user_id),
    INDEX idx_email_ab_tests_form_id (form_id),
    INDEX idx_email_ab_tests_status (status)
);

-- Email Template Library Table (for pre-built templates)
CREATE TABLE email_template_library (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type ENUM('notification', 'autoresponder', 'welcome', 'confirmation', 'follow_up', 'custom') NOT NULL,
    category_id CHAR(36) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    html_content LONGTEXT NOT NULL,
    text_content TEXT,
    variables JSON, -- Array of available variables
    tags JSON, -- Array of tags
    preview TEXT, -- Base64 image or URL for template preview
    is_public BOOLEAN DEFAULT TRUE,
    usage_count INT DEFAULT 0,
    rating DECIMAL(3,2) DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (category_id) REFERENCES email_template_categories(id) ON DELETE CASCADE,
    INDEX idx_email_template_library_type (type),
    INDEX idx_email_template_library_category (category_id),
    INDEX idx_email_template_library_public (is_public),
    INDEX idx_email_template_library_rating (rating DESC),
    FULLTEXT KEY ft_email_template_library_content (name, description, subject)
);

-- Insert Default Email Template Categories
INSERT INTO email_template_categories (id, user_id, name, description, color, icon, category_order, is_system) VALUES
(UUID(), NULL, 'Notifications', 'Templates for form submission notifications', '#3b82f6', 'bell', 1, TRUE),
(UUID(), NULL, 'Autoresponders', 'Templates for automatic responses to users', '#10b981', 'reply', 2, TRUE),
(UUID(), NULL, 'Welcome', 'Templates for welcoming new users', '#f59e0b', 'hand-wave', 3, TRUE),
(UUID(), NULL, 'Confirmations', 'Templates for email confirmations', '#8b5cf6', 'check-circle', 4, TRUE),
(UUID(), NULL, 'Follow-ups', 'Templates for follow-up communications', '#ef4444', 'clock', 5, TRUE),
(UUID(), NULL, 'Marketing', 'Templates for marketing communications', '#ec4899', 'megaphone', 6, TRUE);

-- Insert Default Email Templates
-- Get the category IDs for default templates
SET @notification_cat = (SELECT id FROM email_template_categories WHERE name = 'Notifications' AND is_system = TRUE LIMIT 1);
SET @autoresponder_cat = (SELECT id FROM email_template_categories WHERE name = 'Autoresponders' AND is_system = TRUE LIMIT 1);
SET @welcome_cat = (SELECT id FROM email_template_categories WHERE name = 'Welcome' AND is_system = TRUE LIMIT 1);

-- Default notification template
INSERT INTO email_template_library (id, name, description, type, category_id, subject, html_content, text_content, variables, tags) VALUES
(UUID(), 'Modern Notification', 'Clean and modern template for form submission notifications', 'notification', @notification_cat, 
'New Form Submission: {{form_name}}',
'<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>New Form Submission</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; line-height: 1.6; color: #374151; margin: 0; padding: 0; background-color: #f9fafb; }
        .container { max-width: 600px; margin: 0 auto; background: white; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 40px 30px; text-align: center; }
        .header h1 { margin: 0; font-size: 28px; font-weight: 600; }
        .header p { margin: 10px 0 0; opacity: 0.9; font-size: 16px; }
        .content { padding: 40px 30px; }
        .submission-card { background: #f8fafc; border-radius: 8px; padding: 24px; margin: 20px 0; border-left: 4px solid #667eea; }
        .field { margin-bottom: 16px; }
        .field-label { font-weight: 600; color: #374151; display: block; margin-bottom: 4px; }
        .field-value { color: #6b7280; background: white; padding: 8px 12px; border-radius: 4px; border: 1px solid #e5e7eb; }
        .meta-info { background: #fef3c7; border-radius: 8px; padding: 20px; margin-top: 24px; border-left: 4px solid #f59e0b; }
        .footer { background: #f3f4f6; padding: 24px 30px; text-align: center; color: #6b7280; font-size: 14px; }
        .footer a { color: #667eea; text-decoration: none; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>New Form Submission</h1>
            <p>{{form_name}}</p>
        </div>
        
        <div class="content">
            <div class="submission-card">
                <h3 style="margin-top: 0; color: #374151;">Submission Details</h3>
                {{#each submission_data}}
                <div class="field">
                    <span class="field-label">{{@key}}:</span>
                    <div class="field-value">{{this}}</div>
                </div>
                {{/each}}
            </div>
            
            <div class="meta-info">
                <h4 style="margin-top: 0; color: #92400e;">Technical Information</h4>
                <div class="field">
                    <span class="field-label">IP Address:</span>
                    <div class="field-value">{{ip_address}}</div>
                </div>
                <div class="field">
                    <span class="field-label">Submitted At:</span>
                    <div class="field-value">{{timestamp}}</div>
                </div>
                {{#if referrer}}
                <div class="field">
                    <span class="field-label">Referrer:</span>
                    <div class="field-value">{{referrer}}</div>
                </div>
                {{/if}}
            </div>
        </div>
        
        <div class="footer">
            <p>This email was sent by <a href="https://formhub.io">FormHub</a> - Powerful Form Backend Service</p>
        </div>
    </div>
</body>
</html>',
'New Form Submission - {{form_name}}
======================================

Submission Details:
{{#each submission_data}}
{{@key}}: {{this}}
{{/each}}

Technical Information:
IP Address: {{ip_address}}
Submitted At: {{timestamp}}
{{#if referrer}}Referrer: {{referrer}}{{/if}}

---
This email was sent by FormHub - Powerful Form Backend Service
https://formhub.io',
JSON_ARRAY('form_name', 'submission_data', 'ip_address', 'timestamp', 'referrer', 'user_agent'),
JSON_ARRAY('modern', 'clean', 'professional'));

-- Default autoresponder template  
INSERT INTO email_template_library (id, name, description, type, category_id, subject, html_content, text_content, variables, tags) VALUES
(UUID(), 'Thank You Autoresponder', 'Professional thank you message for form submissions', 'autoresponder', @autoresponder_cat,
'Thank you for contacting us, {{name}}!',
'<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Thank You</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; line-height: 1.6; color: #374151; margin: 0; padding: 0; background-color: #f9fafb; }
        .container { max-width: 600px; margin: 0 auto; background: white; }
        .header { background: linear-gradient(135deg, #10b981 0%, #059669 100%); color: white; padding: 40px 30px; text-align: center; }
        .header h1 { margin: 0; font-size: 28px; font-weight: 600; }
        .content { padding: 40px 30px; }
        .message { font-size: 16px; line-height: 1.8; }
        .highlight { background: #ecfdf5; border-radius: 8px; padding: 20px; margin: 24px 0; border-left: 4px solid #10b981; }
        .footer { background: #f3f4f6; padding: 24px 30px; text-align: center; color: #6b7280; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Thank You!</h1>
        </div>
        
        <div class="content">
            <div class="message">
                <p>Hi {{name}},</p>
                
                <p>Thank you for reaching out to us! We have received your message and appreciate you taking the time to contact us.</p>
                
                <div class="highlight">
                    <strong>What happens next?</strong>
                    <ul>
                        <li>Our team will review your submission within 24 hours</li>
                        <li>We will respond to your inquiry as soon as possible</li>
                        <li>You may receive a follow-up email if we need additional information</li>
                    </ul>
                </div>
                
                <p>If you have any urgent questions or concerns, please don''t hesitate to reach out to us directly.</p>
                
                <p>Best regards,<br>The Team</p>
            </div>
        </div>
        
        <div class="footer">
            <p>This is an automated response. Please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>',
'Thank You for Contacting Us!

Hi {{name}},

Thank you for reaching out to us! We have received your message and appreciate you taking the time to contact us.

What happens next?
- Our team will review your submission within 24 hours
- We will respond to your inquiry as soon as possible  
- You may receive a follow-up email if we need additional information

If you have any urgent questions or concerns, please don''t hesitate to reach out to us directly.

Best regards,
The Team

---
This is an automated response. Please do not reply to this email.',
JSON_ARRAY('name', 'email', 'message', 'company'),
JSON_ARRAY('thank-you', 'autoresponder', 'professional'));

-- Add email provider and template columns to forms table
ALTER TABLE forms 
ADD COLUMN email_provider_id CHAR(36) NULL AFTER webhook_url,
ADD COLUMN notification_template_id CHAR(36) NULL AFTER email_provider_id,
ADD COLUMN autoresponder_template_id CHAR(36) NULL AFTER notification_template_id,
ADD FOREIGN KEY (email_provider_id) REFERENCES email_providers(id) ON DELETE SET NULL,
ADD FOREIGN KEY (notification_template_id) REFERENCES email_templates(id) ON DELETE SET NULL,
ADD FOREIGN KEY (autoresponder_template_id) REFERENCES email_templates(id) ON DELETE SET NULL;

-- Create indexes for performance
CREATE INDEX idx_forms_email_provider ON forms(email_provider_id);
CREATE INDEX idx_forms_notification_template ON forms(notification_template_id);
CREATE INDEX idx_forms_autoresponder_template ON forms(autoresponder_template_id);

-- Template Designs Table (for drag-and-drop builder)
CREATE TABLE template_designs (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id CHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    components JSON NOT NULL, -- Array of TemplateComponent objects
    global_styles JSON, -- GlobalStyles object
    variables JSON, -- Array of available variables
    is_template BOOLEAN DEFAULT FALSE,
    category VARCHAR(100),
    tags JSON, -- Array of tags
    preview_image TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_template_designs_user_id (user_id),
    INDEX idx_template_designs_category (category),
    INDEX idx_template_designs_is_template (is_template),
    FULLTEXT KEY ft_template_designs_content (name, description)
);

-- Add email tracking columns to submissions table
ALTER TABLE submissions
ADD COLUMN notification_sent_at TIMESTAMP NULL AFTER webhook_sent,
ADD COLUMN autoresponder_sent_at TIMESTAMP NULL AFTER notification_sent_at,
ADD COLUMN email_queue_ids JSON NULL AFTER autoresponder_sent_at; -- Track queued emails for this submission