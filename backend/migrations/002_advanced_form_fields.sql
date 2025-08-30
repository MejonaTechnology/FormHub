-- FormHub Database Schema
-- Migration 002: Advanced Form Fields and Validation Support

-- Form Fields table
CREATE TABLE form_fields (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    form_id UUID NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    label VARCHAR(500) NOT NULL,
    type VARCHAR(50) NOT NULL,
    required BOOLEAN NOT NULL DEFAULT false,
    placeholder TEXT,
    default_value TEXT,
    validation JSONB,
    file_settings JSONB,
    field_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure unique field names per form
    CONSTRAINT uq_form_fields_name_form UNIQUE (form_id, name),
    
    -- Validate field types
    CONSTRAINT chk_form_fields_type CHECK (
        type IN ('text', 'email', 'number', 'date', 'time', 'datetime', 
                'url', 'tel', 'textarea', 'select', 'radio', 'checkbox', 
                'file', 'hidden', 'password')
    ),
    
    -- Ensure field order is positive
    CONSTRAINT chk_form_fields_order CHECK (field_order >= 0)
);

-- Form Field Options table (for select, radio, checkbox fields)
CREATE TABLE form_field_options (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    field_id UUID NOT NULL REFERENCES form_fields(id) ON DELETE CASCADE,
    label VARCHAR(500) NOT NULL,
    value TEXT NOT NULL,
    selected BOOLEAN NOT NULL DEFAULT false,
    option_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure unique option values per field
    CONSTRAINT uq_form_field_options_value_field UNIQUE (field_id, value),
    
    -- Ensure option order is positive
    CONSTRAINT chk_form_field_options_order CHECK (option_order >= 0)
);

-- Enhanced File Uploads table with field association
ALTER TABLE file_uploads 
ADD COLUMN field_id UUID REFERENCES form_fields(id) ON DELETE CASCADE,
ADD COLUMN field_name VARCHAR(255),
ADD COLUMN is_validated BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN validation_errors JSONB,
ADD COLUMN file_hash VARCHAR(64), -- SHA-256 hash for duplicate detection
ADD COLUMN expires_at TIMESTAMP WITH TIME ZONE; -- For temporary uploads

-- Form Analytics table for tracking form performance
CREATE TABLE form_analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    form_id UUID NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    submissions_count INTEGER NOT NULL DEFAULT 0,
    spam_count INTEGER NOT NULL DEFAULT 0,
    file_uploads_count INTEGER NOT NULL DEFAULT 0,
    total_file_size BIGINT NOT NULL DEFAULT 0,
    conversion_rate DECIMAL(5,4) DEFAULT 0.0000, -- views to submissions ratio
    bounce_rate DECIMAL(5,4) DEFAULT 0.0000,
    avg_completion_time INTEGER DEFAULT 0, -- in seconds
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure unique analytics per form per day
    CONSTRAINT uq_form_analytics_form_date UNIQUE (form_id, date)
);

-- Form Views table for tracking form visits
CREATE TABLE form_views (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    form_id UUID NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    ip_address INET,
    user_agent TEXT,
    referrer TEXT,
    session_id VARCHAR(255),
    view_duration INTEGER, -- in seconds
    completed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Temporary File Uploads table (for multi-step forms)
CREATE TABLE temp_file_uploads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    form_id UUID NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    field_name VARCHAR(255) NOT NULL,
    session_id VARCHAR(255) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size BIGINT NOT NULL,
    storage_path TEXT NOT NULL,
    file_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for better performance
CREATE INDEX idx_form_fields_form_id ON form_fields(form_id);
CREATE INDEX idx_form_fields_type ON form_fields(type);
CREATE INDEX idx_form_fields_order ON form_fields(form_id, field_order);
CREATE INDEX idx_form_field_options_field_id ON form_field_options(field_id);
CREATE INDEX idx_form_field_options_order ON form_field_options(field_id, option_order);
CREATE INDEX idx_file_uploads_field_id ON file_uploads(field_id);
CREATE INDEX idx_file_uploads_hash ON file_uploads(file_hash);
CREATE INDEX idx_form_analytics_form_date ON form_analytics(form_id, date);
CREATE INDEX idx_form_views_form_id ON form_views(form_id);
CREATE INDEX idx_form_views_session ON form_views(session_id);
CREATE INDEX idx_temp_file_uploads_session ON temp_file_uploads(session_id);
CREATE INDEX idx_temp_file_uploads_expires ON temp_file_uploads(expires_at);

-- Add triggers for automatic timestamp updates
CREATE TRIGGER update_form_fields_updated_at BEFORE UPDATE ON form_fields
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to clean up expired temporary files
CREATE OR REPLACE FUNCTION cleanup_expired_temp_files()
RETURNS void AS $$
BEGIN
    DELETE FROM temp_file_uploads WHERE expires_at < CURRENT_TIMESTAMP;
END;
$$ LANGUAGE plpgsql;

-- Function to update form analytics
CREATE OR REPLACE FUNCTION update_form_analytics(
    p_form_id UUID,
    p_is_spam BOOLEAN DEFAULT false,
    p_file_count INTEGER DEFAULT 0,
    p_file_size BIGINT DEFAULT 0
)
RETURNS void AS $$
BEGIN
    INSERT INTO form_analytics (form_id, date, submissions_count, spam_count, file_uploads_count, total_file_size)
    VALUES (p_form_id, CURRENT_DATE, 
            CASE WHEN p_is_spam THEN 0 ELSE 1 END,
            CASE WHEN p_is_spam THEN 1 ELSE 0 END,
            p_file_count,
            p_file_size)
    ON CONFLICT (form_id, date) DO UPDATE SET
        submissions_count = form_analytics.submissions_count + CASE WHEN p_is_spam THEN 0 ELSE 1 END,
        spam_count = form_analytics.spam_count + CASE WHEN p_is_spam THEN 1 ELSE 0 END,
        file_uploads_count = form_analytics.file_uploads_count + p_file_count,
        total_file_size = form_analytics.total_file_size + p_file_size;
END;
$$ LANGUAGE plpgsql;

-- Enhanced constraints for existing tables
ALTER TABLE forms 
ADD COLUMN has_custom_fields BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN field_validation_mode VARCHAR(20) NOT NULL DEFAULT 'basic', -- 'basic', 'strict', 'custom'
ADD COLUMN auto_response_enabled BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN auto_response_message TEXT,
ADD COLUMN form_version INTEGER NOT NULL DEFAULT 1;

-- Add constraints for new form fields
ALTER TABLE forms ADD CONSTRAINT chk_forms_field_validation_mode 
    CHECK (field_validation_mode IN ('basic', 'strict', 'custom'));

-- Update existing forms to support the new schema
UPDATE forms SET has_custom_fields = false, field_validation_mode = 'basic' WHERE has_custom_fields IS NULL;

-- Comments for documentation
COMMENT ON TABLE form_fields IS 'Stores form field configurations with validation rules and display settings';
COMMENT ON TABLE form_field_options IS 'Stores options for select, radio, and checkbox form fields';
COMMENT ON TABLE form_analytics IS 'Tracks form performance metrics and statistics';
COMMENT ON TABLE form_views IS 'Records form view events for analytics and conversion tracking';
COMMENT ON TABLE temp_file_uploads IS 'Temporary storage for file uploads in multi-step forms';

COMMENT ON COLUMN form_fields.validation IS 'JSON object containing field validation rules (min/max length, pattern, etc.)';
COMMENT ON COLUMN form_fields.file_settings IS 'JSON object containing file upload settings for file fields';
COMMENT ON COLUMN file_uploads.file_hash IS 'SHA-256 hash of file content for duplicate detection';
COMMENT ON COLUMN form_analytics.conversion_rate IS 'Ratio of successful submissions to total form views';