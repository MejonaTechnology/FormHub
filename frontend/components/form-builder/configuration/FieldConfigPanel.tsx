'use client';

import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  X, 
  Settings, 
  Type, 
  Palette, 
  Eye, 
  EyeOff, 
  Plus, 
  Trash2, 
  ChevronDown,
  ChevronRight,
  AlertTriangle 
} from 'lucide-react';
import { HexColorPicker } from 'react-colorful';
import { FormField, ValidationRule, FieldOption, ConditionalLogic } from '@/types/form-builder';

interface FieldConfigPanelProps {
  field: FormField | null;
  allFields: FormField[];
  onFieldUpdate: (field: FormField) => void;
  onClose: () => void;
}

const FieldConfigPanel: React.FC<FieldConfigPanelProps> = ({
  field,
  allFields,
  onFieldUpdate,
  onClose,
}) => {
  const [activeTab, setActiveTab] = useState<'general' | 'validation' | 'styling' | 'logic'>('general');
  const [showColorPicker, setShowColorPicker] = useState<string | null>(null);
  const [expandedSections, setExpandedSections] = useState<string[]>(['basic']);

  if (!field) return null;

  const toggleSection = (section: string) => {
    setExpandedSections(prev => 
      prev.includes(section) 
        ? prev.filter(s => s !== section)
        : [...prev, section]
    );
  };

  const updateField = (updates: Partial<FormField>) => {
    onFieldUpdate({ ...field, ...updates });
  };

  const updateFieldValidation = (rules: ValidationRule[]) => {
    updateField({ validation: rules });
  };

  const addValidationRule = () => {
    const newRule: ValidationRule = {
      type: 'required',
      message: 'This field is required',
    };
    const currentRules = field.validation || [];
    updateFieldValidation([...currentRules, newRule]);
  };

  const removeValidationRule = (index: number) => {
    const currentRules = field.validation || [];
    updateFieldValidation(currentRules.filter((_, i) => i !== index));
  };

  const updateValidationRule = (index: number, updates: Partial<ValidationRule>) => {
    const currentRules = field.validation || [];
    const newRules = currentRules.map((rule, i) => 
      i === index ? { ...rule, ...updates } : rule
    );
    updateFieldValidation(newRules);
  };

  const addOption = () => {
    const newOption: FieldOption = {
      id: `option_${Date.now()}`,
      label: 'New Option',
      value: `option_${Date.now()}`,
    };
    const currentOptions = field.options || [];
    updateField({ options: [...currentOptions, newOption] });
  };

  const removeOption = (index: number) => {
    const currentOptions = field.options || [];
    updateField({ options: currentOptions.filter((_, i) => i !== index) });
  };

  const updateOption = (index: number, updates: Partial<FieldOption>) => {
    const currentOptions = field.options || [];
    const newOptions = currentOptions.map((option, i) => 
      i === index ? { ...option, ...updates } : option
    );
    updateField({ options: newOptions });
  };

  const updateStyling = (styleUpdates: Partial<typeof field.styling>) => {
    updateField({ 
      styling: { 
        ...field.styling, 
        ...styleUpdates 
      }
    });
  };

  const tabs = [
    { id: 'general', name: 'General', icon: <Settings size={16} /> },
    { id: 'validation', name: 'Validation', icon: <AlertTriangle size={16} /> },
    { id: 'styling', name: 'Styling', icon: <Palette size={16} /> },
    { id: 'logic', name: 'Logic', icon: <Eye size={16} /> },
  ];

  const renderGeneralTab = () => (
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
          Basic Settings
        </button>
        
        {expandedSections.includes('basic') && (
          <div className="space-y-4 pl-6">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Field Label
              </label>
              <input
                type="text"
                value={field.label}
                onChange={(e) => updateField({ label: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Field Name
              </label>
              <input
                type="text"
                value={field.name}
                onChange={(e) => updateField({ name: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
              <p className="text-xs text-gray-500 mt-1">
                Used in form submission data
              </p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Placeholder
              </label>
              <input
                type="text"
                value={field.placeholder || ''}
                onChange={(e) => updateField({ placeholder: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Description
              </label>
              <textarea
                value={field.description || ''}
                onChange={(e) => updateField({ description: e.target.value })}
                rows={3}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
            </div>

            <div className="flex items-center">
              <input
                type="checkbox"
                id="required"
                checked={field.required}
                onChange={(e) => updateField({ required: e.target.checked })}
                className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
              />
              <label htmlFor="required" className="ml-2 text-sm text-gray-700">
                Required field
              </label>
            </div>
          </div>
        )}
      </div>

      {/* Options (for select, radio, checkbox fields) */}
      {['select', 'radio', 'checkbox'].includes(field.type) && (
        <div>
          <button
            onClick={() => toggleSection('options')}
            className="flex items-center w-full text-left text-sm font-medium text-gray-900 mb-3"
          >
            {expandedSections.includes('options') ? (
              <ChevronDown size={16} className="mr-2" />
            ) : (
              <ChevronRight size={16} className="mr-2" />
            )}
            Options
          </button>
          
          {expandedSections.includes('options') && (
            <div className="pl-6">
              <div className="space-y-2 mb-3">
                {(field.options || []).map((option, index) => (
                  <div key={option.id} className="flex items-center space-x-2">
                    <input
                      type="text"
                      value={option.label}
                      onChange={(e) => updateOption(index, { label: e.target.value })}
                      placeholder="Option label"
                      className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    />
                    <input
                      type="text"
                      value={option.value}
                      onChange={(e) => updateOption(index, { value: e.target.value })}
                      placeholder="Value"
                      className="w-24 px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    />
                    <button
                      onClick={() => removeOption(index)}
                      className="p-2 text-red-500 hover:text-red-700"
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                ))}
              </div>
              
              <button
                onClick={addOption}
                className="flex items-center space-x-2 text-sm text-blue-600 hover:text-blue-700"
              >
                <Plus size={16} />
                <span>Add Option</span>
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );

  const renderValidationTab = () => (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-medium text-gray-900">Validation Rules</h4>
        <button
          onClick={addValidationRule}
          className="flex items-center space-x-1 text-sm text-blue-600 hover:text-blue-700"
        >
          <Plus size={16} />
          <span>Add Rule</span>
        </button>
      </div>

      <div className="space-y-3">
        {(field.validation || []).map((rule, index) => (
          <div key={index} className="p-3 border border-gray-200 rounded-lg">
            <div className="flex items-center justify-between mb-2">
              <select
                value={rule.type}
                onChange={(e) => updateValidationRule(index, { type: e.target.value as any })}
                className="text-sm border border-gray-300 rounded px-2 py-1"
              >
                <option value="required">Required</option>
                <option value="minLength">Minimum Length</option>
                <option value="maxLength">Maximum Length</option>
                <option value="min">Minimum Value</option>
                <option value="max">Maximum Value</option>
                <option value="pattern">Pattern (Regex)</option>
                <option value="email">Email Format</option>
                <option value="url">URL Format</option>
                <option value="custom">Custom</option>
              </select>
              <button
                onClick={() => removeValidationRule(index)}
                className="text-red-500 hover:text-red-700"
              >
                <Trash2 size={16} />
              </button>
            </div>

            {['minLength', 'maxLength', 'min', 'max', 'pattern'].includes(rule.type) && (
              <input
                type={['min', 'max'].includes(rule.type) ? 'number' : 'text'}
                value={rule.value || ''}
                onChange={(e) => updateValidationRule(index, { value: e.target.value })}
                placeholder={`Enter ${rule.type} value`}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500 mb-2"
              />
            )}

            <input
              type="text"
              value={rule.message}
              onChange={(e) => updateValidationRule(index, { message: e.target.value })}
              placeholder="Error message"
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>
        ))}
      </div>
    </div>
  );

  const renderStylingTab = () => (
    <div className="space-y-6">
      {/* Colors */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 mb-3">Colors</h4>
        <div className="space-y-3">
          {[
            { key: 'backgroundColor', label: 'Background Color' },
            { key: 'textColor', label: 'Text Color' },
            { key: 'borderColor', label: 'Border Color' },
          ].map(({ key, label }) => (
            <div key={key} className="flex items-center space-x-3">
              <label className="text-sm text-gray-700 w-24">{label}</label>
              <button
                onClick={() => setShowColorPicker(showColorPicker === key ? null : key)}
                className="w-8 h-8 rounded border border-gray-300"
                style={{ backgroundColor: (field.styling as any)?.[key] || '#ffffff' }}
              />
              <input
                type="text"
                value={(field.styling as any)?.[key] || ''}
                onChange={(e) => updateStyling({ [key]: e.target.value })}
                placeholder="#ffffff"
                className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
              
              {showColorPicker === key && (
                <div className="absolute z-10 mt-2">
                  <div className="fixed inset-0" onClick={() => setShowColorPicker(null)} />
                  <div className="relative bg-white p-3 rounded-lg shadow-lg border">
                    <HexColorPicker
                      color={(field.styling as any)?.[key] || '#ffffff'}
                      onChange={(color) => updateStyling({ [key]: color })}
                    />
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Dimensions */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 mb-3">Dimensions</h4>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs text-gray-600 mb-1">Width</label>
            <input
              type="text"
              value={field.styling?.width || ''}
              onChange={(e) => updateStyling({ width: e.target.value })}
              placeholder="100%"
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-600 mb-1">Padding</label>
            <input
              type="text"
              value={field.styling?.padding || ''}
              onChange={(e) => updateStyling({ padding: e.target.value })}
              placeholder="8px"
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>
        </div>
      </div>

      {/* Typography */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 mb-3">Typography</h4>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs text-gray-600 mb-1">Font Size</label>
            <input
              type="text"
              value={field.styling?.fontSize || ''}
              onChange={(e) => updateStyling({ fontSize: e.target.value })}
              placeholder="14px"
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-600 mb-1">Font Weight</label>
            <select
              value={field.styling?.fontWeight || ''}
              onChange={(e) => updateStyling({ fontWeight: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="">Default</option>
              <option value="300">Light</option>
              <option value="400">Normal</option>
              <option value="500">Medium</option>
              <option value="600">Semi Bold</option>
              <option value="700">Bold</option>
            </select>
          </div>
        </div>
      </div>

      {/* Custom CSS */}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">Custom CSS</label>
        <textarea
          value={field.styling?.customCSS || ''}
          onChange={(e) => updateStyling({ customCSS: e.target.value })}
          placeholder="/* Custom CSS rules */"
          rows={4}
          className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500 font-mono text-sm"
        />
      </div>
    </div>
  );

  const renderLogicTab = () => (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-medium text-gray-900">Conditional Logic</h4>
        <div className="flex items-center">
          <input
            type="checkbox"
            id="enable-logic"
            checked={!!field.conditionalLogic}
            onChange={(e) => {
              if (e.target.checked) {
                updateField({
                  conditionalLogic: {
                    show: true,
                    conditions: [],
                    logic: 'and'
                  }
                });
              } else {
                updateField({ conditionalLogic: undefined });
              }
            }}
            className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
          />
          <label htmlFor="enable-logic" className="ml-2 text-sm text-gray-700">
            Enable
          </label>
        </div>
      </div>

      {field.conditionalLogic && (
        <div className="space-y-4 p-4 bg-gray-50 rounded-lg">
          <div className="flex items-center space-x-2">
            <select
              value={field.conditionalLogic.show ? 'show' : 'hide'}
              onChange={(e) => updateField({
                conditionalLogic: {
                  ...field.conditionalLogic!,
                  show: e.target.value === 'show'
                }
              })}
              className="px-3 py-2 border border-gray-300 rounded-md"
            >
              <option value="show">Show</option>
              <option value="hide">Hide</option>
            </select>
            <span className="text-sm text-gray-600">this field when</span>
            <select
              value={field.conditionalLogic.logic}
              onChange={(e) => updateField({
                conditionalLogic: {
                  ...field.conditionalLogic!,
                  logic: e.target.value as 'and' | 'or'
                }
              })}
              className="px-3 py-2 border border-gray-300 rounded-md"
            >
              <option value="and">All</option>
              <option value="or">Any</option>
            </select>
            <span className="text-sm text-gray-600">conditions are met:</span>
          </div>

          <div className="text-sm text-gray-500">
            <p>Conditional logic will be fully implemented in the next update.</p>
            <p>This feature will allow you to show/hide fields based on other field values.</p>
          </div>
        </div>
      )}
    </div>
  );

  return (
    <AnimatePresence>
      <motion.div
        initial={{ x: 400 }}
        animate={{ x: 0 }}
        exit={{ x: 400 }}
        className="w-80 bg-white border-l border-gray-200 flex flex-col h-full"
      >
        {/* Header */}
        <div className="p-4 border-b border-gray-200">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-gray-900">Field Settings</h2>
            <button
              onClick={onClose}
              className="p-1 text-gray-400 hover:text-gray-600"
            >
              <X size={20} />
            </button>
          </div>

          {/* Tabs */}
          <div className="flex space-x-1 bg-gray-100 rounded-lg p-1">
            {tabs.map(tab => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id as any)}
                className={`
                  flex-1 flex items-center justify-center space-x-1 px-3 py-2 text-xs font-medium rounded-md transition-colors
                  ${activeTab === tab.id 
                    ? 'bg-white text-gray-900 shadow-sm' 
                    : 'text-gray-600 hover:text-gray-900'
                  }
                `}
              >
                {tab.icon}
                <span className="hidden sm:inline">{tab.name}</span>
              </button>
            ))}
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4">
          <AnimatePresence mode="wait">
            <motion.div
              key={activeTab}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              transition={{ duration: 0.2 }}
            >
              {activeTab === 'general' && renderGeneralTab()}
              {activeTab === 'validation' && renderValidationTab()}
              {activeTab === 'styling' && renderStylingTab()}
              {activeTab === 'logic' && renderLogicTab()}
            </motion.div>
          </AnimatePresence>
        </div>
      </motion.div>
    </AnimatePresence>
  );
};

export default FieldConfigPanel;