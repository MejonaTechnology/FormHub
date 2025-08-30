'use client';

import React from 'react';
import BaseField from './BaseField';
import { FormField } from '@/types/form-builder';

interface InputFieldProps {
  field: FormField;
  isSelected?: boolean;
  isPreview?: boolean;
  onSelect?: (fieldId: string) => void;
  onDelete?: (fieldId: string) => void;
  onDuplicate?: (fieldId: string) => void;
  onEdit?: (fieldId: string) => void;
  onChange?: (value: string) => void;
  value?: string;
}

const InputField: React.FC<InputFieldProps> = ({
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
  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (onChange) {
      onChange(e.target.value);
    }
  };

  const getInputType = () => {
    switch (field.type) {
      case 'email':
        return 'email';
      case 'password':
        return 'password';
      case 'number':
        return 'number';
      case 'phone':
        return 'tel';
      case 'url':
        return 'url';
      case 'date':
        return 'date';
      case 'datetime-local':
        return 'datetime-local';
      case 'time':
        return 'time';
      case 'color':
        return 'color';
      case 'range':
        return 'range';
      default:
        return 'text';
    }
  };

  const getInputAttributes = () => {
    const attrs: any = {
      type: getInputType(),
      id: field.id,
      name: field.name,
      placeholder: field.placeholder,
      required: field.required,
      value: value,
      onChange: handleChange,
      disabled: !isPreview,
    };

    // Add validation attributes
    if (field.validation) {
      field.validation.forEach(rule => {
        switch (rule.type) {
          case 'minLength':
            attrs.minLength = rule.value;
            break;
          case 'maxLength':
            attrs.maxLength = rule.value;
            break;
          case 'min':
            attrs.min = rule.value;
            break;
          case 'max':
            attrs.max = rule.value;
            break;
          case 'pattern':
            attrs.pattern = rule.value;
            break;
        }
      });
    }

    return attrs;
  };

  const renderInput = () => {
    const inputClasses = `
      w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm
      focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500
      disabled:bg-gray-50 disabled:text-gray-500 disabled:cursor-not-allowed
      transition-colors duration-200
      ${field.styling?.className || ''}
    `;

    if (field.type === 'textarea') {
      return (
        <textarea
          {...getInputAttributes()}
          rows={4}
          className={inputClasses}
        />
      );
    }

    if (field.type === 'range') {
      return (
        <div className="space-y-2">
          <input
            {...getInputAttributes()}
            className="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer slider"
          />
          <div className="flex justify-between text-sm text-gray-500">
            <span>{getInputAttributes().min || 0}</span>
            <span className="font-medium">{value}</span>
            <span>{getInputAttributes().max || 100}</span>
          </div>
        </div>
      );
    }

    if (field.type === 'color') {
      return (
        <div className="flex items-center space-x-3">
          <input
            {...getInputAttributes()}
            className="w-12 h-10 border border-gray-300 rounded cursor-pointer"
          />
          <input
            type="text"
            value={value}
            onChange={handleChange}
            placeholder="#000000"
            className={inputClasses.replace('w-full', 'flex-1')}
          />
        </div>
      );
    }

    return (
      <input
        {...getInputAttributes()}
        className={inputClasses}
      />
    );
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
      {renderInput()}
    </BaseField>
  );
};

export default InputField;