'use client';

import React, { useState, useEffect, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { toast } from 'react-hot-toast';
import FormBuilder from '@/components/form-builder/FormBuilder';
import { FormBuilder as FormBuilderType } from '@/types/form-builder';

function FormBuilderContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const formId = searchParams.get('id');
  
  const [form, setForm] = useState<FormBuilderType | null>(null);
  const [loading, setLoading] = useState(true);
  const [user, setUser] = useState<any>(null);

  useEffect(() => {
    // Check authentication
    const token = localStorage.getItem('formhub_token');
    if (!token) {
      router.push('/auth/login');
      return;
    }

    const userData = localStorage.getItem('formhub_user');
    if (userData) {
      setUser(JSON.parse(userData));
    }

    // Load form data if editing existing form
    if (formId) {
      loadForm(formId);
    } else {
      setLoading(false);
    }
  }, [formId, router]);

  const loadForm = async (id: string) => {
    try {
      const token = localStorage.getItem('formhub_token');
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'https://formhub.mejona.in/api/v1';
      
      const response = await fetch(`${apiUrl}/forms/${id}`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      });

      if (response.ok) {
        const data = await response.json();
        // Convert backend form data to FormBuilder format
        const formBuilderData: FormBuilderType = {
          id: data.form.id,
          name: data.form.name,
          description: data.form.description,
          fields: [], // Will be populated from form configuration
          steps: undefined,
          isMultiStep: false,
          settings: {
            title: data.form.name,
            description: data.form.description || '',
            submitButtonText: 'Submit',
            successMessage: data.form.success_message || 'Thank you for your submission!',
            errorMessage: 'There was an error submitting the form. Please try again.',
            redirectUrl: data.form.redirect_url || '',
            allowMultipleSubmissions: true,
            requireAuthentication: false,
            collectIP: false,
            collectUserAgent: false,
            enableSpamProtection: data.form.is_active,
            enableCaptcha: false,
            enableSaveAndContinue: false,
          },
          styling: {
            theme: 'default',
            primaryColor: '#3b82f6',
            secondaryColor: '#64748b',
            backgroundColor: '#ffffff',
            textColor: '#1f2937',
            font: 'Inter',
            spacing: 'normal',
            borderRadius: 'md',
            shadow: 'sm',
            responsive: {
              mobile: { columns: 1, spacing: '1rem', fontSize: '14px', padding: '1rem' },
              tablet: { columns: 2, spacing: '1.5rem', fontSize: '16px', padding: '2rem' },
              desktop: { columns: 2, spacing: '2rem', fontSize: '16px', padding: '2rem' },
            },
          },
          notifications: {
            email: {
              enabled: true,
              to: [data.form.target_email],
              cc: Array.isArray(data.form.cc_emails) ? data.form.cc_emails : 
                  (data.form.cc_emails ? [data.form.cc_emails] : []),
              bcc: [],
              subject: data.form.subject || 'New Form Submission',
              template: 'You have received a new form submission.',
              includeAttachments: true,
            },
            webhook: {
              enabled: false,
              url: '',
              method: 'POST',
            },
            autoresponder: {
              enabled: false,
              emailField: 'email',
              subject: 'Thank you for your submission',
              template: 'We have received your submission and will get back to you soon.',
            },
          },
          integrations: {},
          analytics: {
            enabled: true,
            trackViews: true,
            trackCompletions: true,
            trackConversions: true,
            trackFieldInteractions: false,
            heatmap: false,
            abTesting: false,
          },
          createdAt: new Date(data.form.created_at),
          updatedAt: new Date(data.form.updated_at),
        };
        
        setForm(formBuilderData);
      } else if (response.status === 401) {
        localStorage.removeItem('formhub_token');
        router.push('/auth/login');
      } else {
        toast.error('Failed to load form');
      }
    } catch (error) {
      console.error('Error loading form:', error);
      toast.error('Failed to load form');
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async (formData: FormBuilderType) => {
    try {
      const token = localStorage.getItem('formhub_token');
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'https://formhub.mejona.in/api/v1';
      
      // Convert FormBuilder data to backend format
      const backendData = {
        name: formData.name,
        description: formData.description,
        target_email: formData.notifications.email.to[0] || '',
        cc_emails: formData.notifications.email.cc,
        subject: formData.notifications.email.subject,
        success_message: formData.settings.successMessage,
        redirect_url: formData.settings.redirectUrl || '',
        is_active: formData.settings.enableSpamProtection,
        // Additional FormBuilder-specific data can be stored as JSON
        form_builder_config: JSON.stringify({
          fields: formData.fields,
          settings: formData.settings,
          styling: formData.styling,
          notifications: formData.notifications,
          integrations: formData.integrations,
          analytics: formData.analytics,
        }),
      };

      const url = formId 
        ? `${apiUrl}/forms/${formId}`
        : `${apiUrl}/forms`;
      
      const method = formId ? 'PUT' : 'POST';

      const response = await fetch(url, {
        method,
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(backendData)
      });

      if (response.ok) {
        const data = await response.json();
        if (!formId) {
          // If this was a new form, update the URL with the new form ID
          const newFormId = data.form.id;
          router.replace(`/dashboard/forms/builder?id=${newFormId}`);
        }
        return data;
      } else {
        throw new Error('Save failed');
      }
    } catch (error) {
      console.error('Error saving form:', error);
      throw error;
    }
  };

  const handlePublish = async (formData: FormBuilderType) => {
    try {
      // First save the form
      await handleSave(formData);
      
      // Then activate it
      const token = localStorage.getItem('formhub_token');
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'https://formhub.mejona.in/api/v1';
      
      if (formId) {
        const response = await fetch(`${apiUrl}/forms/${formId}`, {
          method: 'PUT',
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({ is_active: true })
        });

        if (!response.ok) {
          throw new Error('Publish failed');
        }
      }
    } catch (error) {
      console.error('Error publishing form:', error);
      throw error;
    }
  };

  const handlePreview = (formData: FormBuilderType) => {
    // Open preview in a new window or tab
    const previewData = {
      form: formData,
      timestamp: Date.now()
    };
    
    // Store preview data in sessionStorage for the preview window
    sessionStorage.setItem('form_preview', JSON.stringify(previewData));
    
    // Open preview window with correct base path for GitHub Pages
    const basePath = process.env.NEXT_PUBLIC_BASE_PATH || '';
    const previewUrl = `${basePath}/form-preview`;
    const previewWindow = window.open(previewUrl, '_blank', 'width=800,height=600');
    if (!previewWindow) {
      toast.error('Please allow popups to open preview window');
    }
  };

  if (loading) {
    return (
      <div className="h-screen flex items-center justify-center bg-gray-50">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading form builder...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-screen">
      <FormBuilder
        initialForm={form || undefined}
        onSave={handleSave}
        onPublish={handlePublish}
        onPreview={handlePreview}
      />
    </div>
  );
}

export default function FormBuilderPage() {
  return (
    <Suspense fallback={
      <div className="h-screen flex items-center justify-center bg-gray-50">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading form builder...</p>
        </div>
      </div>
    }>
      <FormBuilderContent />
    </Suspense>
  );
}