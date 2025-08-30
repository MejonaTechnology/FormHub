'use client';

import React from 'react';
import { motion } from 'framer-motion';
import { Trash2, Settings, Copy, Eye, EyeOff } from 'lucide-react';
import { FormField, FieldStyling } from '@/types/form-builder';

interface BaseFieldProps {
  field: FormField;
  isSelected?: boolean;
  isPreview?: boolean;
  onSelect?: (fieldId: string) => void;
  onDelete?: (fieldId: string) => void;
  onDuplicate?: (fieldId: string) => void;
  onEdit?: (fieldId: string) => void;
  children: React.ReactNode;
  className?: string;
}

const BaseField: React.FC<BaseFieldProps> = ({
  field,
  isSelected = false,
  isPreview = false,
  onSelect,
  onDelete,
  onDuplicate,
  onEdit,
  children,
  className = '',
}) => {
  const handleClick = (e: React.MouseEvent) => {
    if (!isPreview && onSelect) {
      e.stopPropagation();
      onSelect(field.id);
    }
  };

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onDelete) onDelete(field.id);
  };

  const handleDuplicate = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onDuplicate) onDuplicate(field.id);
  };

  const handleEdit = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onEdit) onEdit(field.id);
  };

  const fieldStyling = field.styling || {};
  const customStyles: React.CSSProperties = {
    backgroundColor: fieldStyling.backgroundColor,
    color: fieldStyling.textColor,
    borderColor: fieldStyling.borderColor,
    borderWidth: fieldStyling.borderWidth,
    borderRadius: fieldStyling.borderRadius,
    fontSize: fieldStyling.fontSize,
    fontWeight: fieldStyling.fontWeight,
    padding: fieldStyling.padding,
    margin: fieldStyling.margin,
    width: fieldStyling.width,
  };

  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -20 }}
      whileHover={!isPreview ? { scale: 1.01 } : {}}
      className={`
        relative group transition-all duration-200
        ${!isPreview ? 'cursor-pointer' : ''}
        ${isSelected && !isPreview ? 'ring-2 ring-blue-500 ring-opacity-50' : ''}
        ${className}
      `}
      style={customStyles}
      onClick={handleClick}
    >
      {/* Field Content */}
      <div className="relative">
        {/* Field Label */}
        {field.label && (
          <label className="block text-sm font-medium text-gray-700 mb-2">
            {field.label}
            {field.required && <span className="text-red-500 ml-1">*</span>}
          </label>
        )}

        {/* Field Description */}
        {field.description && (
          <p className="text-sm text-gray-500 mb-2">{field.description}</p>
        )}

        {/* Field Component */}
        {children}

        {/* Field Actions (only in builder mode) */}
        {!isPreview && isSelected && (
          <motion.div
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: 1, scale: 1 }}
            className="absolute -top-2 -right-2 flex space-x-1 bg-white border border-gray-200 rounded-lg shadow-lg p-1"
          >
            <button
              onClick={handleEdit}
              className="p-1 text-gray-500 hover:text-blue-600 hover:bg-blue-50 rounded transition-colors"
              title="Edit field"
            >
              <Settings size={12} />
            </button>
            <button
              onClick={handleDuplicate}
              className="p-1 text-gray-500 hover:text-green-600 hover:bg-green-50 rounded transition-colors"
              title="Duplicate field"
            >
              <Copy size={12} />
            </button>
            <button
              onClick={handleDelete}
              className="p-1 text-gray-500 hover:text-red-600 hover:bg-red-50 rounded transition-colors"
              title="Delete field"
            >
              <Trash2 size={12} />
            </button>
          </motion.div>
        )}

        {/* Conditional Logic Indicator */}
        {field.conditionalLogic && !isPreview && (
          <div className="absolute -top-1 -left-1">
            <div className="flex items-center justify-center w-4 h-4 bg-purple-500 text-white text-xs rounded-full">
              {field.conditionalLogic.show ? <Eye size={10} /> : <EyeOff size={10} />}
            </div>
          </div>
        )}

        {/* Custom CSS */}
        {fieldStyling.customCSS && (
          <style jsx>{`${fieldStyling.customCSS}`}</style>
        )}
      </div>
    </motion.div>
  );
};

export default BaseField;