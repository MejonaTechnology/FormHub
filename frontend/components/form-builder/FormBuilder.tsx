'use client';

import React, { useState, useCallback, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { toast } from 'react-hot-toast';
import { 
  Save, 
  Undo, 
  Redo, 
  Settings, 
  Eye, 
  Code, 
  Download, 
  Upload,
  Send,
  FileText,
  Layers
} from 'lucide-react';

import { 
  FormBuilder as FormBuilderType, 
  FormField, 
  FormSettings, 
  FormStyling, 
  FormNotifications,
  FormBuilderHistory,
  PaletteComponent 
} from '@/types/form-builder';

import ComponentPalette from './palette/ComponentPalette';
import FormCanvas from './FormCanvas';
import FieldConfigPanel from './configuration/FieldConfigPanel';
import FormSettingsPanel from './configuration/FormSettingsPanel';

interface FormBuilderProps {
  initialForm?: FormBuilderType;
  onSave?: (form: FormBuilderType) => void;
  onPublish?: (form: FormBuilderType) => void;
  onPreview?: (form: FormBuilderType) => void;
}

const FormBuilder: React.FC<FormBuilderProps> = ({
  initialForm,
  onSave,
  onPublish,
  onPreview,
}) => {
  // Form state
  const [form, setForm] = useState<FormBuilderType>(() => 
    initialForm || {
      id: `form_${Date.now()}`,
      name: 'Untitled Form',
      description: '',
      fields: [],
      steps: undefined,
      isMultiStep: false,
      settings: {
        title: 'Contact Form',
        description: 'Please fill out this form to get in touch.',
        submitButtonText: 'Submit',
        successMessage: 'Thank you for your submission!',
        errorMessage: 'There was an error submitting the form. Please try again.',
        allowMultipleSubmissions: true,
        requireAuthentication: false,
        collectIP: false,
        collectUserAgent: false,
        enableSpamProtection: true,
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
          to: ['admin@example.com'],
          cc: [],
          bcc: [],
          subject: 'New Form Submission',
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
      createdAt: new Date(),
      updatedAt: new Date(),
    }
  );

  // UI state
  const [selectedFieldId, setSelectedFieldId] = useState<string | undefined>();
  const [isPreviewMode, setIsPreviewMode] = useState(false);
  const [showFieldConfig, setShowFieldConfig] = useState(false);
  const [showFormSettings, setShowFormSettings] = useState(false);
  const [formValues, setFormValues] = useState<Record<string, any>>({});
  const [isSaving, setIsSaving] = useState(false);
  const [lastSaved, setLastSaved] = useState<Date | null>(null);

  // History for undo/redo
  const [history, setHistory] = useState<FormBuilderHistory>({
    past: [],
    present: form,
    future: [],
  });

  // Auto-save functionality
  useEffect(() => {
    const autoSaveTimer = setTimeout(() => {
      if (form !== history.present && onSave) {
        handleSave(true);
      }
    }, 30000); // Auto-save every 30 seconds

    return () => clearTimeout(autoSaveTimer);
  }, [form, history.present]);

  // Update history when form changes
  const updateHistory = useCallback((newForm: FormBuilderType) => {
    setHistory(prev => ({
      past: [...prev.past, prev.present],
      present: newForm,
      future: [],
    }));
  }, []);

  // Form manipulation handlers
  const handleFieldsChange = useCallback((newFields: FormField[]) => {
    const updatedForm = { ...form, fields: newFields, updatedAt: new Date() };
    setForm(updatedForm);
    updateHistory(updatedForm);
  }, [form, updateHistory]);

  const handleFieldSelect = useCallback((fieldId: string) => {
    setSelectedFieldId(fieldId);
    setShowFieldConfig(true);
    setShowFormSettings(false);
  }, []);

  const handleFieldDelete = useCallback((fieldId: string) => {
    const newFields = form.fields.filter(field => field.id !== fieldId);
    handleFieldsChange(newFields);
    
    if (selectedFieldId === fieldId) {
      setSelectedFieldId(undefined);
      setShowFieldConfig(false);
    }
    
    toast.success('Field deleted successfully');
  }, [form.fields, selectedFieldId, handleFieldsChange]);

  const handleFieldDuplicate = useCallback((fieldId: string) => {
    const fieldToDuplicate = form.fields.find(field => field.id === fieldId);
    if (!fieldToDuplicate) return;

    const duplicatedField: FormField = {
      ...fieldToDuplicate,
      id: `field_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      name: `${fieldToDuplicate.name}_copy`,
      label: `${fieldToDuplicate.label} (Copy)`,
    };

    const fieldIndex = form.fields.findIndex(field => field.id === fieldId);
    const newFields = [...form.fields];
    newFields.splice(fieldIndex + 1, 0, duplicatedField);
    
    handleFieldsChange(newFields);
    toast.success('Field duplicated successfully');
  }, [form.fields, handleFieldsChange]);

  const handleFieldEdit = useCallback((fieldId: string) => {
    handleFieldSelect(fieldId);
  }, [handleFieldSelect]);

  const handleFieldUpdate = useCallback((updatedField: FormField) => {
    const newFields = form.fields.map(field => 
      field.id === updatedField.id ? updatedField : field
    );
    handleFieldsChange(newFields);
  }, [form.fields, handleFieldsChange]);

  const handleFormValueChange = useCallback((fieldId: string, value: any) => {
    setFormValues(prev => ({ ...prev, [fieldId]: value }));
  }, []);

  // Settings handlers
  const handleSettingsUpdate = useCallback((newSettings: FormSettings) => {
    const updatedForm = { ...form, settings: newSettings, updatedAt: new Date() };
    setForm(updatedForm);
    updateHistory(updatedForm);
  }, [form, updateHistory]);

  const handleStylingUpdate = useCallback((newStyling: FormStyling) => {
    const updatedForm = { ...form, styling: newStyling, updatedAt: new Date() };
    setForm(updatedForm);
    updateHistory(updatedForm);
  }, [form, updateHistory]);

  const handleNotificationsUpdate = useCallback((newNotifications: FormNotifications) => {
    const updatedForm = { ...form, notifications: newNotifications, updatedAt: new Date() };
    setForm(updatedForm);
    updateHistory(updatedForm);
  }, [form, updateHistory]);

  // Action handlers
  const handleSave = useCallback(async (isAutoSave = false) => {
    if (!onSave) return;
    
    setIsSaving(true);
    try {
      await onSave(form);
      setLastSaved(new Date());
      if (!isAutoSave) {
        toast.success('Form saved successfully');
      }
    } catch (error) {
      toast.error('Failed to save form');
      console.error('Save error:', error);
    } finally {
      setIsSaving(false);
    }
  }, [form, onSave]);

  const handlePublish = useCallback(async () => {
    if (!onPublish) return;
    
    // Validation before publishing
    if (!form.fields.length) {
      toast.error('Cannot publish form without any fields');
      return;
    }

    if (!form.notifications.email.enabled || !form.notifications.email.to.length) {
      toast.error('Please configure email notifications before publishing');
      return;
    }

    try {
      await onPublish(form);
      toast.success('Form published successfully');
    } catch (error) {
      toast.error('Failed to publish form');
      console.error('Publish error:', error);
    }
  }, [form, onPublish]);

  const handlePreview = useCallback(() => {
    if (onPreview) {
      onPreview(form);
    }
    setIsPreviewMode(!isPreviewMode);
  }, [form, isPreviewMode, onPreview]);

  const handleUndo = useCallback(() => {
    if (history.past.length === 0) return;
    
    const previous = history.past[history.past.length - 1];
    const newPast = history.past.slice(0, history.past.length - 1);
    
    setHistory({
      past: newPast,
      present: previous,
      future: [history.present, ...history.future],
    });
    
    setForm(previous);
  }, [history]);

  const handleRedo = useCallback(() => {
    if (history.future.length === 0) return;
    
    const next = history.future[0];
    const newFuture = history.future.slice(1);
    
    setHistory({
      past: [...history.past, history.present],
      present: next,
      future: newFuture,
    });
    
    setForm(next);
  }, [history]);

  const selectedField = selectedFieldId ? form.fields.find(field => field.id === selectedFieldId) : null;

  return (
    <div className="h-screen flex flex-col bg-gray-50">
      {/* Top Toolbar */}
      <div className="bg-white border-b border-gray-200 px-6 py-3">
        <div className="flex items-center justify-between">
          {/* Left section */}
          <div className="flex items-center space-x-4">
            <div>
              <input
                type="text"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                className="text-lg font-semibold text-gray-900 bg-transparent border-none p-0 focus:ring-0 focus:outline-none"
                placeholder="Untitled Form"
              />
              <div className="flex items-center space-x-2 text-sm text-gray-500">
                <span>{form.fields.length} fields</span>
                {lastSaved && (
                  <>
                    <span>•</span>
                    <span>Saved {lastSaved.toLocaleTimeString()}</span>
                  </>
                )}
              </div>
            </div>
          </div>

          {/* Center section - Actions */}
          <div className="flex items-center space-x-2">
            <button
              onClick={handleUndo}
              disabled={history.past.length === 0}
              className="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed"
              title="Undo"
            >
              <Undo size={18} />
            </button>
            
            <button
              onClick={handleRedo}
              disabled={history.future.length === 0}
              className="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed"
              title="Redo"
            >
              <Redo size={18} />
            </button>

            <div className="w-px h-6 bg-gray-300" />

            <button
              onClick={() => {
                setShowFormSettings(!showFormSettings);
                setShowFieldConfig(false);
              }}
              className={`
                flex items-center space-x-2 px-3 py-2 text-sm font-medium rounded-lg transition-colors
                ${showFormSettings 
                  ? 'bg-blue-100 text-blue-700' 
                  : 'text-gray-700 hover:text-gray-900 hover:bg-gray-100'
                }
              `}
            >
              <Settings size={16} />
              <span>Settings</span>
            </button>

            <button
              onClick={() => {
                // TODO: Implement template functionality
                toast('Templates feature coming soon!', { icon: 'ℹ️' });
              }}
              className="flex items-center space-x-2 px-3 py-2 text-sm font-medium text-gray-700 hover:text-gray-900 hover:bg-gray-100 rounded-lg transition-colors"
            >
              <FileText size={16} />
              <span>Templates</span>
            </button>
          </div>

          {/* Right section */}
          <div className="flex items-center space-x-3">
            <button
              onClick={handlePreview}
              className={`
                flex items-center space-x-2 px-4 py-2 text-sm font-medium rounded-lg transition-colors
                ${isPreviewMode 
                  ? 'bg-gray-100 text-gray-700' 
                  : 'bg-blue-600 text-white hover:bg-blue-700'
                }
              `}
            >
              <Eye size={16} />
              <span>{isPreviewMode ? 'Edit' : 'Preview'}</span>
            </button>

            <button
              onClick={() => handleSave()}
              disabled={isSaving}
              className="flex items-center space-x-2 px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 rounded-lg transition-colors disabled:opacity-50"
            >
              <Save size={16} />
              <span>{isSaving ? 'Saving...' : 'Save'}</span>
            </button>

            <button
              onClick={handlePublish}
              className="flex items-center space-x-2 px-4 py-2 text-sm font-medium text-white bg-green-600 hover:bg-green-700 rounded-lg transition-colors"
            >
              <Send size={16} />
              <span>Publish</span>
            </button>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left Sidebar - Component Palette */}
        {!isPreviewMode && (
          <ComponentPalette />
        )}

        {/* Center - Form Canvas */}
        <FormCanvas
          fields={form.fields}
          selectedFieldId={selectedFieldId}
          isPreviewMode={isPreviewMode}
          onFieldsChange={handleFieldsChange}
          onFieldSelect={handleFieldSelect}
          onFieldDelete={handleFieldDelete}
          onFieldDuplicate={handleFieldDuplicate}
          onFieldEdit={handleFieldEdit}
          onPreviewToggle={handlePreview}
          formValues={formValues}
          onFormValueChange={handleFormValueChange}
        />

        {/* Right Sidebar - Configuration Panels */}
        <AnimatePresence mode="wait">
          {showFieldConfig && selectedField && !isPreviewMode && (
            <FieldConfigPanel
              field={selectedField}
              allFields={form.fields}
              onFieldUpdate={handleFieldUpdate}
              onClose={() => setShowFieldConfig(false)}
            />
          )}
          
          {showFormSettings && !isPreviewMode && (
            <FormSettingsPanel
              settings={form.settings}
              styling={form.styling}
              notifications={form.notifications}
              onSettingsUpdate={handleSettingsUpdate}
              onStylingUpdate={handleStylingUpdate}
              onNotificationsUpdate={handleNotificationsUpdate}
              onClose={() => setShowFormSettings(false)}
            />
          )}
        </AnimatePresence>
      </div>

      {/* Bottom Status Bar */}
      <div className="bg-white border-t border-gray-200 px-6 py-2">
        <div className="flex items-center justify-between text-sm text-gray-500">
          <div className="flex items-center space-x-4">
            <span>{form.fields.length} fields</span>
            <span>•</span>
            <span>Form ID: {form.id}</span>
            {form.analytics?.enabled && (
              <>
                <span>•</span>
                <span>Analytics enabled</span>
              </>
            )}
          </div>
          
          <div className="flex items-center space-x-2">
            {isSaving && (
              <div className="flex items-center space-x-2 text-blue-600">
                <div className="w-3 h-3 border border-blue-600 border-t-transparent rounded-full animate-spin" />
                <span>Saving...</span>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default FormBuilder;