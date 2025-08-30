// Form Builder Types
export interface FormField {
  id: string;
  type: FieldType;
  label: string;
  name: string;
  placeholder?: string;
  description?: string;
  required: boolean;
  validation?: ValidationRule[];
  options?: FieldOption[];
  styling?: FieldStyling;
  conditionalLogic?: ConditionalLogic;
  position?: {
    x: number;
    y: number;
  };
  size?: {
    width: number;
    height?: number;
  };
}

export type FieldType =
  | 'text'
  | 'email'
  | 'password'
  | 'textarea'
  | 'number'
  | 'phone'
  | 'url'
  | 'date'
  | 'datetime-local'
  | 'time'
  | 'select'
  | 'radio'
  | 'checkbox'
  | 'file'
  | 'image'
  | 'range'
  | 'color'
  | 'rating'
  | 'signature'
  | 'divider'
  | 'heading'
  | 'paragraph'
  | 'html';

export interface FieldOption {
  id: string;
  label: string;
  value: string;
  selected?: boolean;
}

export interface ValidationRule {
  type: 'required' | 'minLength' | 'maxLength' | 'pattern' | 'min' | 'max' | 'email' | 'url' | 'custom';
  value?: string | number;
  message: string;
}

export interface FieldStyling {
  backgroundColor?: string;
  textColor?: string;
  borderColor?: string;
  borderWidth?: string;
  borderRadius?: string;
  fontSize?: string;
  fontWeight?: string;
  padding?: string;
  margin?: string;
  width?: string;
  className?: string;
  customCSS?: string;
}

export interface ConditionalLogic {
  show: boolean;
  conditions: LogicCondition[];
  logic: 'and' | 'or';
}

export interface LogicCondition {
  fieldId: string;
  operator: '=' | '!=' | '>' | '<' | '>=' | '<=' | 'contains' | 'not_contains' | 'empty' | 'not_empty';
  value: string;
}

export interface FormStep {
  id: string;
  title: string;
  description?: string;
  fields: string[]; // Array of field IDs
  showProgress?: boolean;
  buttonText?: {
    previous?: string;
    next?: string;
    submit?: string;
  };
}

export interface FormBuilder {
  id: string;
  name: string;
  description?: string;
  fields: FormField[];
  steps?: FormStep[];
  isMultiStep: boolean;
  settings: FormSettings;
  styling: FormStyling;
  notifications: FormNotifications;
  integrations: FormIntegrations;
  analytics?: FormAnalytics;
  createdAt: Date;
  updatedAt: Date;
}

export interface FormSettings {
  title: string;
  description?: string;
  submitButtonText: string;
  successMessage: string;
  errorMessage: string;
  redirectUrl?: string;
  allowMultipleSubmissions: boolean;
  requireAuthentication: boolean;
  collectIP: boolean;
  collectUserAgent: boolean;
  enableSpamProtection: boolean;
  enableCaptcha: boolean;
  maxSubmissions?: number;
  submissionDeadline?: Date;
  enableSaveAndContinue: boolean;
}

export interface FormStyling {
  theme: 'default' | 'minimal' | 'modern' | 'classic' | 'custom';
  primaryColor: string;
  secondaryColor: string;
  backgroundColor: string;
  textColor: string;
  font: string;
  spacing: 'tight' | 'normal' | 'relaxed';
  borderRadius: 'none' | 'sm' | 'md' | 'lg' | 'xl';
  shadow: 'none' | 'sm' | 'md' | 'lg' | 'xl';
  customCSS?: string;
  responsive: {
    mobile: ResponsiveStyling;
    tablet: ResponsiveStyling;
    desktop: ResponsiveStyling;
  };
}

export interface ResponsiveStyling {
  columns: 1 | 2 | 3 | 4;
  spacing: string;
  fontSize: string;
  padding: string;
}

export interface FormNotifications {
  email: {
    enabled: boolean;
    to: string[];
    cc?: string[];
    bcc?: string[];
    subject: string;
    template: string;
    includeAttachments: boolean;
  };
  webhook: {
    enabled: boolean;
    url: string;
    method: 'POST' | 'PUT';
    headers?: Record<string, string>;
    authentication?: {
      type: 'none' | 'basic' | 'bearer' | 'api_key';
      credentials?: Record<string, string>;
    };
  };
  autoresponder: {
    enabled: boolean;
    emailField: string;
    subject: string;
    template: string;
    delay?: number;
  };
}

export interface FormIntegrations {
  zapier?: {
    enabled: boolean;
    webhookUrl: string;
  };
  googleSheets?: {
    enabled: boolean;
    spreadsheetId: string;
    worksheetName: string;
  };
  hubspot?: {
    enabled: boolean;
    apiKey: string;
    portalId: string;
  };
  mailchimp?: {
    enabled: boolean;
    apiKey: string;
    listId: string;
  };
  slack?: {
    enabled: boolean;
    webhookUrl: string;
    channel: string;
  };
}

export interface FormAnalytics {
  enabled: boolean;
  trackViews: boolean;
  trackCompletions: boolean;
  trackConversions: boolean;
  trackFieldInteractions: boolean;
  heatmap: boolean;
  abTesting: boolean;
}

// Form Builder UI States
export interface FormBuilderState {
  selectedFieldId?: string;
  activeStep: number;
  isDragging: boolean;
  isPreviewMode: boolean;
  viewMode: 'desktop' | 'tablet' | 'mobile';
  showGrid: boolean;
  snapToGrid: boolean;
  zoom: number;
  history: FormBuilderHistory;
}

export interface FormBuilderHistory {
  past: FormBuilder[];
  present: FormBuilder;
  future: FormBuilder[];
}

// Component Palette
export interface PaletteComponent {
  id: string;
  type: FieldType;
  name: string;
  description: string;
  icon: React.ReactNode;
  category: ComponentCategory;
  preview: React.ReactNode;
  defaultProps: Partial<FormField>;
}

export type ComponentCategory = 
  | 'input'
  | 'choice'
  | 'layout'
  | 'media'
  | 'advanced';

// Form Templates
export interface FormTemplate {
  id: string;
  name: string;
  description: string;
  category: TemplateCategory;
  preview: string;
  fields: FormField[];
  settings: FormSettings;
  styling: FormStyling;
  tags: string[];
  popularity: number;
  isPremium: boolean;
}

export type TemplateCategory =
  | 'contact'
  | 'registration'
  | 'survey'
  | 'order'
  | 'application'
  | 'feedback'
  | 'newsletter'
  | 'event'
  | 'support'
  | 'custom';

// Drag and Drop
export interface DragDropContext {
  draggedItem?: PaletteComponent | FormField;
  dropTarget?: string;
  dragOverlay?: React.ReactNode;
}

// Form Validation Results
export interface ValidationResult {
  isValid: boolean;
  errors: ValidationError[];
  warnings: ValidationWarning[];
}

export interface ValidationError {
  fieldId: string;
  message: string;
  type: 'required' | 'format' | 'logic' | 'constraint';
}

export interface ValidationWarning {
  fieldId: string;
  message: string;
  type: 'accessibility' | 'ux' | 'performance';
}

// Form Submission
export interface FormSubmission {
  id: string;
  formId: string;
  data: Record<string, any>;
  files: FileSubmission[];
  metadata: SubmissionMetadata;
  status: 'pending' | 'processed' | 'failed';
  submittedAt: Date;
}

export interface FileSubmission {
  fieldId: string;
  filename: string;
  url: string;
  size: number;
  type: string;
}

export interface SubmissionMetadata {
  ip?: string;
  userAgent?: string;
  location?: {
    country: string;
    city: string;
    coordinates?: [number, number];
  };
  duration: number;
  source: string;
}

// Export/Import
export interface FormExport {
  version: string;
  form: FormBuilder;
  exportedAt: Date;
  exportedBy: string;
}

export interface FormImportResult {
  success: boolean;
  form?: FormBuilder;
  errors: string[];
  warnings: string[];
}