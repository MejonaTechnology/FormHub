import { FormTemplate, FormField } from '@/types/form-builder';

/**
 * Default form templates
 */
export const formTemplates: FormTemplate[] = [
  {
    id: 'contact-form',
    name: 'Contact Form',
    description: 'A simple contact form with name, email, and message fields',
    category: 'contact',
    preview: 'contact-form-preview.jpg',
    popularity: 95,
    isPremium: false,
    tags: ['contact', 'simple', 'business'],
    fields: [
      {
        id: 'name',
        type: 'text',
        label: 'Full Name',
        name: 'name',
        placeholder: 'Enter your full name',
        required: true,
        validation: [
          {
            type: 'required',
            message: 'Name is required'
          },
          {
            type: 'minLength',
            value: '2',
            message: 'Name must be at least 2 characters'
          }
        ]
      },
      {
        id: 'email',
        type: 'email',
        label: 'Email Address',
        name: 'email',
        placeholder: 'your@email.com',
        required: true,
        validation: [
          {
            type: 'required',
            message: 'Email is required'
          },
          {
            type: 'email',
            message: 'Please enter a valid email address'
          }
        ]
      },
      {
        id: 'phone',
        type: 'phone',
        label: 'Phone Number',
        name: 'phone',
        placeholder: '+1 (555) 123-4567',
        required: false
      },
      {
        id: 'subject',
        type: 'select',
        label: 'Subject',
        name: 'subject',
        placeholder: 'Select a subject',
        required: true,
        options: [
          { id: '1', label: 'General Inquiry', value: 'general' },
          { id: '2', label: 'Support Request', value: 'support' },
          { id: '3', label: 'Sales Question', value: 'sales' },
          { id: '4', label: 'Partnership', value: 'partnership' },
          { id: '5', label: 'Other', value: 'other' }
        ]
      },
      {
        id: 'message',
        type: 'textarea',
        label: 'Message',
        name: 'message',
        placeholder: 'Please describe your inquiry in detail...',
        required: true,
        validation: [
          {
            type: 'required',
            message: 'Message is required'
          },
          {
            type: 'minLength',
            value: '10',
            message: 'Message must be at least 10 characters'
          }
        ]
      }
    ],
    settings: {
      title: 'Get in Touch',
      description: 'We\'d love to hear from you. Send us a message and we\'ll respond as soon as possible.',
      submitButtonText: 'Send Message',
      successMessage: 'Thank you for your message! We\'ll get back to you soon.',
      errorMessage: 'There was an error sending your message. Please try again.',
      allowMultipleSubmissions: true,
      requireAuthentication: false,
      collectIP: false,
      collectUserAgent: false,
      enableSpamProtection: true,
      enableCaptcha: false,
      enableSaveAndContinue: false,
    },
    styling: {
      theme: 'modern',
      primaryColor: '#3b82f6',
      secondaryColor: '#64748b',
      backgroundColor: '#ffffff',
      textColor: '#1f2937',
      font: 'Inter',
      spacing: 'normal',
      borderRadius: 'md',
      shadow: 'md',
      responsive: {
        mobile: { columns: 1, spacing: '1rem', fontSize: '14px', padding: '1rem' },
        tablet: { columns: 1, spacing: '1.5rem', fontSize: '16px', padding: '2rem' },
        desktop: { columns: 1, spacing: '2rem', fontSize: '16px', padding: '2rem' },
      },
    }
  },

  {
    id: 'newsletter-signup',
    name: 'Newsletter Signup',
    description: 'Simple email collection form for newsletter subscriptions',
    category: 'newsletter',
    preview: 'newsletter-signup-preview.jpg',
    popularity: 88,
    isPremium: false,
    tags: ['newsletter', 'email', 'marketing', 'simple'],
    fields: [
      {
        id: 'email',
        type: 'email',
        label: 'Email Address',
        name: 'email',
        placeholder: 'Enter your email address',
        required: true,
        validation: [
          {
            type: 'required',
            message: 'Email is required'
          },
          {
            type: 'email',
            message: 'Please enter a valid email address'
          }
        ]
      },
      {
        id: 'interests',
        type: 'checkbox',
        label: 'I\'m interested in:',
        name: 'interests',
        required: false,
        options: [
          { id: '1', label: 'Product Updates', value: 'product_updates' },
          { id: '2', label: 'Company News', value: 'company_news' },
          { id: '3', label: 'Industry Insights', value: 'industry_insights' },
          { id: '4', label: 'Special Offers', value: 'special_offers' }
        ]
      },
      {
        id: 'consent',
        type: 'checkbox',
        label: '',
        name: 'consent',
        required: true,
        options: [
          { 
            id: '1', 
            label: 'I agree to receive marketing communications and understand I can unsubscribe at any time.', 
            value: 'agreed' 
          }
        ]
      }
    ],
    settings: {
      title: 'Stay Updated',
      description: 'Subscribe to our newsletter and never miss important updates.',
      submitButtonText: 'Subscribe',
      successMessage: 'Welcome! You\'ve successfully subscribed to our newsletter.',
      errorMessage: 'Subscription failed. Please try again.',
      allowMultipleSubmissions: false,
      requireAuthentication: false,
      collectIP: true,
      collectUserAgent: false,
      enableSpamProtection: true,
      enableCaptcha: false,
      enableSaveAndContinue: false,
    },
    styling: {
      theme: 'minimal',
      primaryColor: '#10b981',
      secondaryColor: '#6b7280',
      backgroundColor: '#f9fafb',
      textColor: '#111827',
      font: 'Inter',
      spacing: 'normal',
      borderRadius: 'lg',
      shadow: 'sm',
      responsive: {
        mobile: { columns: 1, spacing: '1rem', fontSize: '14px', padding: '1.5rem' },
        tablet: { columns: 1, spacing: '1.5rem', fontSize: '16px', padding: '2rem' },
        desktop: { columns: 1, spacing: '1.5rem', fontSize: '16px', padding: '2rem' },
      },
    }
  },

  {
    id: 'event-registration',
    name: 'Event Registration',
    description: 'Complete event registration form with attendee details',
    category: 'event',
    preview: 'event-registration-preview.jpg',
    popularity: 75,
    isPremium: false,
    tags: ['event', 'registration', 'attendee', 'business'],
    fields: [
      {
        id: 'heading',
        type: 'heading',
        label: 'Event Registration',
        name: 'heading',
        required: false
      },
      {
        id: 'first_name',
        type: 'text',
        label: 'First Name',
        name: 'first_name',
        placeholder: 'Enter your first name',
        required: true
      },
      {
        id: 'last_name',
        type: 'text',
        label: 'Last Name',
        name: 'last_name',
        placeholder: 'Enter your last name',
        required: true
      },
      {
        id: 'email',
        type: 'email',
        label: 'Email Address',
        name: 'email',
        placeholder: 'your@email.com',
        required: true
      },
      {
        id: 'company',
        type: 'text',
        label: 'Company/Organization',
        name: 'company',
        placeholder: 'Your company name',
        required: false
      },
      {
        id: 'job_title',
        type: 'text',
        label: 'Job Title',
        name: 'job_title',
        placeholder: 'Your job title',
        required: false
      },
      {
        id: 'attendance_type',
        type: 'radio',
        label: 'Attendance Type',
        name: 'attendance_type',
        required: true,
        options: [
          { id: '1', label: 'In-Person', value: 'in_person' },
          { id: '2', label: 'Virtual', value: 'virtual' }
        ]
      },
      {
        id: 'dietary_requirements',
        type: 'select',
        label: 'Dietary Requirements',
        name: 'dietary_requirements',
        placeholder: 'Select if applicable',
        required: false,
        options: [
          { id: '1', label: 'None', value: 'none' },
          { id: '2', label: 'Vegetarian', value: 'vegetarian' },
          { id: '3', label: 'Vegan', value: 'vegan' },
          { id: '4', label: 'Gluten-Free', value: 'gluten_free' },
          { id: '5', label: 'Other (please specify in comments)', value: 'other' }
        ]
      },
      {
        id: 'comments',
        type: 'textarea',
        label: 'Additional Comments',
        name: 'comments',
        placeholder: 'Any special requirements or comments...',
        required: false
      }
    ],
    settings: {
      title: 'Register for Our Event',
      description: 'Please fill out the form below to register for the upcoming event.',
      submitButtonText: 'Register Now',
      successMessage: 'Registration successful! You\'ll receive a confirmation email shortly.',
      errorMessage: 'Registration failed. Please check your information and try again.',
      allowMultipleSubmissions: false,
      requireAuthentication: false,
      collectIP: true,
      collectUserAgent: true,
      enableSpamProtection: true,
      enableCaptcha: false,
      enableSaveAndContinue: true,
    },
    styling: {
      theme: 'modern',
      primaryColor: '#8b5cf6',
      secondaryColor: '#64748b',
      backgroundColor: '#ffffff',
      textColor: '#1f2937',
      font: 'Inter',
      spacing: 'normal',
      borderRadius: 'md',
      shadow: 'lg',
      responsive: {
        mobile: { columns: 1, spacing: '1rem', fontSize: '14px', padding: '1rem' },
        tablet: { columns: 2, spacing: '1.5rem', fontSize: '16px', padding: '2rem' },
        desktop: { columns: 2, spacing: '2rem', fontSize: '16px', padding: '2rem' },
      },
    }
  },

  {
    id: 'job-application',
    name: 'Job Application',
    description: 'Comprehensive job application form with file upload',
    category: 'application',
    preview: 'job-application-preview.jpg',
    popularity: 65,
    isPremium: false,
    tags: ['job', 'application', 'hr', 'recruitment'],
    fields: [
      {
        id: 'personal_info_heading',
        type: 'heading',
        label: 'Personal Information',
        name: 'personal_info_heading',
        required: false
      },
      {
        id: 'full_name',
        type: 'text',
        label: 'Full Name',
        name: 'full_name',
        placeholder: 'Enter your full legal name',
        required: true
      },
      {
        id: 'email',
        type: 'email',
        label: 'Email Address',
        name: 'email',
        placeholder: 'your@email.com',
        required: true
      },
      {
        id: 'phone',
        type: 'phone',
        label: 'Phone Number',
        name: 'phone',
        placeholder: '+1 (555) 123-4567',
        required: true
      },
      {
        id: 'location',
        type: 'text',
        label: 'Current Location',
        name: 'location',
        placeholder: 'City, State/Province, Country',
        required: true
      },
      {
        id: 'divider1',
        type: 'divider',
        label: '',
        name: 'divider1',
        required: false
      },
      {
        id: 'position_info_heading',
        type: 'heading',
        label: 'Position Information',
        name: 'position_info_heading',
        required: false
      },
      {
        id: 'position',
        type: 'select',
        label: 'Position Applying For',
        name: 'position',
        placeholder: 'Select a position',
        required: true,
        options: [
          { id: '1', label: 'Software Engineer', value: 'software_engineer' },
          { id: '2', label: 'Frontend Developer', value: 'frontend_developer' },
          { id: '3', label: 'Backend Developer', value: 'backend_developer' },
          { id: '4', label: 'Full Stack Developer', value: 'fullstack_developer' },
          { id: '5', label: 'Product Manager', value: 'product_manager' },
          { id: '6', label: 'UI/UX Designer', value: 'ui_ux_designer' },
          { id: '7', label: 'Data Scientist', value: 'data_scientist' },
          { id: '8', label: 'DevOps Engineer', value: 'devops_engineer' }
        ]
      },
      {
        id: 'experience_level',
        type: 'radio',
        label: 'Experience Level',
        name: 'experience_level',
        required: true,
        options: [
          { id: '1', label: 'Entry Level (0-2 years)', value: 'entry' },
          { id: '2', label: 'Mid Level (3-5 years)', value: 'mid' },
          { id: '3', label: 'Senior Level (6-10 years)', value: 'senior' },
          { id: '4', label: 'Lead/Principal (10+ years)', value: 'lead' }
        ]
      },
      {
        id: 'salary_expectation',
        type: 'number',
        label: 'Salary Expectation (Annual USD)',
        name: 'salary_expectation',
        placeholder: 'e.g., 75000',
        required: false
      },
      {
        id: 'start_date',
        type: 'date',
        label: 'Available Start Date',
        name: 'start_date',
        required: true
      },
      {
        id: 'divider2',
        type: 'divider',
        label: '',
        name: 'divider2',
        required: false
      },
      {
        id: 'documents_heading',
        type: 'heading',
        label: 'Documents',
        name: 'documents_heading',
        required: false
      },
      {
        id: 'resume',
        type: 'file',
        label: 'Resume/CV',
        name: 'resume',
        description: 'Please upload your resume in PDF format',
        required: true
      },
      {
        id: 'cover_letter',
        type: 'file',
        label: 'Cover Letter',
        name: 'cover_letter',
        description: 'Optional cover letter',
        required: false
      },
      {
        id: 'portfolio',
        type: 'url',
        label: 'Portfolio/Website',
        name: 'portfolio',
        placeholder: 'https://your-portfolio.com',
        required: false
      },
      {
        id: 'linkedin',
        type: 'url',
        label: 'LinkedIn Profile',
        name: 'linkedin',
        placeholder: 'https://linkedin.com/in/yourprofile',
        required: false
      },
      {
        id: 'additional_info',
        type: 'textarea',
        label: 'Why are you interested in this position?',
        name: 'additional_info',
        placeholder: 'Tell us about your interest in this role and our company...',
        required: false
      }
    ],
    settings: {
      title: 'Join Our Team',
      description: 'We\'re looking for talented individuals to join our growing team. Please complete the application below.',
      submitButtonText: 'Submit Application',
      successMessage: 'Your application has been submitted successfully! We\'ll review it and get back to you within 5-7 business days.',
      errorMessage: 'There was an error submitting your application. Please check all fields and try again.',
      allowMultipleSubmissions: false,
      requireAuthentication: false,
      collectIP: true,
      collectUserAgent: true,
      enableSpamProtection: true,
      enableCaptcha: true,
      enableSaveAndContinue: true,
    },
    styling: {
      theme: 'modern',
      primaryColor: '#059669',
      secondaryColor: '#64748b',
      backgroundColor: '#ffffff',
      textColor: '#1f2937',
      font: 'Inter',
      spacing: 'normal',
      borderRadius: 'md',
      shadow: 'lg',
      responsive: {
        mobile: { columns: 1, spacing: '1rem', fontSize: '14px', padding: '1rem' },
        tablet: { columns: 2, spacing: '1.5rem', fontSize: '16px', padding: '2rem' },
        desktop: { columns: 2, spacing: '2rem', fontSize: '16px', padding: '2rem' },
      },
    }
  },

  {
    id: 'customer-feedback',
    name: 'Customer Feedback',
    description: 'Collect customer feedback with ratings and comments',
    category: 'feedback',
    preview: 'customer-feedback-preview.jpg',
    popularity: 72,
    isPremium: false,
    tags: ['feedback', 'customer', 'rating', 'survey'],
    fields: [
      {
        id: 'overall_rating',
        type: 'rating',
        label: 'Overall Experience',
        name: 'overall_rating',
        description: 'How would you rate your overall experience?',
        required: true
      },
      {
        id: 'service_quality',
        type: 'radio',
        label: 'Service Quality',
        name: 'service_quality',
        required: true,
        options: [
          { id: '1', label: 'Excellent', value: 'excellent' },
          { id: '2', label: 'Good', value: 'good' },
          { id: '3', label: 'Average', value: 'average' },
          { id: '4', label: 'Below Average', value: 'below_average' },
          { id: '5', label: 'Poor', value: 'poor' }
        ]
      },
      {
        id: 'recommend',
        type: 'radio',
        label: 'Would you recommend us to others?',
        name: 'recommend',
        required: true,
        options: [
          { id: '1', label: 'Definitely', value: 'definitely' },
          { id: '2', label: 'Probably', value: 'probably' },
          { id: '3', label: 'Not sure', value: 'not_sure' },
          { id: '4', label: 'Probably not', value: 'probably_not' },
          { id: '5', label: 'Definitely not', value: 'definitely_not' }
        ]
      },
      {
        id: 'improvements',
        type: 'checkbox',
        label: 'What areas could we improve?',
        name: 'improvements',
        required: false,
        options: [
          { id: '1', label: 'Response time', value: 'response_time' },
          { id: '2', label: 'Product quality', value: 'product_quality' },
          { id: '3', label: 'Customer service', value: 'customer_service' },
          { id: '4', label: 'Website usability', value: 'website_usability' },
          { id: '5', label: 'Pricing', value: 'pricing' },
          { id: '6', label: 'Communication', value: 'communication' }
        ]
      },
      {
        id: 'comments',
        type: 'textarea',
        label: 'Additional Comments',
        name: 'comments',
        placeholder: 'Please share any additional feedback or suggestions...',
        required: false
      },
      {
        id: 'email',
        type: 'email',
        label: 'Email (Optional)',
        name: 'email',
        placeholder: 'your@email.com',
        description: 'Provide your email if you\'d like us to follow up on your feedback',
        required: false
      }
    ],
    settings: {
      title: 'We Value Your Feedback',
      description: 'Your feedback helps us improve our service. Please take a few minutes to share your experience.',
      submitButtonText: 'Submit Feedback',
      successMessage: 'Thank you for your valuable feedback! We appreciate you taking the time to help us improve.',
      errorMessage: 'Failed to submit feedback. Please try again.',
      allowMultipleSubmissions: true,
      requireAuthentication: false,
      collectIP: false,
      collectUserAgent: false,
      enableSpamProtection: true,
      enableCaptcha: false,
      enableSaveAndContinue: false,
    },
    styling: {
      theme: 'modern',
      primaryColor: '#f59e0b',
      secondaryColor: '#64748b',
      backgroundColor: '#ffffff',
      textColor: '#1f2937',
      font: 'Inter',
      spacing: 'normal',
      borderRadius: 'lg',
      shadow: 'md',
      responsive: {
        mobile: { columns: 1, spacing: '1rem', fontSize: '14px', padding: '1rem' },
        tablet: { columns: 1, spacing: '1.5rem', fontSize: '16px', padding: '2rem' },
        desktop: { columns: 1, spacing: '2rem', fontSize: '16px', padding: '2rem' },
      },
    }
  }
];

/**
 * Get template by ID
 */
export const getTemplateById = (id: string): FormTemplate | undefined => {
  return formTemplates.find(template => template.id === id);
};

/**
 * Get templates by category
 */
export const getTemplatesByCategory = (category: string): FormTemplate[] => {
  return formTemplates.filter(template => template.category === category);
};

/**
 * Search templates
 */
export const searchTemplates = (query: string): FormTemplate[] => {
  const lowercaseQuery = query.toLowerCase();
  return formTemplates.filter(template => 
    template.name.toLowerCase().includes(lowercaseQuery) ||
    template.description.toLowerCase().includes(lowercaseQuery) ||
    template.tags.some(tag => tag.toLowerCase().includes(lowercaseQuery))
  );
};

/**
 * Get popular templates
 */
export const getPopularTemplates = (limit: number = 5): FormTemplate[] => {
  return formTemplates
    .sort((a, b) => b.popularity - a.popularity)
    .slice(0, limit);
};

/**
 * Get template categories
 */
export const getTemplateCategories = () => {
  const categories = Array.from(new Set(formTemplates.map(template => template.category)));
  return categories.map(category => ({
    id: category,
    name: category.charAt(0).toUpperCase() + category.slice(1).replace('-', ' '),
    count: formTemplates.filter(template => template.category === category).length
  }));
};