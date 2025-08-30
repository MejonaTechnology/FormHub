package services

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"formhub/internal/models"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type FileUploadService struct {
	db            *sql.DB
	uploadDir     string
	tempUploadDir string
	maxFileSize   int64
	allowedTypes  map[string]bool
}

func NewFileUploadService(db *sql.DB, uploadDir string) *FileUploadService {
	// Default allowed MIME types - restrictive but comprehensive
	defaultAllowedTypes := map[string]bool{
		// Images
		"image/jpeg":    true,
		"image/jpg":     true,
		"image/png":     true,
		"image/gif":     true,
		"image/webp":    true,
		"image/svg+xml": true,
		// Documents
		"application/pdf":                                           true,
		"application/msword":                                        true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.ms-excel":                                                 true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":       true,
		"application/vnd.ms-powerpoint":                                            true,
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
		// Text files
		"text/plain":       true,
		"text/csv":         true,
		"application/json": true,
		"application/xml":  true,
		// Archives
		"application/zip":             true,
		"application/x-zip-compressed": true,
		"application/x-rar-compressed": true,
		// Audio/Video (limited)
		"audio/mpeg":  true,
		"audio/wav":   true,
		"video/mp4":   true,
		"video/webm":  true,
		"video/quicktime": true,
	}

	// Ensure upload directories exist
	os.MkdirAll(uploadDir, 0755)
	tempDir := filepath.Join(uploadDir, "temp")
	os.MkdirAll(tempDir, 0755)

	return &FileUploadService{
		db:            db,
		uploadDir:     uploadDir,
		tempUploadDir: tempDir,
		maxFileSize:   50 * 1024 * 1024, // 50MB default
		allowedTypes:  defaultAllowedTypes,
	}
}

// UploadFile handles single file upload with validation
func (s *FileUploadService) UploadFile(file *multipart.FileHeader, fieldID uuid.UUID, sessionID string) (*models.FileUploadResult, error) {
	// Validate file size
	if file.Size > s.maxFileSize {
		return &models.FileUploadResult{
			Error: fmt.Sprintf("File size %d bytes exceeds maximum allowed size %d bytes", file.Size, s.maxFileSize),
		}, nil
	}

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return &models.FileUploadResult{
			Error: "Failed to open uploaded file",
		}, err
	}
	defer src.Close()

	// Validate file type
	buffer := make([]byte, 512)
	_, err = src.Read(buffer)
	if err != nil {
		return &models.FileUploadResult{
			Error: "Failed to read file content",
		}, err
	}

	// Reset file pointer
	src.Seek(0, 0)

	// Detect MIME type
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Additional validation against file extension
	if !s.isAllowedType(contentType, filepath.Ext(file.Filename)) {
		return &models.FileUploadResult{
			Error: fmt.Sprintf("File type %s is not allowed", contentType),
		}, nil
	}

	// Generate file hash for duplicate detection
	hasher := sha256.New()
	_, err = io.Copy(hasher, src)
	if err != nil {
		return &models.FileUploadResult{
			Error: "Failed to generate file hash",
		}, err
	}
	fileHash := hex.EncodeToString(hasher.Sum(nil))

	// Reset file pointer again
	src.Seek(0, 0)

	// Check for duplicate files
	existingFile, err := s.checkDuplicateFile(fileHash)
	if err == nil && existingFile != nil {
		return &models.FileUploadResult{
			ID:           existingFile.ID,
			FileName:     existingFile.FileName,
			OriginalName: existingFile.OriginalName,
			Size:         existingFile.Size,
			ContentType:  existingFile.ContentType,
		}, nil
	}

	// Generate unique filename
	fileID := uuid.New()
	ext := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("%s%s", fileID.String(), ext)
	filePath := filepath.Join(s.uploadDir, fileName)

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return &models.FileUploadResult{
			Error: "Failed to create destination file",
		}, err
	}
	defer dst.Close()

	// Copy file content
	_, err = io.Copy(dst, src)
	if err != nil {
		os.Remove(filePath) // Cleanup on failure
		return &models.FileUploadResult{
			Error: "Failed to save file",
		}, err
	}

	// Create file upload record
	fileUpload := &models.FileUpload{
		ID:           fileID,
		FileName:     fileName,
		OriginalName: file.Filename,
		ContentType:  contentType,
		Size:         file.Size,
		StoragePath:  filePath,
		CreatedAt:    time.Now(),
	}

	// Store in temporary uploads table for later association
	if sessionID != "" {
		err = s.saveTemporaryFile(fileUpload, fieldID, sessionID, fileHash)
		if err != nil {
			os.Remove(filePath) // Cleanup on failure
			return &models.FileUploadResult{
				Error: "Failed to save file metadata",
			}, err
		}
	}

	return &models.FileUploadResult{
		ID:           fileID,
		FileName:     fileName,
		OriginalName: file.Filename,
		Size:         file.Size,
		ContentType:  contentType,
	}, nil
}

// UploadMultipleFiles handles multiple file uploads with validation
func (s *FileUploadService) UploadMultipleFiles(files []*multipart.FileHeader, fieldID uuid.UUID, sessionID string, maxFiles int) ([]models.FileUploadResult, error) {
	if len(files) > maxFiles && maxFiles > 0 {
		return nil, fmt.Errorf("too many files: %d (maximum: %d)", len(files), maxFiles)
	}

	results := make([]models.FileUploadResult, 0, len(files))
	totalSize := int64(0)

	// Calculate total size
	for _, file := range files {
		totalSize += file.Size
	}

	// Check total size limit (100MB for multiple files)
	if totalSize > 100*1024*1024 {
		return nil, fmt.Errorf("total file size %d bytes exceeds maximum allowed size", totalSize)
	}

	// Upload each file
	for _, file := range files {
		result, err := s.UploadFile(file, fieldID, sessionID)
		if err != nil {
			result = &models.FileUploadResult{
				OriginalName: file.Filename,
				Error:        err.Error(),
			}
		}
		results = append(results, *result)
	}

	return results, nil
}

// ValidateFieldFiles validates uploaded files against form field settings
func (s *FileUploadService) ValidateFieldFiles(files []models.FileUploadResult, fieldSettings *models.FormFieldFileSettings) *models.FormValidationResult {
	result := &models.FormValidationResult{
		IsValid: true,
		Errors:  make(map[string][]string),
	}

	if fieldSettings == nil {
		return result
	}

	// Check file count
	if fieldSettings.MaxFiles > 0 && len(files) > fieldSettings.MaxFiles {
		result.IsValid = false
		result.Errors["file_count"] = []string{fmt.Sprintf("Too many files: %d (maximum: %d)", len(files), fieldSettings.MaxFiles)}
	}

	// Check individual files
	for i, file := range files {
		fileKey := fmt.Sprintf("file_%d", i)
		
		// Check file size
		if fieldSettings.MaxFileSize > 0 && file.Size > fieldSettings.MaxFileSize {
			result.IsValid = false
			if result.Errors[fileKey] == nil {
				result.Errors[fileKey] = []string{}
			}
			result.Errors[fileKey] = append(result.Errors[fileKey], 
				fmt.Sprintf("File size %d bytes exceeds maximum %d bytes", file.Size, fieldSettings.MaxFileSize))
		}

		// Check file type
		if len(fieldSettings.AllowedTypes) > 0 {
			allowed := false
			for _, allowedType := range fieldSettings.AllowedTypes {
				if file.ContentType == allowedType {
					allowed = true
					break
				}
			}
			if !allowed {
				result.IsValid = false
				if result.Errors[fileKey] == nil {
					result.Errors[fileKey] = []string{}
				}
				result.Errors[fileKey] = append(result.Errors[fileKey], 
					fmt.Sprintf("File type %s is not allowed", file.ContentType))
			}
		}

		// Check file extension
		if len(fieldSettings.AllowedExts) > 0 {
			ext := strings.ToLower(filepath.Ext(file.OriginalName))
			allowed := false
			for _, allowedExt := range fieldSettings.AllowedExts {
				if ext == strings.ToLower(allowedExt) {
					allowed = true
					break
				}
			}
			if !allowed {
				result.IsValid = false
				if result.Errors[fileKey] == nil {
					result.Errors[fileKey] = []string{}
				}
				result.Errors[fileKey] = append(result.Errors[fileKey], 
					fmt.Sprintf("File extension %s is not allowed", ext))
			}
		}
	}

	return result
}

// AssociateFilesWithSubmission associates temporary files with a submission
func (s *FileUploadService) AssociateFilesWithSubmission(submissionID uuid.UUID, sessionID string) error {
	query := `
		INSERT INTO file_uploads (id, submission_id, field_id, field_name, file_name, original_name, 
			content_type, size, storage_path, file_hash, created_at)
		SELECT uuid_generate_v4(), ?, 
			(SELECT id FROM form_fields WHERE form_id = t.form_id AND name = t.field_name LIMIT 1),
			field_name, file_name, original_name, content_type, size, storage_path, file_hash, created_at
		FROM temp_file_uploads t
		WHERE session_id = ?
	`

	_, err := s.db.Exec(query, submissionID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to associate files with submission: %w", err)
	}

	// Clean up temporary files
	cleanupQuery := `DELETE FROM temp_file_uploads WHERE session_id = ?`
	_, err = s.db.Exec(cleanupQuery, sessionID)
	
	return err
}

// CleanupExpiredFiles removes expired temporary files
func (s *FileUploadService) CleanupExpiredFiles() error {
	// Get expired files
	query := `SELECT storage_path FROM temp_file_uploads WHERE expires_at < CURRENT_TIMESTAMP`
	rows, err := s.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Delete physical files
	for rows.Next() {
		var storagePath string
		if err := rows.Scan(&storagePath); err != nil {
			continue
		}
		os.Remove(storagePath)
	}

	// Delete database records
	_, err = s.db.Exec("DELETE FROM temp_file_uploads WHERE expires_at < CURRENT_TIMESTAMP")
	return err
}

// GetFileByID retrieves file information by ID
func (s *FileUploadService) GetFileByID(fileID uuid.UUID) (*models.FileUpload, error) {
	query := `
		SELECT id, submission_id, field_id, field_name, file_name, original_name, 
			content_type, size, storage_path, created_at
		FROM file_uploads 
		WHERE id = ?
	`

	var file models.FileUpload
	var fieldID, submissionID sql.NullString
	var fieldName sql.NullString

	err := s.db.QueryRow(query, fileID).Scan(
		&file.ID, &submissionID, &fieldID, &fieldName,
		&file.FileName, &file.OriginalName, &file.ContentType,
		&file.Size, &file.StoragePath, &file.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	if submissionID.Valid {
		file.SubmissionID, _ = uuid.Parse(submissionID.String)
	}

	return &file, nil
}

// Helper methods

func (s *FileUploadService) isAllowedType(contentType, extension string) bool {
	// Check MIME type
	if s.allowedTypes[contentType] {
		return true
	}

	// Check by extension as fallback
	ext := strings.ToLower(extension)
	allowedExtensions := map[string]bool{
		".jpg":  true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".svg": true,
		".pdf":  true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
		".txt":  true, ".csv": true, ".json": true, ".xml": true,
		".zip":  true, ".rar": true, ".7z": true,
		".mp3":  true, ".wav": true, ".mp4": true, ".webm": true, ".mov": true,
	}

	return allowedExtensions[ext]
}

func (s *FileUploadService) checkDuplicateFile(fileHash string) (*models.FileUpload, error) {
	query := `
		SELECT id, file_name, original_name, size, content_type
		FROM file_uploads 
		WHERE file_hash = ? 
		LIMIT 1
	`

	var file models.FileUpload
	err := s.db.QueryRow(query, fileHash).Scan(
		&file.ID, &file.FileName, &file.OriginalName,
		&file.Size, &file.ContentType,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &file, err
}

func (s *FileUploadService) saveTemporaryFile(fileUpload *models.FileUpload, fieldID uuid.UUID, sessionID, fileHash string) error {
	query := `
		INSERT INTO temp_file_uploads (id, field_name, session_id, file_name, original_name,
			content_type, size, storage_path, file_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Files expire after 24 hours
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err := s.db.Exec(query,
		fileUpload.ID, "", sessionID, // field_name will be updated when associated
		fileUpload.FileName, fileUpload.OriginalName,
		fileUpload.ContentType, fileUpload.Size, fileUpload.StoragePath,
		fileHash, expiresAt, fileUpload.CreatedAt,
	)

	return err
}

// GetUploadDirectory returns the upload directory path
func (s *FileUploadService) GetUploadDirectory() string {
	return s.uploadDir
}

// UpdateMaxFileSize updates the maximum file size limit
func (s *FileUploadService) UpdateMaxFileSize(maxSize int64) {
	s.maxFileSize = maxSize
}

// AddAllowedType adds a new allowed MIME type
func (s *FileUploadService) AddAllowedType(mimeType string) {
	s.allowedTypes[mimeType] = true
}

// RemoveAllowedType removes an allowed MIME type
func (s *FileUploadService) RemoveAllowedType(mimeType string) {
	delete(s.allowedTypes, mimeType)
}