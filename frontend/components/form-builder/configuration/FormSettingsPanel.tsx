'use client';

import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { 
  Settings, 
  Mail, 
  Palette, 
  Shield, 
  Webhook, 
  ChevronDown, 
  ChevronRight,
  X,
  Copy,
  Eye,
  Code
} from 'lucide-react';
import { HexColorPicker } from 'react-colorful';
import { FormSettings, FormStyling, FormNotifications } from '@/types/form-builder';

interface FormSettingsPanelProps {
  settings: FormSettings;
  styling: FormStyling;
  notifications: FormNotifications;
  onSettingsUpdate: (settings: FormSettings) => void;
  onStylingUpdate: (styling: FormStyling) => void;
  onNotificationsUpdate: (notifications: FormNotifications) => void;
  onClose: () => void;
}

const FormSettingsPanel: React.FC<FormSettingsPanelProps> = ({
  settings,
  styling,
  notifications,
  onSettingsUpdate,
  onStylingUpdate,
  onNotificationsUpdate,
  onClose,
}) => {
  const [activeSection, setActiveSection] = useState<string>('general');
  const [showColorPicker, setShowColorPicker] = useState<string | null>(null);
  const [expandedSections, setExpandedSections] = useState<string[]>(['basic']);

  const toggleSection = (section: string) => {
    setExpandedSections(prev => 
      prev.includes(section) 
        ? prev.filter(s => s !== section)
        : [...prev, section]
    );
  };

  const updateSettings = (updates: Partial<FormSettings>) => {
    onSettingsUpdate({ ...settings, ...updates });
  };

  const updateStyling = (updates: Partial<FormStyling>) => {
    onStylingUpdate({ ...styling, ...updates });
  };

  const updateNotifications = (updates: Partial<FormNotifications>) => {
    onNotificationsUpdate({ ...notifications, ...updates });
  };

  const sections = [
    { id: 'general', name: 'General', icon: <Settings size={16} /> },
    { id: 'styling', name: 'Styling', icon: <Palette size={16} /> },
    { id: 'notifications', name: 'Notifications', icon: <Mail size={16} /> },
    { id: 'security', name: 'Security', icon: <Shield size={16} /> },
    { id: 'integrations', name: 'Integrations', icon: <Webhook size={16} /> },
  ];

  const renderGeneralSection = () => (
    <div className="space-y-6">
      {/* Basic Settings */}
      <div>
        <button
          onClick={() => toggleSection('basic')}
          className="flex items-center w-full text-left text-sm font-medium text-gray-900 mb-3"
        >
          {expandedSections.includes('basic') ? (
            <ChevronDown size={16} className="mr-2" />
          ) : (
            <ChevronRight size={16} className="mr-2" />
          )}
          Basic Information
        </button>
        
        {expandedSections.includes('basic') && (
          <div className="space-y-4 pl-6">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Form Title *
              </label>
              <input
                type="text"
                value={settings.title}
                onChange={(e) => updateSettings({ title: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Description
              </label>
              <textarea
                value={settings.description || ''}
                onChange={(e) => updateSettings({ description: e.target.value })}
                rows={3}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Submit Button Text
              </label>
              <input
                type="text"
                value={settings.submitButtonText}
                onChange={(e) => updateSettings({ submitButtonText: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
            </div>
          </div>
        )}
      </div>

      {/* Messages */}
      <div>
        <button
          onClick={() => toggleSection('messages')}
          className="flex items-center w-full text-left text-sm font-medium text-gray-900 mb-3"
        >
          {expandedSections.includes('messages') ? (
            <ChevronDown size={16} className="mr-2" />
          ) : (
            <ChevronRight size={16} className="mr-2" />
          )}
          Messages
        </button>
        
        {expandedSections.includes('messages') && (
          <div className="space-y-4 pl-6">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Success Message
              </label>
              <textarea
                value={settings.successMessage}
                onChange={(e) => updateSettings({ successMessage: e.target.value })}
                rows={3}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Error Message
              </label>
              <textarea
                value={settings.errorMessage}
                onChange={(e) => updateSettings({ errorMessage: e.target.value })}
                rows={3}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Redirect URL (Optional)
              </label>
              <input
                type="url"
                value={settings.redirectUrl || ''}
                onChange={(e) => updateSettings({ redirectUrl: e.target.value })}
                placeholder="https://example.com/thank-you"
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
            </div>
          </div>
        )}
      </div>

      {/* Advanced Options */}
      <div>
        <button
          onClick={() => toggleSection('advanced')}
          className="flex items-center w-full text-left text-sm font-medium text-gray-900 mb-3"
        >
          {expandedSections.includes('advanced') ? (
            <ChevronDown size={16} className="mr-2" />
          ) : (
            <ChevronRight size={16} className="mr-2" />
          )}
          Advanced Options
        </button>
        
        {expandedSections.includes('advanced') && (
          <div className="space-y-4 pl-6">
            <div className="space-y-3">
              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={settings.allowMultipleSubmissions}
                  onChange={(e) => updateSettings({ allowMultipleSubmissions: e.target.checked })}
                  className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                />
                <span className="ml-2 text-sm text-gray-700">
                  Allow multiple submissions from same user
                </span>
              </label>

              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={settings.collectIP}
                  onChange={(e) => updateSettings({ collectIP: e.target.checked })}
                  className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                />
                <span className="ml-2 text-sm text-gray-700">
                  Collect IP addresses
                </span>
              </label>

              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={settings.collectUserAgent}
                  onChange={(e) => updateSettings({ collectUserAgent: e.target.checked })}
                  className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                />
                <span className="ml-2 text-sm text-gray-700">
                  Collect browser information
                </span>
              </label>

              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={settings.enableSaveAndContinue}
                  onChange={(e) => updateSettings({ enableSaveAndContinue: e.target.checked })}
                  className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                />
                <span className="ml-2 text-sm text-gray-700">
                  Enable save and continue later
                </span>
              </label>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs text-gray-600 mb-1">Max Submissions</label>
                <input
                  type="number"
                  value={settings.maxSubmissions || ''}
                  onChange={(e) => updateSettings({ maxSubmissions: e.target.value ? Number(e.target.value) : undefined })}
                  placeholder="Unlimited"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs text-gray-600 mb-1">Deadline</label>
                <input
                  type="datetime-local"
                  value={settings.submissionDeadline ? new Date(settings.submissionDeadline).toISOString().slice(0, 16) : ''}
                  onChange={(e) => updateSettings({ submissionDeadline: e.target.value ? new Date(e.target.value) : undefined })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );

  const renderStylingSection = () => (
    <div className="space-y-6">
      {/* Theme */}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-3">Theme</label>
        <div className="grid grid-cols-2 gap-2">
          {(['default', 'minimal', 'modern', 'classic'] as const).map(theme => (
            <button
              key={theme}
              onClick={() => updateStyling({ theme })}
              className={`
                p-3 border rounded-lg text-left capitalize transition-colors
                ${styling.theme === theme 
                  ? 'border-blue-500 bg-blue-50 text-blue-700' 
                  : 'border-gray-300 hover:border-gray-400'
                }
              `}
            >
              {theme}
            </button>
          ))}
        </div>
      </div>

      {/* Colors */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 mb-3">Colors</h4>
        <div className="space-y-3">
          {[
            { key: 'primaryColor', label: 'Primary Color' },
            { key: 'secondaryColor', label: 'Secondary Color' },
            { key: 'backgroundColor', label: 'Background Color' },
            { key: 'textColor', label: 'Text Color' },
          ].map(({ key, label }) => (
            <div key={key} className="flex items-center space-x-3">
              <label className="text-sm text-gray-700 w-24">{label}</label>
              <button
                onClick={() => setShowColorPicker(showColorPicker === key ? null : key)}
                className="w-8 h-8 rounded border border-gray-300 relative"
                style={{ backgroundColor: (styling as any)[key] }}
              />
              <input
                type="text"
                value={(styling as any)[key]}
                onChange={(e) => updateStyling({ [key]: e.target.value } as any)}
                className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
              
              {showColorPicker === key && (
                <div className="absolute z-10 mt-2">
                  <div className="fixed inset-0" onClick={() => setShowColorPicker(null)} />
                  <div className="relative bg-white p-3 rounded-lg shadow-lg border">
                    <HexColorPicker
                      color={(styling as any)[key]}
                      onChange={(color) => updateStyling({ [key]: color } as any)}
                    />
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Typography */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 mb-3">Typography</h4>
        <div>
          <label className="block text-sm text-gray-700 mb-1">Font Family</label>
          <select
            value={styling.font}
            onChange={(e) => updateStyling({ font: e.target.value })}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          >
            <option value="Inter">Inter</option>
            <option value="Roboto">Roboto</option>
            <option value="Open Sans">Open Sans</option>
            <option value="Lato">Lato</option>
            <option value="Montserrat">Montserrat</option>
            <option value="Poppins">Poppins</option>
          </select>
        </div>
      </div>

      {/* Spacing & Layout */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 mb-3">Layout</h4>
        <div className="space-y-3">
          <div>
            <label className="block text-sm text-gray-700 mb-1">Spacing</label>
            <select
              value={styling.spacing}
              onChange={(e) => updateStyling({ spacing: e.target.value as any })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="tight">Tight</option>
              <option value="normal">Normal</option>
              <option value="relaxed">Relaxed</option>
            </select>
          </div>

          <div>
            <label className="block text-sm text-gray-700 mb-1">Border Radius</label>
            <select
              value={styling.borderRadius}
              onChange={(e) => updateStyling({ borderRadius: e.target.value as any })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="none">None</option>
              <option value="sm">Small</option>
              <option value="md">Medium</option>
              <option value="lg">Large</option>
              <option value="xl">Extra Large</option>
            </select>
          </div>

          <div>
            <label className="block text-sm text-gray-700 mb-1">Shadow</label>
            <select
              value={styling.shadow}
              onChange={(e) => updateStyling({ shadow: e.target.value as any })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="none">None</option>
              <option value="sm">Small</option>
              <option value="md">Medium</option>
              <option value="lg">Large</option>
              <option value="xl">Extra Large</option>
            </select>
          </div>
        </div>
      </div>

      {/* Custom CSS */}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">Custom CSS</label>
        <textarea
          value={styling.customCSS || ''}
          onChange={(e) => updateStyling({ customCSS: e.target.value })}
          placeholder="/* Add your custom CSS here */"
          rows={6}
          className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500 font-mono text-sm"
        />
      </div>
    </div>
  );

  const renderNotificationsSection = () => (
    <div className="space-y-6">
      {/* Email Notifications */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h4 className="text-sm font-medium text-gray-900">Email Notifications</h4>
          <div className="flex items-center">
            <input
              type="checkbox"
              id="email-enabled"
              checked={notifications.email.enabled}
              onChange={(e) => updateNotifications({
                email: { ...notifications.email, enabled: e.target.checked }
              })}
              className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
            />
            <label htmlFor="email-enabled" className="ml-2 text-sm text-gray-700">
              Enable
            </label>
          </div>
        </div>

        {notifications.email.enabled && (
          <div className="space-y-4 p-4 bg-gray-50 rounded-lg">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                To Email(s) *
              </label>
              <input
                type="text"
                value={notifications.email.to.join(', ')}
                onChange={(e) => updateNotifications({
                  email: { 
                    ...notifications.email, 
                    to: e.target.value.split(',').map(email => email.trim()).filter(Boolean)
                  }
                })}
                placeholder="admin@example.com, support@example.com"
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm text-gray-700 mb-1">CC</label>
                <input
                  type="text"
                  value={(notifications.email.cc || []).join(', ')}
                  onChange={(e) => updateNotifications({
                    email: { 
                      ...notifications.email, 
                      cc: e.target.value.split(',').map(email => email.trim()).filter(Boolean)
                    }
                  })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm text-gray-700 mb-1">BCC</label>
                <input
                  type="text"
                  value={(notifications.email.bcc || []).join(', ')}
                  onChange={(e) => updateNotifications({
                    email: { 
                      ...notifications.email, 
                      bcc: e.target.value.split(',').map(email => email.trim()).filter(Boolean)
                    }
                  })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Email Subject
              </label>
              <input
                type="text"
                value={notifications.email.subject}
                onChange={(e) => updateNotifications({
                  email: { ...notifications.email, subject: e.target.value }
                })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
            </div>

            <div className="flex items-center">
              <input
                type="checkbox"
                id="include-attachments"
                checked={notifications.email.includeAttachments}
                onChange={(e) => updateNotifications({
                  email: { ...notifications.email, includeAttachments: e.target.checked }
                })}
                className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
              />
              <label htmlFor="include-attachments" className="ml-2 text-sm text-gray-700">
                Include file attachments
              </label>
            </div>
          </div>
        )}
      </div>

      {/* Autoresponder */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h4 className="text-sm font-medium text-gray-900">Autoresponder</h4>
          <div className="flex items-center">
            <input
              type="checkbox"
              id="autoresponder-enabled"
              checked={notifications.autoresponder.enabled}
              onChange={(e) => updateNotifications({
                autoresponder: { ...notifications.autoresponder, enabled: e.target.checked }
              })}
              className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
            />
            <label htmlFor="autoresponder-enabled" className="ml-2 text-sm text-gray-700">
              Enable
            </label>
          </div>
        </div>

        {notifications.autoresponder.enabled && (
          <div className="space-y-4 p-4 bg-gray-50 rounded-lg">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Email Field *
              </label>
              <input
                type="text"
                value={notifications.autoresponder.emailField}
                onChange={(e) => updateNotifications({
                  autoresponder: { ...notifications.autoresponder, emailField: e.target.value }
                })}
                placeholder="email"
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
              <p className="text-xs text-gray-500 mt-1">
                Field name that contains the recipient's email
              </p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Subject
              </label>
              <input
                type="text"
                value={notifications.autoresponder.subject}
                onChange={(e) => updateNotifications({
                  autoresponder: { ...notifications.autoresponder, subject: e.target.value }
                })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Message Template
              </label>
              <textarea
                value={notifications.autoresponder.template}
                onChange={(e) => updateNotifications({
                  autoresponder: { ...notifications.autoresponder, template: e.target.value }
                })}
                rows={4}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
            </div>
          </div>
        )}
      </div>
    </div>
  );

  const renderSecuritySection = () => (
    <div className="space-y-6">
      <div className="space-y-4">
        <label className="flex items-center">
          <input
            type="checkbox"
            checked={settings.enableSpamProtection}
            onChange={(e) => updateSettings({ enableSpamProtection: e.target.checked })}
            className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
          />
          <span className="ml-2 text-sm text-gray-700">
            Enable spam protection
          </span>
        </label>

        <label className="flex items-center">
          <input
            type="checkbox"
            checked={settings.enableCaptcha}
            onChange={(e) => updateSettings({ enableCaptcha: e.target.checked })}
            className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
          />
          <span className="ml-2 text-sm text-gray-700">
            Enable CAPTCHA verification
          </span>
        </label>

        <label className="flex items-center">
          <input
            type="checkbox"
            checked={settings.requireAuthentication}
            onChange={(e) => updateSettings({ requireAuthentication: e.target.checked })}
            className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
          />
          <span className="ml-2 text-sm text-gray-700">
            Require user authentication
          </span>
        </label>
      </div>

      <div className="p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
        <h4 className="text-sm font-medium text-yellow-800 mb-2">Security Features</h4>
        <ul className="text-sm text-yellow-700 space-y-1">
          <li>• Rate limiting: 10 submissions per IP per hour</li>
          <li>• Automatic spam detection using AI</li>
          <li>• Honeypot fields for bot detection</li>
          <li>• SSL/TLS encryption for all data</li>
        </ul>
      </div>
    </div>
  );

  const renderIntegrationsSection = () => (
    <div className="space-y-6">
      <div className="text-center py-8 text-gray-500">
        <Webhook size={48} className="mx-auto mb-4 opacity-50" />
        <h3 className="text-lg font-medium text-gray-900 mb-2">Integrations</h3>
        <p className="text-sm">
          Connect your forms to external services like Zapier, Google Sheets, and more.
        </p>
        <p className="text-xs mt-2 text-gray-400">
          Available in the full implementation
        </p>
      </div>
    </div>
  );

  return (
    <motion.div
      initial={{ x: 400 }}
      animate={{ x: 0 }}
      exit={{ x: 400 }}
      className="w-80 bg-white border-l border-gray-200 flex flex-col h-full"
    >
      {/* Header */}
      <div className="p-4 border-b border-gray-200">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-gray-900">Form Settings</h2>
          <button
            onClick={onClose}
            className="p-1 text-gray-400 hover:text-gray-600"
          >
            <X size={20} />
          </button>
        </div>

        {/* Section Navigation */}
        <div className="space-y-1">
          {sections.map(section => (
            <button
              key={section.id}
              onClick={() => setActiveSection(section.id)}
              className={`
                w-full flex items-center space-x-3 px-3 py-2 text-sm font-medium rounded-lg transition-colors text-left
                ${activeSection === section.id 
                  ? 'bg-blue-100 text-blue-700' 
                  : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                }
              `}
            >
              {section.icon}
              <span>{section.name}</span>
            </button>
          ))}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4">
        <motion.div
          key={activeSection}
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.2 }}
        >
          {activeSection === 'general' && renderGeneralSection()}
          {activeSection === 'styling' && renderStylingSection()}
          {activeSection === 'notifications' && renderNotificationsSection()}
          {activeSection === 'security' && renderSecuritySection()}
          {activeSection === 'integrations' && renderIntegrationsSection()}
        </motion.div>
      </div>
    </motion.div>
  );
};

export default FormSettingsPanel;