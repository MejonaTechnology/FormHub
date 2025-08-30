package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"formhub/internal/models"
	"strings"
	"time"

	"github.com/google/uuid"
)

type TemplateBuilderService struct {
	db *sql.DB
}

// Template Components
type TemplateComponent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`      // header, text, button, image, divider, social, etc.
	Content   string                 `json:"content"`   // Main content/text
	Properties map[string]interface{} `json:"properties"` // Component-specific properties
	Styles    ComponentStyles        `json:"styles"`    // Styling properties
	Children  []TemplateComponent    `json:"children,omitempty"` // For container components
	Order     int                    `json:"order"`
}

type ComponentStyles struct {
	BackgroundColor  string            `json:"background_color,omitempty"`
	TextColor       string            `json:"text_color,omitempty"`
	FontFamily      string            `json:"font_family,omitempty"`
	FontSize        string            `json:"font_size,omitempty"`
	FontWeight      string            `json:"font_weight,omitempty"`
	TextAlign       string            `json:"text_align,omitempty"`
	Padding         string            `json:"padding,omitempty"`
	Margin          string            `json:"margin,omitempty"`
	BorderRadius    string            `json:"border_radius,omitempty"`
	Border          string            `json:"border,omitempty"`
	Width           string            `json:"width,omitempty"`
	Height          string            `json:"height,omitempty"`
	CustomCSS       string            `json:"custom_css,omitempty"`
	CustomStyles    map[string]string `json:"custom_styles,omitempty"`
}

type TemplateDesign struct {
	ID           uuid.UUID           `json:"id"`
	UserID       uuid.UUID           `json:"user_id"`
	Name         string              `json:"name"`
	Description  string              `json:"description"`
	Components   []TemplateComponent `json:"components"`
	GlobalStyles GlobalStyles        `json:"global_styles"`
	Variables    []string            `json:"variables"`
	IsTemplate   bool                `json:"is_template"` // Whether this is a reusable template
	Category     string              `json:"category"`
	Tags         []string            `json:"tags"`
	PreviewImage string              `json:"preview_image,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

type GlobalStyles struct {
	ContainerWidth   string            `json:"container_width"`
	BackgroundColor  string            `json:"background_color"`
	DefaultFontFamily string           `json:"default_font_family"`
	DefaultTextColor string            `json:"default_text_color"`
	LinkColor       string            `json:"link_color"`
	CustomCSS       string            `json:"custom_css"`
	ResponsiveRules map[string]string `json:"responsive_rules,omitempty"`
}

type TemplateLibraryItem struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Tags        []string  `json:"tags"`
	Preview     string    `json:"preview"`
	IsPublic    bool      `json:"is_public"`
	UsageCount  int       `json:"usage_count"`
	Rating      float64   `json:"rating"`
	Design      TemplateDesign `json:"design"`
}

type BuilderRequest struct {
	Name         string              `json:"name" binding:"required"`
	Description  string              `json:"description"`
	Components   []TemplateComponent `json:"components" binding:"required"`
	GlobalStyles GlobalStyles        `json:"global_styles"`
	Category     string              `json:"category"`
	Tags         []string            `json:"tags"`
	IsTemplate   bool                `json:"is_template"`
}

type PreviewRequest struct {
	Components   []TemplateComponent    `json:"components" binding:"required"`
	GlobalStyles GlobalStyles           `json:"global_styles"`
	Variables    map[string]interface{} `json:"variables"`
}

func NewTemplateBuilderService(db *sql.DB) *TemplateBuilderService {
	return &TemplateBuilderService{
		db: db,
	}
}

// CreateTemplate creates a new template design
func (s *TemplateBuilderService) CreateTemplate(userID uuid.UUID, req BuilderRequest) (*TemplateDesign, error) {
	template := &TemplateDesign{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         req.Name,
		Description:  req.Description,
		Components:   req.Components,
		GlobalStyles: req.GlobalStyles,
		Variables:    s.extractVariablesFromComponents(req.Components),
		IsTemplate:   req.IsTemplate,
		Category:     req.Category,
		Tags:         req.Tags,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Generate preview image (would integrate with screenshot service in production)
	template.PreviewImage = s.generatePreviewImage(template)

	// Insert into database
	componentsJSON, _ := json.Marshal(template.Components)
	globalStylesJSON, _ := json.Marshal(template.GlobalStyles)
	variablesJSON, _ := json.Marshal(template.Variables)
	tagsJSON, _ := json.Marshal(template.Tags)

	query := `
		INSERT INTO template_designs (
			id, user_id, name, description, components, global_styles, 
			variables, is_template, category, tags, preview_image, 
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		template.ID, template.UserID, template.Name, template.Description,
		componentsJSON, globalStylesJSON, variablesJSON, template.IsTemplate,
		template.Category, tagsJSON, template.PreviewImage,
		template.CreatedAt, template.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	return template, nil
}

// GetTemplate retrieves a template design by ID
func (s *TemplateBuilderService) GetTemplate(userID, templateID uuid.UUID) (*TemplateDesign, error) {
	query := `
		SELECT id, user_id, name, description, components, global_styles,
		       variables, is_template, category, tags, preview_image,
		       created_at, updated_at
		FROM template_designs 
		WHERE id = ? AND user_id = ?`

	var template TemplateDesign
	var componentsJSON, globalStylesJSON, variablesJSON, tagsJSON []byte

	err := s.db.QueryRow(query, templateID, userID).Scan(
		&template.ID, &template.UserID, &template.Name, &template.Description,
		&componentsJSON, &globalStylesJSON, &variablesJSON,
		&template.IsTemplate, &template.Category, &tagsJSON,
		&template.PreviewImage, &template.CreatedAt, &template.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Parse JSON fields
	if len(componentsJSON) > 0 {
		json.Unmarshal(componentsJSON, &template.Components)
	}
	if len(globalStylesJSON) > 0 {
		json.Unmarshal(globalStylesJSON, &template.GlobalStyles)
	}
	if len(variablesJSON) > 0 {
		json.Unmarshal(variablesJSON, &template.Variables)
	}
	if len(tagsJSON) > 0 {
		json.Unmarshal(tagsJSON, &template.Tags)
	}

	return &template, nil
}

// ListTemplates retrieves template designs for a user
func (s *TemplateBuilderService) ListTemplates(userID uuid.UUID, category *string, isTemplate *bool) ([]TemplateDesign, error) {
	query := `
		SELECT id, user_id, name, description, components, global_styles,
		       variables, is_template, category, tags, preview_image,
		       created_at, updated_at
		FROM template_designs 
		WHERE user_id = ?`
	
	args := []interface{}{userID}

	if category != nil {
		query += " AND category = ?"
		args = append(args, *category)
	}

	if isTemplate != nil {
		query += " AND is_template = ?"
		args = append(args, *isTemplate)
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	defer rows.Close()

	var templates []TemplateDesign
	for rows.Next() {
		var template TemplateDesign
		var componentsJSON, globalStylesJSON, variablesJSON, tagsJSON []byte

		err := rows.Scan(
			&template.ID, &template.UserID, &template.Name, &template.Description,
			&componentsJSON, &globalStylesJSON, &variablesJSON,
			&template.IsTemplate, &template.Category, &tagsJSON,
			&template.PreviewImage, &template.CreatedAt, &template.UpdatedAt,
		)
		if err != nil {
			continue
		}

		// Parse JSON fields
		if len(componentsJSON) > 0 {
			json.Unmarshal(componentsJSON, &template.Components)
		}
		if len(globalStylesJSON) > 0 {
			json.Unmarshal(globalStylesJSON, &template.GlobalStyles)
		}
		if len(variablesJSON) > 0 {
			json.Unmarshal(variablesJSON, &template.Variables)
		}
		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &template.Tags)
		}

		templates = append(templates, template)
	}

	return templates, nil
}

// UpdateTemplate updates an existing template design
func (s *TemplateBuilderService) UpdateTemplate(userID, templateID uuid.UUID, req BuilderRequest) (*TemplateDesign, error) {
	// Extract variables from updated components
	variables := s.extractVariablesFromComponents(req.Components)

	// Update template
	componentsJSON, _ := json.Marshal(req.Components)
	globalStylesJSON, _ := json.Marshal(req.GlobalStyles)
	variablesJSON, _ := json.Marshal(variables)
	tagsJSON, _ := json.Marshal(req.Tags)

	query := `
		UPDATE template_designs SET
			name = ?, description = ?, components = ?, global_styles = ?,
			variables = ?, is_template = ?, category = ?, tags = ?, 
			preview_image = ?, updated_at = ?
		WHERE id = ? AND user_id = ?`

	previewImage := s.generatePreviewImageFromComponents(req.Components, req.GlobalStyles)

	_, err := s.db.Exec(query,
		req.Name, req.Description, componentsJSON, globalStylesJSON,
		variablesJSON, req.IsTemplate, req.Category, tagsJSON,
		previewImage, time.Now(), templateID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	return s.GetTemplate(userID, templateID)
}

// DeleteTemplate deletes a template design
func (s *TemplateBuilderService) DeleteTemplate(userID, templateID uuid.UUID) error {
	query := `DELETE FROM template_designs WHERE id = ? AND user_id = ?`
	
	_, err := s.db.Exec(query, templateID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	return nil
}

// GenerateHTML generates HTML from template components
func (s *TemplateBuilderService) GenerateHTML(components []TemplateComponent, globalStyles GlobalStyles, variables map[string]interface{}) (string, error) {
	htmlBuilder := &strings.Builder{}

	// Start HTML document
	htmlBuilder.WriteString(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Email Template</title>
    <style>`)

	// Add global styles
	htmlBuilder.WriteString(s.generateGlobalCSS(globalStyles))

	// Add component-specific styles
	htmlBuilder.WriteString(s.generateComponentCSS(components))

	htmlBuilder.WriteString(`    </style>
</head>
<body>
    <div class="email-container">`)

	// Generate component HTML
	for _, component := range components {
		componentHTML := s.generateComponentHTML(component, variables)
		htmlBuilder.WriteString(componentHTML)
	}

	htmlBuilder.WriteString(`    </div>
</body>
</html>`)

	return htmlBuilder.String(), nil
}

// GeneratePreview generates a preview of the template
func (s *TemplateBuilderService) GeneratePreview(req PreviewRequest) (string, error) {
	return s.GenerateHTML(req.Components, req.GlobalStyles, req.Variables)
}

// CloneTemplate creates a copy of a template
func (s *TemplateBuilderService) CloneTemplate(userID, templateID uuid.UUID, newName string) (*TemplateDesign, error) {
	// Get original template
	original, err := s.GetTemplate(userID, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original template: %w", err)
	}

	// Create clone request
	cloneReq := BuilderRequest{
		Name:         newName,
		Description:  "Clone of " + original.Name,
		Components:   original.Components,
		GlobalStyles: original.GlobalStyles,
		Category:     original.Category,
		Tags:         original.Tags,
		IsTemplate:   original.IsTemplate,
	}

	return s.CreateTemplate(userID, cloneReq)
}

// GetAvailableComponents returns the list of available drag-and-drop components
func (s *TemplateBuilderService) GetAvailableComponents() map[string]interface{} {
	return map[string]interface{}{
		"text": map[string]interface{}{
			"name":        "Text Block",
			"description": "Rich text content with formatting options",
			"icon":        "text",
			"properties": map[string]interface{}{
				"content": map[string]interface{}{
					"type":        "richtext",
					"default":     "Enter your text here...",
					"placeholder": "Text content",
				},
				"tag": map[string]interface{}{
					"type":    "select",
					"options": []string{"p", "h1", "h2", "h3", "h4", "h5", "h6"},
					"default": "p",
				},
			},
		},
		"button": map[string]interface{}{
			"name":        "Button",
			"description": "Call-to-action button with customizable styling",
			"icon":        "cursor-click",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "text",
					"default":     "Click Here",
					"placeholder": "Button text",
				},
				"url": map[string]interface{}{
					"type":        "url",
					"default":     "#",
					"placeholder": "Button URL",
				},
				"target": map[string]interface{}{
					"type":    "select",
					"options": []string{"_self", "_blank"},
					"default": "_blank",
				},
			},
		},
		"image": map[string]interface{}{
			"name":        "Image",
			"description": "Responsive image with optional link",
			"icon":        "photograph",
			"properties": map[string]interface{}{
				"src": map[string]interface{}{
					"type":        "image",
					"placeholder": "Image URL",
				},
				"alt": map[string]interface{}{
					"type":        "text",
					"placeholder": "Alt text",
				},
				"url": map[string]interface{}{
					"type":        "url",
					"placeholder": "Link URL (optional)",
				},
			},
		},
		"divider": map[string]interface{}{
			"name":        "Divider",
			"description": "Horizontal line to separate content",
			"icon":        "minus",
			"properties": map[string]interface{}{
				"style": map[string]interface{}{
					"type":    "select",
					"options": []string{"solid", "dashed", "dotted"},
					"default": "solid",
				},
			},
		},
		"spacer": map[string]interface{}{
			"name":        "Spacer",
			"description": "Vertical spacing between elements",
			"icon":        "arrows-expand",
			"properties": map[string]interface{}{
				"height": map[string]interface{}{
					"type":    "number",
					"default": 20,
					"min":     5,
					"max":     200,
				},
			},
		},
		"social": map[string]interface{}{
			"name":        "Social Links",
			"description": "Social media icons with links",
			"icon":        "share",
			"properties": map[string]interface{}{
				"platforms": map[string]interface{}{
					"type":    "multi-select",
					"options": []string{"facebook", "twitter", "instagram", "linkedin", "youtube"},
				},
				"urls": map[string]interface{}{
					"type": "object",
				},
				"style": map[string]interface{}{
					"type":    "select",
					"options": []string{"icons", "buttons", "text"},
					"default": "icons",
				},
			},
		},
		"container": map[string]interface{}{
			"name":        "Container",
			"description": "Container for grouping other components",
			"icon":        "template",
			"properties": map[string]interface{}{
				"layout": map[string]interface{}{
					"type":    "select",
					"options": []string{"single", "two-column", "three-column"},
					"default": "single",
				},
			},
		},
		"header": map[string]interface{}{
			"name":        "Header",
			"description": "Email header with logo and navigation",
			"icon":        "menu",
			"properties": map[string]interface{}{
				"logo": map[string]interface{}{
					"type":        "image",
					"placeholder": "Logo URL",
				},
				"title": map[string]interface{}{
					"type":        "text",
					"placeholder": "Header title",
				},
			},
		},
		"footer": map[string]interface{}{
			"name":        "Footer",
			"description": "Email footer with contact info and unsubscribe",
			"icon":        "menu-alt-4",
			"properties": map[string]interface{}{
				"company_name": map[string]interface{}{
					"type":        "text",
					"placeholder": "Company name",
				},
				"address": map[string]interface{}{
					"type":        "textarea",
					"placeholder": "Company address",
				},
				"unsubscribe_url": map[string]interface{}{
					"type":        "url",
					"placeholder": "Unsubscribe URL",
				},
			},
		},
	}
}

// Helper methods

func (s *TemplateBuilderService) extractVariablesFromComponents(components []TemplateComponent) []string {
	variables := make(map[string]bool)
	
	for _, component := range components {
		s.extractVariablesFromComponent(component, variables)
	}

	var variableList []string
	for v := range variables {
		variableList = append(variableList, v)
	}

	return variableList
}

func (s *TemplateBuilderService) extractVariablesFromComponent(component TemplateComponent, variables map[string]bool) {
	// Extract variables from content
	s.extractVariablesFromString(component.Content, variables)
	
	// Extract variables from properties
	for _, prop := range component.Properties {
		if str, ok := prop.(string); ok {
			s.extractVariablesFromString(str, variables)
		}
	}

	// Extract from children
	for _, child := range component.Children {
		s.extractVariablesFromComponent(child, variables)
	}
}

func (s *TemplateBuilderService) extractVariablesFromString(content string, variables map[string]bool) {
	// Extract {{variable}} patterns
	matches := strings.Split(content, "{{")
	for i := 1; i < len(matches); i++ {
		endIdx := strings.Index(matches[i], "}}")
		if endIdx > 0 {
			variable := strings.TrimSpace(matches[i][:endIdx])
			if variable != "" {
				variables[variable] = true
			}
		}
	}
}

func (s *TemplateBuilderService) generateGlobalCSS(styles GlobalStyles) string {
	css := strings.Builder{}
	
	css.WriteString(`
        body {
            margin: 0;
            padding: 0;
            font-family: ` + styles.DefaultFontFamily + `;
            background-color: ` + styles.BackgroundColor + `;
            color: ` + styles.DefaultTextColor + `;
        }
        .email-container {
            max-width: ` + styles.ContainerWidth + `;
            margin: 0 auto;
            background-color: #ffffff;
        }
        a {
            color: ` + styles.LinkColor + `;
            text-decoration: none;
        }
        .responsive {
            width: 100%;
            height: auto;
        }
    `)

	// Add custom CSS
	if styles.CustomCSS != "" {
		css.WriteString("\n" + styles.CustomCSS + "\n")
	}

	// Add responsive rules
	if len(styles.ResponsiveRules) > 0 {
		css.WriteString("\n@media screen and (max-width: 600px) {\n")
		for selector, rules := range styles.ResponsiveRules {
			css.WriteString(selector + " { " + rules + " }\n")
		}
		css.WriteString("}\n")
	}

	return css.String()
}

func (s *TemplateBuilderService) generateComponentCSS(components []TemplateComponent) string {
	css := strings.Builder{}
	
	for _, component := range components {
		componentCSS := s.generateComponentStyles(component)
		css.WriteString(componentCSS)
	}

	return css.String()
}

func (s *TemplateBuilderService) generateComponentStyles(component TemplateComponent) string {
	if component.Styles.CustomCSS != "" {
		return "\n" + component.Styles.CustomCSS + "\n"
	}

	css := strings.Builder{}
	selector := "#" + component.ID

	css.WriteString(selector + " {\n")

	if component.Styles.BackgroundColor != "" {
		css.WriteString("  background-color: " + component.Styles.BackgroundColor + ";\n")
	}
	if component.Styles.TextColor != "" {
		css.WriteString("  color: " + component.Styles.TextColor + ";\n")
	}
	if component.Styles.FontFamily != "" {
		css.WriteString("  font-family: " + component.Styles.FontFamily + ";\n")
	}
	if component.Styles.FontSize != "" {
		css.WriteString("  font-size: " + component.Styles.FontSize + ";\n")
	}
	if component.Styles.FontWeight != "" {
		css.WriteString("  font-weight: " + component.Styles.FontWeight + ";\n")
	}
	if component.Styles.TextAlign != "" {
		css.WriteString("  text-align: " + component.Styles.TextAlign + ";\n")
	}
	if component.Styles.Padding != "" {
		css.WriteString("  padding: " + component.Styles.Padding + ";\n")
	}
	if component.Styles.Margin != "" {
		css.WriteString("  margin: " + component.Styles.Margin + ";\n")
	}
	if component.Styles.BorderRadius != "" {
		css.WriteString("  border-radius: " + component.Styles.BorderRadius + ";\n")
	}
	if component.Styles.Border != "" {
		css.WriteString("  border: " + component.Styles.Border + ";\n")
	}
	if component.Styles.Width != "" {
		css.WriteString("  width: " + component.Styles.Width + ";\n")
	}
	if component.Styles.Height != "" {
		css.WriteString("  height: " + component.Styles.Height + ";\n")
	}

	// Add custom styles
	for property, value := range component.Styles.CustomStyles {
		css.WriteString("  " + property + ": " + value + ";\n")
	}

	css.WriteString("}\n")

	return css.String()
}

func (s *TemplateBuilderService) generateComponentHTML(component TemplateComponent, variables map[string]interface{}) string {
	html := strings.Builder{}
	
	switch component.Type {
	case "text":
		html.WriteString(s.generateTextHTML(component, variables))
	case "button":
		html.WriteString(s.generateButtonHTML(component, variables))
	case "image":
		html.WriteString(s.generateImageHTML(component, variables))
	case "divider":
		html.WriteString(s.generateDividerHTML(component))
	case "spacer":
		html.WriteString(s.generateSpacerHTML(component))
	case "social":
		html.WriteString(s.generateSocialHTML(component, variables))
	case "container":
		html.WriteString(s.generateContainerHTML(component, variables))
	case "header":
		html.WriteString(s.generateHeaderHTML(component, variables))
	case "footer":
		html.WriteString(s.generateFooterHTML(component, variables))
	default:
		// Generic component
		html.WriteString(`<div id="` + component.ID + `">`)
		html.WriteString(s.replaceVariables(component.Content, variables))
		html.WriteString(`</div>`)
	}

	return html.String()
}

func (s *TemplateBuilderService) generateTextHTML(component TemplateComponent, variables map[string]interface{}) string {
	tag := "p"
	if tagVal, ok := component.Properties["tag"].(string); ok {
		tag = tagVal
	}

	content := s.replaceVariables(component.Content, variables)
	return fmt.Sprintf(`<%s id="%s">%s</%s>`, tag, component.ID, content, tag)
}

func (s *TemplateBuilderService) generateButtonHTML(component TemplateComponent, variables map[string]interface{}) string {
	text := s.replaceVariables(component.Content, variables)
	url := "#"
	target := "_blank"

	if urlVal, ok := component.Properties["url"].(string); ok {
		url = s.replaceVariables(urlVal, variables)
	}
	if targetVal, ok := component.Properties["target"].(string); ok {
		target = targetVal
	}

	return fmt.Sprintf(`<div id="%s"><a href="%s" target="%s" style="display: inline-block; padding: 12px 24px; text-decoration: none;">%s</a></div>`, 
		component.ID, url, target, text)
}

func (s *TemplateBuilderService) generateImageHTML(component TemplateComponent, variables map[string]interface{}) string {
	src := ""
	alt := ""
	url := ""

	if srcVal, ok := component.Properties["src"].(string); ok {
		src = s.replaceVariables(srcVal, variables)
	}
	if altVal, ok := component.Properties["alt"].(string); ok {
		alt = s.replaceVariables(altVal, variables)
	}
	if urlVal, ok := component.Properties["url"].(string); ok {
		url = s.replaceVariables(urlVal, variables)
	}

	imgTag := fmt.Sprintf(`<img src="%s" alt="%s" class="responsive">`, src, alt)
	
	if url != "" {
		return fmt.Sprintf(`<div id="%s"><a href="%s">%s</a></div>`, component.ID, url, imgTag)
	}
	
	return fmt.Sprintf(`<div id="%s">%s</div>`, component.ID, imgTag)
}

func (s *TemplateBuilderService) generateDividerHTML(component TemplateComponent) string {
	style := "solid"
	if styleVal, ok := component.Properties["style"].(string); ok {
		style = styleVal
	}

	return fmt.Sprintf(`<div id="%s"><hr style="border: none; border-top: 1px %s #cccccc; margin: 20px 0;"></div>`, 
		component.ID, style)
}

func (s *TemplateBuilderService) generateSpacerHTML(component TemplateComponent) string {
	height := "20"
	if heightVal, ok := component.Properties["height"]; ok {
		height = fmt.Sprintf("%v", heightVal)
	}

	return fmt.Sprintf(`<div id="%s" style="height: %spx;"></div>`, component.ID, height)
}

func (s *TemplateBuilderService) generateSocialHTML(component TemplateComponent, variables map[string]interface{}) string {
	// Simplified social HTML generation
	return fmt.Sprintf(`<div id="%s" class="social-links">Social links placeholder</div>`, component.ID)
}

func (s *TemplateBuilderService) generateContainerHTML(component TemplateComponent, variables map[string]interface{}) string {
	html := strings.Builder{}
	html.WriteString(`<div id="` + component.ID + `">`)
	
	for _, child := range component.Children {
		html.WriteString(s.generateComponentHTML(child, variables))
	}
	
	html.WriteString(`</div>`)
	return html.String()
}

func (s *TemplateBuilderService) generateHeaderHTML(component TemplateComponent, variables map[string]interface{}) string {
	title := s.replaceVariables(component.Content, variables)
	logo := ""
	
	if logoVal, ok := component.Properties["logo"].(string); ok {
		logo = s.replaceVariables(logoVal, variables)
	}

	html := fmt.Sprintf(`<div id="%s" class="header">`, component.ID)
	if logo != "" {
		html += fmt.Sprintf(`<img src="%s" alt="Logo" style="height: 40px;">`, logo)
	}
	if title != "" {
		html += fmt.Sprintf(`<h1>%s</h1>`, title)
	}
	html += `</div>`
	
	return html
}

func (s *TemplateBuilderService) generateFooterHTML(component TemplateComponent, variables map[string]interface{}) string {
	companyName := ""
	address := ""
	unsubscribeURL := ""

	if nameVal, ok := component.Properties["company_name"].(string); ok {
		companyName = s.replaceVariables(nameVal, variables)
	}
	if addrVal, ok := component.Properties["address"].(string); ok {
		address = s.replaceVariables(addrVal, variables)
	}
	if unsubVal, ok := component.Properties["unsubscribe_url"].(string); ok {
		unsubscribeURL = s.replaceVariables(unsubVal, variables)
	}

	html := fmt.Sprintf(`<div id="%s" class="footer">`, component.ID)
	if companyName != "" {
		html += fmt.Sprintf(`<p><strong>%s</strong></p>`, companyName)
	}
	if address != "" {
		html += fmt.Sprintf(`<p>%s</p>`, address)
	}
	if unsubscribeURL != "" {
		html += fmt.Sprintf(`<p><a href="%s">Unsubscribe</a></p>`, unsubscribeURL)
	}
	html += `</div>`
	
	return html
}

func (s *TemplateBuilderService) replaceVariables(content string, variables map[string]interface{}) string {
	for key, value := range variables {
		placeholder := "{{" + key + "}}"
		replacement := fmt.Sprintf("%v", value)
		content = strings.ReplaceAll(content, placeholder, replacement)
	}
	return content
}

func (s *TemplateBuilderService) generatePreviewImage(template *TemplateDesign) string {
	// In a real implementation, this would generate a screenshot of the email
	// For now, return a placeholder
	return fmt.Sprintf("/api/templates/%s/preview.png", template.ID.String())
}

func (s *TemplateBuilderService) generatePreviewImageFromComponents(components []TemplateComponent, styles GlobalStyles) string {
	// In a real implementation, this would generate a screenshot
	return "/api/templates/preview.png"
}