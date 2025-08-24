package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"formhub/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type FormService struct {
	db    *sql.DB
	redis *redis.Client
}

func NewFormService(db *sql.DB, redis *redis.Client) *FormService {
	return &FormService{
		db:    db,
		redis: redis,
	}
}

func (s *FormService) CreateForm(userID uuid.UUID, req models.CreateFormRequest) (*models.Form, error) {
	// Check plan limits
	user, err := s.getUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	limits := models.PlanLimitsMap[user.PlanType]
	if limits.FormsLimit != -1 {
		formCount, err := s.getUserFormCount(userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get form count: %w", err)
		}
		if formCount >= limits.FormsLimit {
			return nil, fmt.Errorf("form limit reached for plan %s", user.PlanType)
		}
	}

	form := &models.Form{
		ID:             uuid.New(),
		UserID:         userID,
		Name:           req.Name,
		Description:    req.Description,
		TargetEmail:    req.TargetEmail,
		Subject:        req.Subject,
		SuccessMessage: req.SuccessMessage,
		RedirectURL:    req.RedirectURL,
		WebhookURL:     req.WebhookURL,
		SpamProtection: req.SpamProtection,
		RecaptchaSecret: req.RecaptchaSecret,
		FileUploads:    req.FileUploads && limits.FileUploads,
		MaxFileSize:    req.MaxFileSize,
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Convert slice fields to JSON
	if len(req.CCEmails) > 0 {
		ccEmailsJSON, _ := json.Marshal(req.CCEmails)
		form.CCEmails = string(ccEmailsJSON)
	}

	if len(req.AllowedOrigins) > 0 {
		originsJSON, _ := json.Marshal(req.AllowedOrigins)
		form.AllowedOrigins = string(originsJSON)
	}

	query := `
		INSERT INTO forms (id, user_id, name, description, target_email, cc_emails, subject, 
			success_message, redirect_url, webhook_url, spam_protection, recaptcha_secret,
			file_uploads, max_file_size, allowed_origins, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	_, err = s.db.Exec(query,
		form.ID, form.UserID, form.Name, form.Description, form.TargetEmail,
		form.CCEmails, form.Subject, form.SuccessMessage, form.RedirectURL,
		form.WebhookURL, form.SpamProtection, form.RecaptchaSecret,
		form.FileUploads, form.MaxFileSize, form.AllowedOrigins,
		form.IsActive, form.CreatedAt, form.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create form: %w", err)
	}

	return form, nil
}

func (s *FormService) GetFormByID(formID uuid.UUID) (*models.Form, error) {
	form := &models.Form{}
	query := `
		SELECT id, user_id, name, description, target_email, cc_emails, subject,
			success_message, redirect_url, webhook_url, spam_protection, recaptcha_secret,
			file_uploads, max_file_size, allowed_origins, is_active, submission_count,
			created_at, updated_at
		FROM forms WHERE id = $1 AND is_active = true
	`

	err := s.db.QueryRow(query, formID).Scan(
		&form.ID, &form.UserID, &form.Name, &form.Description, &form.TargetEmail,
		&form.CCEmails, &form.Subject, &form.SuccessMessage, &form.RedirectURL,
		&form.WebhookURL, &form.SpamProtection, &form.RecaptchaSecret,
		&form.FileUploads, &form.MaxFileSize, &form.AllowedOrigins,
		&form.IsActive, &form.SubmissionCount, &form.CreatedAt, &form.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("form not found")
		}
		return nil, fmt.Errorf("failed to get form: %w", err)
	}

	return form, nil
}

func (s *FormService) GetUserForms(userID uuid.UUID) ([]models.Form, error) {
	query := `
		SELECT id, user_id, name, description, target_email, cc_emails, subject,
			success_message, redirect_url, webhook_url, spam_protection, recaptcha_secret,
			file_uploads, max_file_size, allowed_origins, is_active, submission_count,
			created_at, updated_at
		FROM forms WHERE user_id = $1 AND is_active = true
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get forms: %w", err)
	}
	defer rows.Close()

	var forms []models.Form
	for rows.Next() {
		var form models.Form
		err := rows.Scan(
			&form.ID, &form.UserID, &form.Name, &form.Description, &form.TargetEmail,
			&form.CCEmails, &form.Subject, &form.SuccessMessage, &form.RedirectURL,
			&form.WebhookURL, &form.SpamProtection, &form.RecaptchaSecret,
			&form.FileUploads, &form.MaxFileSize, &form.AllowedOrigins,
			&form.IsActive, &form.SubmissionCount, &form.CreatedAt, &form.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan form: %w", err)
		}
		forms = append(forms, form)
	}

	return forms, nil
}

func (s *FormService) UpdateForm(formID uuid.UUID, userID uuid.UUID, req models.CreateFormRequest) (*models.Form, error) {
	// Verify ownership
	form, err := s.GetFormByID(formID)
	if err != nil {
		return nil, err
	}

	if form.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	// Get user for plan limits
	user, err := s.getUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	limits := models.PlanLimitsMap[user.PlanType]

	// Convert slice fields to JSON
	ccEmailsJSON := ""
	if len(req.CCEmails) > 0 {
		ccEmails, _ := json.Marshal(req.CCEmails)
		ccEmailsJSON = string(ccEmails)
	}

	originsJSON := ""
	if len(req.AllowedOrigins) > 0 {
		origins, _ := json.Marshal(req.AllowedOrigins)
		originsJSON = string(origins)
	}

	query := `
		UPDATE forms SET 
			name = $2, description = $3, target_email = $4, cc_emails = $5,
			subject = $6, success_message = $7, redirect_url = $8, webhook_url = $9,
			spam_protection = $10, recaptcha_secret = $11, file_uploads = $12,
			max_file_size = $13, allowed_origins = $14, updated_at = $15
		WHERE id = $1 AND user_id = $16
	`

	_, err = s.db.Exec(query,
		formID, req.Name, req.Description, req.TargetEmail, ccEmailsJSON,
		req.Subject, req.SuccessMessage, req.RedirectURL, req.WebhookURL,
		req.SpamProtection, req.RecaptchaSecret, req.FileUploads && limits.FileUploads,
		req.MaxFileSize, originsJSON, time.Now(), userID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update form: %w", err)
	}

	return s.GetFormByID(formID)
}

func (s *FormService) DeleteForm(formID uuid.UUID, userID uuid.UUID) error {
	query := `
		UPDATE forms SET is_active = false, updated_at = $1 
		WHERE id = $2 AND user_id = $3
	`

	result, err := s.db.Exec(query, time.Now(), formID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete form: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("form not found or unauthorized")
	}

	return nil
}

func (s *FormService) IncrementSubmissionCount(formID uuid.UUID) error {
	query := `
		UPDATE forms SET submission_count = submission_count + 1, updated_at = $1
		WHERE id = $2
	`

	_, err := s.db.Exec(query, time.Now(), formID)
	if err != nil {
		return fmt.Errorf("failed to increment submission count: %w", err)
	}

	return nil
}

func (s *FormService) getUserByID(userID uuid.UUID) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, first_name, last_name, company, plan_type, is_active,
			created_at, updated_at
		FROM users WHERE id = $1 AND is_active = true
	`

	err := s.db.QueryRow(query, userID).Scan(
		&user.ID, &user.Email, &user.FirstName, &user.LastName,
		&user.Company, &user.PlanType, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *FormService) getUserFormCount(userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM forms WHERE user_id = $1 AND is_active = true`
	err := s.db.QueryRow(query, userID).Scan(&count)
	return count, err
}