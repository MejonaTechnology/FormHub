'use client';

import React from 'react';
import { FormField } from '@/types/form-builder';
import InputField from './InputField';
import SelectField from './SelectField';
import FileField from './FileField';
import AdvancedFields from './AdvancedFields';

interface FieldRendererProps {
  field: FormField;
  isSelected?: boolean;
  isPreview?: boolean;
  onSelect?: (fieldId: string) => void;
  onDelete?: (fieldId: string) => void;
  onDuplicate?: (fieldId: string) => void;
  onEdit?: (fieldId: string) => void;
  onChange?: (fieldId: string, value: any) => void;
  value?: any;
}

const FieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  isSelected,
  isPreview,
  onSelect,
  onDelete,
  onDuplicate,
  onEdit,
  onChange,
  value,
}) => {
  const handleFieldChange = (newValue: any) => {
    if (onChange) {
      onChange(field.id, newValue);
    }
  };

  // Check if field should be shown based on conditional logic
  const shouldShowField = () => {
    if (!field.conditionalLogic || !field.conditionalLogic.conditions.length) {
      return true;
    }

    // In a real implementation, this would evaluate against form data
    // For now, we'll show the field based on the show property
    return field.conditionalLogic.show;
  };

  if (!shouldShowField() && isPreview) {
    return null;
  }

  // Input-type fields
  if ([
    'text',
    'email',
    'password',
    'textarea',
    'number',
    'phone',
    'url',
    'date',
    'datetime-local',
    'time',
    'color',
    'range'
  ].includes(field.type)) {
    return (
      <InputField
        field={field}
        isSelected={isSelected}
        isPreview={isPreview}
        onSelect={onSelect}
        onDelete={onDelete}
        onDuplicate={onDuplicate}
        onEdit={onEdit}
        onChange={handleFieldChange}
        value={value}
      />
    );
  }

  // Choice fields
  if (['select', 'radio', 'checkbox'].includes(field.type)) {
    return (
      <SelectField
        field={field}
        isSelected={isSelected}
        isPreview={isPreview}
        onSelect={onSelect}
        onDelete={onDelete}
        onDuplicate={onDuplicate}
        onEdit={onEdit}
        onChange={handleFieldChange}
        value={value}
      />
    );
  }

  // File fields
  if (['file', 'image'].includes(field.type)) {
    return (
      <FileField
        field={field}
        isSelected={isSelected}
        isPreview={isPreview}
        onSelect={onSelect}
        onDelete={onDelete}
        onDuplicate={onDuplicate}
        onEdit={onEdit}
        onChange={handleFieldChange}
        value={value}
      />
    );
  }

  // Advanced fields
  if ([
    'rating',
    'signature',
    'divider',
    'heading',
    'paragraph',
    'html'
  ].includes(field.type)) {
    return (
      <AdvancedFields
        field={field}
        isSelected={isSelected}
        isPreview={isPreview}
        onSelect={onSelect}
        onDelete={onDelete}
        onDuplicate={onDuplicate}
        onEdit={onEdit}
        onChange={handleFieldChange}
        value={value}
      />
    );
  }

  // Fallback for unknown field types
  return (
    <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-center">
      <p className="text-sm text-red-600">
        Unknown field type: <code className="bg-red-100 px-1 rounded">{field.type}</code>
      </p>
      <p className="text-xs text-red-500 mt-1">
        This field type is not supported by the form builder.
      </p>
    </div>
  );
};

export default FieldRenderer;