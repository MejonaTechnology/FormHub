'use client';

import React from 'react';
import BaseField from './BaseField';
import { FormField } from '@/types/form-builder';
import { ChevronDown } from 'lucide-react';

interface SelectFieldProps {
  field: FormField;
  isSelected?: boolean;
  isPreview?: boolean;
  onSelect?: (fieldId: string) => void;
  onDelete?: (fieldId: string) => void;
  onDuplicate?: (fieldId: string) => void;
  onEdit?: (fieldId: string) => void;
  onChange?: (value: string | string[]) => void;
  value?: string | string[];
}

const SelectField: React.FC<SelectFieldProps> = ({
  field,
  isSelected,
  isPreview,
  onSelect,
  onDelete,
  onDuplicate,
  onEdit,
  onChange,
  value = '',
}) => {
  const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    if (onChange) {
      const newValue = e.target.value;
      onChange(newValue);
    }
  };

  const handleRadioChange = (optionValue: string) => {
    if (onChange) {
      onChange(optionValue);
    }
  };

  const handleCheckboxChange = (optionValue: string) => {
    if (onChange) {
      const currentValues = Array.isArray(value) ? value : [];
      const newValues = currentValues.includes(optionValue)
        ? currentValues.filter(v => v !== optionValue)
        : [...currentValues, optionValue];
      onChange(newValues);
    }
  };

  const renderSelect = () => {
    const selectClasses = `
      w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm
      focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500
      disabled:bg-gray-50 disabled:text-gray-500 disabled:cursor-not-allowed
      appearance-none bg-white
      transition-colors duration-200
      ${field.styling?.className || ''}
    `;

    return (
      <div className="relative">
        <select
          id={field.id}
          name={field.name}
          required={field.required}
          value={typeof value === 'string' ? value : ''}
          onChange={handleChange}
          disabled={!isPreview}
          className={selectClasses}
        >
          <option value="">
            {field.placeholder || `Select ${field.label?.toLowerCase() || 'option'}`}
          </option>
          {field.options?.map(option => (
            <option key={option.id} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
        <ChevronDown 
          className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 pointer-events-none" 
          size={16} 
        />
      </div>
    );
  };

  const renderRadio = () => {
    return (
      <div className="space-y-2">
        {field.options?.map(option => (
          <label key={option.id} className="flex items-center space-x-3 cursor-pointer">
            <input
              type="radio"
              name={field.name}
              value={option.value}
              checked={value === option.value}
              onChange={() => handleRadioChange(option.value)}
              disabled={!isPreview}
              className="w-4 h-4 text-blue-600 border-gray-300 focus:ring-blue-500 focus:ring-2 disabled:opacity-50"
            />
            <span className="text-sm text-gray-700">{option.label}</span>
          </label>
        ))}
      </div>
    );
  };

  const renderCheckbox = () => {
    const currentValues = Array.isArray(value) ? value : [];

    return (
      <div className="space-y-2">
        {field.options?.map(option => (
          <label key={option.id} className="flex items-center space-x-3 cursor-pointer">
            <input
              type="checkbox"
              name={`${field.name}[]`}
              value={option.value}
              checked={currentValues.includes(option.value)}
              onChange={() => handleCheckboxChange(option.value)}
              disabled={!isPreview}
              className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500 focus:ring-2 disabled:opacity-50"
            />
            <span className="text-sm text-gray-700">{option.label}</span>
          </label>
        ))}
      </div>
    );
  };

  const renderField = () => {
    switch (field.type) {
      case 'radio':
        return renderRadio();
      case 'checkbox':
        return renderCheckbox();
      default:
        return renderSelect();
    }
  };

  return (
    <BaseField
      field={field}
      isSelected={isSelected}
      isPreview={isPreview}
      onSelect={onSelect}
      onDelete={onDelete}
      onDuplicate={onDuplicate}
      onEdit={onEdit}
      className="mb-4"
    >
      {renderField()}
    </BaseField>
  );
};

export default SelectField;