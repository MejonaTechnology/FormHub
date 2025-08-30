import { FormField, FormBuilder, ValidationResult, ValidationError, ValidationWarning } from '@/types/form-builder';

/**
 * Generate a unique field ID
 */
export const generateFieldId = (): string => {
  return `field_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
};

/**
 * Generate a unique form ID
 */
export const generateFormId = (): string => {
  return `form_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
};

/**
 * Validate a form field
 */
export const validateField = (field: FormField, value: any): ValidationResult => {
  const errors: ValidationError[] = [];
  const warnings: ValidationWarning[] = [];

  // Required field validation
  if (field.required && (!value || value === '' || (Array.isArray(value) && value.length === 0))) {
    errors.push({
      fieldId: field.id,
      message: `${field.label} is required`,
      type: 'required'
    });
  }

  // Type-specific validation
  if (value && value !== '') {
    // Email validation
    if (field.type === 'email') {
      const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
      if (!emailRegex.test(value)) {
        errors.push({
          fieldId: field.id,
          message: 'Please enter a valid email address',
          type: 'format'
        });
      }
    }

    // URL validation
    if (field.type === 'url') {
      try {
        new URL(value);
      } catch {
        errors.push({
          fieldId: field.id,
          message: 'Please enter a valid URL',
          type: 'format'
        });
      }
    }

    // Phone validation
    if (field.type === 'phone') {
      const phoneRegex = /^[\+]?[1-9][\d]{0,15}$/;
      if (!phoneRegex.test(value.replace(/\D/g, ''))) {
        errors.push({
          fieldId: field.id,
          message: 'Please enter a valid phone number',
          type: 'format'
        });
      }
    }

    // Number validation
    if (field.type === 'number') {
      const numValue = Number(value);
      if (isNaN(numValue)) {
        errors.push({
          fieldId: field.id,
          message: 'Please enter a valid number',
          type: 'format'
        });
      }
    }
  }

  // Custom validation rules
  if (field.validation && value) {
    field.validation.forEach(rule => {
      switch (rule.type) {
        case 'minLength':
          if (typeof value === 'string' && value.length < Number(rule.value)) {
            errors.push({
              fieldId: field.id,
              message: rule.message || `Must be at least ${rule.value} characters`,
              type: 'constraint'
            });
          }
          break;

        case 'maxLength':
          if (typeof value === 'string' && value.length > Number(rule.value)) {
            errors.push({
              fieldId: field.id,
              message: rule.message || `Must be no more than ${rule.value} characters`,
              type: 'constraint'
            });
          }
          break;

        case 'min':
          if (Number(value) < Number(rule.value)) {
            errors.push({
              fieldId: field.id,
              message: rule.message || `Must be at least ${rule.value}`,
              type: 'constraint'
            });
          }
          break;

        case 'max':
          if (Number(value) > Number(rule.value)) {
            errors.push({
              fieldId: field.id,
              message: rule.message || `Must be no more than ${rule.value}`,
              type: 'constraint'
            });
          }
          break;

        case 'pattern':
          if (rule.value && !new RegExp(rule.value as string).test(value)) {
            errors.push({
              fieldId: field.id,
              message: rule.message || 'Invalid format',
              type: 'format'
            });
          }
          break;
      }
    });
  }

  // Accessibility warnings
  if (!field.label) {
    warnings.push({
      fieldId: field.id,
      message: 'Field should have a label for accessibility',
      type: 'accessibility'
    });
  }

  return {
    isValid: errors.length === 0,
    errors,
    warnings
  };
};

/**
 * Validate an entire form
 */
export const validateForm = (form: FormBuilder, formData: Record<string, any>): ValidationResult => {
  const allErrors: ValidationError[] = [];
  const allWarnings: ValidationWarning[] = [];

  // Validate each field
  form.fields.forEach(field => {
    const fieldResult = validateField(field, formData[field.id]);
    allErrors.push(...fieldResult.errors);
    allWarnings.push(...fieldResult.warnings);
  });

  // Form-level validation
  if (form.fields.length === 0) {
    allErrors.push({
      fieldId: 'form',
      message: 'Form must have at least one field',
      type: 'constraint'
    });
  }

  // Check for duplicate field names
  const fieldNames = form.fields.map(field => field.name);
  const duplicateNames = fieldNames.filter((name, index) => fieldNames.indexOf(name) !== index);
  duplicateNames.forEach(name => {
    const duplicateFields = form.fields.filter(field => field.name === name);
    duplicateFields.forEach(field => {
      allErrors.push({
        fieldId: field.id,
        message: `Field name "${name}" is used by multiple fields`,
        type: 'constraint'
      });
    });
  });

  return {
    isValid: allErrors.length === 0,
    errors: allErrors,
    warnings: allWarnings
  };
};

/**
 * Convert form data to submission format
 */
export const formatSubmissionData = (form: FormBuilder, formData: Record<string, any>) => {
  const submission: any = {};
  
  form.fields.forEach(field => {
    const value = formData[field.id];
    if (value !== undefined) {
      submission[field.name] = value;
    }
  });

  return submission;
};

/**
 * Generate form HTML
 */
export const generateFormHTML = (form: FormBuilder): string => {
  const generateFieldHTML = (field: FormField): string => {
    const commonAttrs = `
      id="${field.id}"
      name="${field.name}"
      ${field.required ? 'required' : ''}
      ${field.placeholder ? `placeholder="${field.placeholder}"` : ''}
    `.trim();

    let fieldHTML = '';

    switch (field.type) {
      case 'text':
      case 'email':
      case 'password':
      case 'url':
      case 'phone':
      case 'date':
      case 'time':
      case 'datetime-local':
      case 'number':
        fieldHTML = `<input type="${field.type}" ${commonAttrs} class="form-input" />`;
        break;

      case 'textarea':
        fieldHTML = `<textarea ${commonAttrs} rows="4" class="form-textarea"></textarea>`;
        break;

      case 'select':
        const options = field.options?.map(option => 
          `<option value="${option.value}">${option.label}</option>`
        ).join('') || '';
        fieldHTML = `<select ${commonAttrs} class="form-select">${options}</select>`;
        break;

      case 'radio':
        fieldHTML = field.options?.map(option => `
          <label class="radio-label">
            <input type="radio" name="${field.name}" value="${option.value}" class="form-radio" />
            <span>${option.label}</span>
          </label>
        `).join('') || '';
        break;

      case 'checkbox':
        fieldHTML = field.options?.map(option => `
          <label class="checkbox-label">
            <input type="checkbox" name="${field.name}[]" value="${option.value}" class="form-checkbox" />
            <span>${option.label}</span>
          </label>
        `).join('') || '';
        break;

      case 'file':
      case 'image':
        fieldHTML = `<input type="file" ${commonAttrs} ${field.type === 'image' ? 'accept="image/*"' : ''} class="form-file" />`;
        break;

      case 'heading':
        fieldHTML = `<h3 class="form-heading">${field.label}</h3>`;
        break;

      case 'paragraph':
        fieldHTML = `<p class="form-paragraph">${field.description || field.label}</p>`;
        break;

      case 'divider':
        fieldHTML = `<hr class="form-divider" />`;
        break;

      default:
        fieldHTML = `<!-- Unsupported field type: ${field.type} -->`;
    }

    if (['heading', 'paragraph', 'divider'].includes(field.type)) {
      return fieldHTML;
    }

    return `
      <div class="form-field">
        ${field.label ? `<label for="${field.id}" class="form-label">${field.label}${field.required ? '<span class="required">*</span>' : ''}</label>` : ''}
        ${field.description ? `<p class="form-description">${field.description}</p>` : ''}
        ${fieldHTML}
      </div>
    `;
  };

  const fieldsHTML = form.fields.map(generateFieldHTML).join('\n');

  return `
    <form class="formhub-form" method="POST" action="https://formhub.mejona.in/api/v1/submit">
      <input type="hidden" name="access_key" value="YOUR_API_KEY_HERE" />
      
      ${form.settings.title ? `<h2 class="form-title">${form.settings.title}</h2>` : ''}
      ${form.settings.description ? `<p class="form-description">${form.settings.description}</p>` : ''}
      
      ${fieldsHTML}
      
      <button type="submit" class="form-submit">${form.settings.submitButtonText}</button>
    </form>
  `;
};

/**
 * Generate CSS styles for a form
 */
export const generateFormCSS = (form: FormBuilder): string => {
  const styling = form.styling;
  
  return `
    .formhub-form {
      max-width: 600px;
      margin: 0 auto;
      padding: 2rem;
      font-family: ${styling.font}, -apple-system, BlinkMacSystemFont, sans-serif;
      background-color: ${styling.backgroundColor};
      color: ${styling.textColor};
      border-radius: ${styling.borderRadius === 'none' ? '0' : 
                      styling.borderRadius === 'sm' ? '0.375rem' :
                      styling.borderRadius === 'md' ? '0.5rem' :
                      styling.borderRadius === 'lg' ? '0.75rem' : '1rem'};
      box-shadow: ${styling.shadow === 'none' ? 'none' :
                  styling.shadow === 'sm' ? '0 1px 2px 0 rgba(0, 0, 0, 0.05)' :
                  styling.shadow === 'md' ? '0 4px 6px -1px rgba(0, 0, 0, 0.1)' :
                  styling.shadow === 'lg' ? '0 10px 15px -3px rgba(0, 0, 0, 0.1)' :
                  '0 20px 25px -5px rgba(0, 0, 0, 0.1)'};
    }
    
    .form-title {
      font-size: 1.875rem;
      font-weight: 700;
      margin-bottom: 0.5rem;
      color: ${styling.primaryColor};
    }
    
    .form-description {
      font-size: 1rem;
      margin-bottom: 2rem;
      color: ${styling.textColor}CC;
    }
    
    .form-field {
      margin-bottom: ${styling.spacing === 'tight' ? '1rem' :
                     styling.spacing === 'normal' ? '1.5rem' : '2rem'};
    }
    
    .form-label {
      display: block;
      font-size: 0.875rem;
      font-weight: 500;
      margin-bottom: 0.25rem;
    }
    
    .required {
      color: #ef4444;
    }
    
    .form-input,
    .form-textarea,
    .form-select {
      width: 100%;
      padding: 0.75rem;
      border: 1px solid #d1d5db;
      border-radius: ${styling.borderRadius === 'none' ? '0' : 
                      styling.borderRadius === 'sm' ? '0.375rem' : '0.5rem'};
      font-size: 1rem;
      transition: border-color 0.15s ease-in-out, box-shadow 0.15s ease-in-out;
    }
    
    .form-input:focus,
    .form-textarea:focus,
    .form-select:focus {
      outline: none;
      border-color: ${styling.primaryColor};
      box-shadow: 0 0 0 3px ${styling.primaryColor}33;
    }
    
    .form-submit {
      background-color: ${styling.primaryColor};
      color: white;
      padding: 0.75rem 2rem;
      border: none;
      border-radius: ${styling.borderRadius === 'none' ? '0' : 
                      styling.borderRadius === 'sm' ? '0.375rem' : '0.5rem'};
      font-size: 1rem;
      font-weight: 500;
      cursor: pointer;
      transition: background-color 0.15s ease-in-out;
      width: 100%;
    }
    
    .form-submit:hover {
      background-color: ${styling.primaryColor}E6;
    }
    
    .radio-label,
    .checkbox-label {
      display: flex;
      align-items: center;
      margin-bottom: 0.5rem;
      cursor: pointer;
    }
    
    .form-radio,
    .form-checkbox {
      margin-right: 0.5rem;
    }
    
    .form-heading {
      font-size: 1.25rem;
      font-weight: 600;
      margin-bottom: 1rem;
      color: ${styling.primaryColor};
    }
    
    .form-paragraph {
      margin-bottom: 1rem;
      line-height: 1.5;
    }
    
    .form-divider {
      border: none;
      border-top: 1px solid #e5e7eb;
      margin: 2rem 0;
    }
    
    @media (max-width: 640px) {
      .formhub-form {
        padding: 1rem;
      }
      
      .form-field {
        margin-bottom: 1rem;
      }
    }
    
    ${styling.customCSS || ''}
  `;
};

/**
 * Export form configuration as JSON
 */
export const exportFormConfig = (form: FormBuilder) => {
  return {
    version: '1.0.0',
    form,
    exportedAt: new Date().toISOString(),
    exportedBy: 'FormHub Builder'
  };
};

/**
 * Import form configuration from JSON
 */
export const importFormConfig = (configData: any): { success: boolean; form?: FormBuilder; errors: string[] } => {
  try {
    if (!configData.form) {
      return { success: false, errors: ['Invalid configuration: missing form data'] };
    }

    const form = configData.form as FormBuilder;
    const errors: string[] = [];

    // Validate required fields
    if (!form.name) errors.push('Form name is required');
    if (!form.settings) errors.push('Form settings are required');
    if (!form.styling) errors.push('Form styling is required');
    if (!form.notifications) errors.push('Form notifications are required');

    if (errors.length > 0) {
      return { success: false, errors };
    }

    // Generate new IDs
    form.id = generateFormId();
    form.fields = form.fields.map(field => ({
      ...field,
      id: generateFieldId()
    }));

    return { success: true, form, errors: [] };
  } catch (error) {
    return { success: false, errors: ['Failed to parse configuration file'] };
  }
};