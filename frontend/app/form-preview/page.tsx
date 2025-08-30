'use client';

import React, { useState, useEffect } from 'react';
import { FormBuilder as FormBuilderType } from '@/types/form-builder';
import FieldRenderer from '@/components/form-builder/fields/FieldRenderer';
import { validateForm, formatSubmissionData } from '@/lib/form-builder/utils';
import { toast } from 'react-hot-toast';

export default function FormPreviewPage() {
  const [form, setForm] = useState<FormBuilderType | null>(null);
  const [formValues, setFormValues] = useState<Record<string, any>>({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    // Get form data from sessionStorage (set by form builder)
    const previewData = sessionStorage.getItem('form_preview');
    if (previewData) {
      try {
        const data = JSON.parse(previewData);
        setForm(data.form);
      } catch (error) {
        console.error('Failed to parse preview data:', error);
        toast.error('Failed to load form preview');
      }
    } else {
      toast.error('No form data found');
    }
  }, []);

  const handleFieldChange = (fieldId: string, value: any) => {
    setFormValues(prev => ({ ...prev, [fieldId]: value }));
    // Clear field error when value changes
    if (errors[fieldId]) {
      setErrors(prev => ({ ...prev, [fieldId]: '' }));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form) return;

    // Validate form
    const validation = validateForm(form, formValues);
    if (!validation.isValid) {
      const fieldErrors: Record<string, string> = {};
      validation.errors.forEach(error => {
        fieldErrors[error.fieldId] = error.message;
      });
      setErrors(fieldErrors);
      toast.error('Please fix the errors and try again');
      return;
    }

    setIsSubmitting(true);
    setErrors({});

    try {
      // Format submission data
      const submissionData = formatSubmissionData(form, formValues);
      
      // In preview mode, just show the data
      console.log('Form submission data:', submissionData);
      toast.success('Form validation passed! Check console for submission data.');
      
      // Reset form after successful "submission"
      setTimeout(() => {
        setFormValues({});
      }, 1000);
      
    } catch (error) {
      console.error('Submission error:', error);
      toast.error('Failed to submit form');
    } finally {
      setIsSubmitting(false);
    }
  };

  if (!form) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading form preview...</p>
        </div>
      </div>
    );
  }

  const formStyles: React.CSSProperties = {
    backgroundColor: form.styling.backgroundColor,
    color: form.styling.textColor,
    fontFamily: form.styling.font,
  };

  const containerClasses = `
    min-h-screen py-8 px-4
    ${form.styling.spacing === 'tight' ? 'space-y-4' : 
      form.styling.spacing === 'relaxed' ? 'space-y-8' : 'space-y-6'}
  `;

  return (
    <div className={containerClasses} style={{ backgroundColor: form.styling.backgroundColor }}>
      <div className="max-w-2xl mx-auto">
        {/* Preview Header */}
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-8">
          <h1 className="text-lg font-semibold text-blue-900 mb-2">Form Preview</h1>
          <p className="text-blue-700 text-sm">
            This is a preview of your form. Form submissions will be logged to the console.
          </p>
        </div>

        {/* Form */}
        <form 
          onSubmit={handleSubmit}
          className={`
            bg-white rounded-lg p-8 shadow-sm border
            ${form.styling.shadow === 'none' ? '' :
              form.styling.shadow === 'sm' ? 'shadow-sm' :
              form.styling.shadow === 'md' ? 'shadow-md' :
              form.styling.shadow === 'lg' ? 'shadow-lg' : 'shadow-xl'}
          `}
          style={formStyles}
        >
          {/* Form Header */}
          {form.settings.title && (
            <div className="mb-8">
              <h2 
                className="text-2xl font-bold mb-2"
                style={{ color: form.styling.primaryColor }}
              >
                {form.settings.title}
              </h2>
              {form.settings.description && (
                <p className="text-gray-600">
                  {form.settings.description}
                </p>
              )}
            </div>
          )}

          {/* Form Fields */}
          <div className="space-y-6">
            {form.fields.map(field => (
              <div key={field.id}>
                <FieldRenderer
                  field={field}
                  isPreview={true}
                  onChange={handleFieldChange}
                  value={formValues[field.id]}
                />
                {errors[field.id] && (
                  <p className="mt-1 text-sm text-red-600">
                    {errors[field.id]}
                  </p>
                )}
              </div>
            ))}
          </div>

          {/* Submit Button */}
          <div className="mt-8">
            <button
              type="submit"
              disabled={isSubmitting}
              className="w-full py-3 px-4 rounded-md font-medium text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              style={{ 
                backgroundColor: form.styling.primaryColor,
                borderRadius: form.styling.borderRadius === 'none' ? '0' :
                               form.styling.borderRadius === 'sm' ? '0.375rem' :
                               form.styling.borderRadius === 'md' ? '0.5rem' :
                               form.styling.borderRadius === 'lg' ? '0.75rem' : '1rem'
              }}
            >
              {isSubmitting ? (
                <div className="flex items-center justify-center">
                  <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin mr-2"></div>
                  Submitting...
                </div>
              ) : (
                form.settings.submitButtonText
              )}
            </button>
          </div>
        </form>

        {/* Form Info */}
        <div className="mt-8 text-center text-sm text-gray-500">
          <p>Form ID: {form.id}</p>
          <p>Fields: {form.fields.length}</p>
          <p>Last updated: {new Date(form.updatedAt).toLocaleString()}</p>
        </div>
      </div>
    </div>
  );
}