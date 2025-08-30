'use client';

import React, { useState } from 'react';
import BaseField from './BaseField';
import { FormField } from '@/types/form-builder';
import { Star, Minus, Edit3, Code } from 'lucide-react';

interface AdvancedFieldProps {
  field: FormField;
  isSelected?: boolean;
  isPreview?: boolean;
  onSelect?: (fieldId: string) => void;
  onDelete?: (fieldId: string) => void;
  onDuplicate?: (fieldId: string) => void;
  onEdit?: (fieldId: string) => void;
  onChange?: (value: any) => void;
  value?: any;
}

const AdvancedFields: React.FC<AdvancedFieldProps> = ({
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
  const [signatureDrawing, setSignatureDrawing] = useState(false);
  const [signatureRef, setSignatureRef] = useState<HTMLCanvasElement | null>(null);

  // Rating Field Component
  const renderRating = () => {
    const maxRating = 5; // Could be configurable
    const currentRating = typeof value === 'number' ? value : 0;

    return (
      <div className="flex items-center space-x-1">
        {[...Array(maxRating)].map((_, index) => (
          <button
            key={index}
            type="button"
            onClick={() => isPreview && onChange && onChange(index + 1)}
            disabled={!isPreview}
            className={`
              w-8 h-8 focus:outline-none focus:ring-2 focus:ring-blue-500 rounded
              transition-colors duration-150
              ${index < currentRating 
                ? 'text-yellow-400 hover:text-yellow-500' 
                : 'text-gray-300 hover:text-yellow-300'
              }
              ${!isPreview ? 'cursor-default' : 'cursor-pointer'}
            `}
          >
            <Star size={20} fill="currentColor" />
          </button>
        ))}
        <span className="ml-2 text-sm text-gray-600">
          {currentRating > 0 ? `${currentRating} / ${maxRating}` : 'No rating'}
        </span>
      </div>
    );
  };

  // Signature Field Component
  const renderSignature = () => {
    const startDrawing = (e: React.MouseEvent<HTMLCanvasElement>) => {
      if (!isPreview || !signatureRef) return;
      setSignatureDrawing(true);
      const rect = signatureRef.getBoundingClientRect();
      const ctx = signatureRef.getContext('2d');
      if (ctx) {
        ctx.beginPath();
        ctx.moveTo(e.clientX - rect.left, e.clientY - rect.top);
      }
    };

    const draw = (e: React.MouseEvent<HTMLCanvasElement>) => {
      if (!signatureDrawing || !signatureRef) return;
      const rect = signatureRef.getBoundingClientRect();
      const ctx = signatureRef.getContext('2d');
      if (ctx) {
        ctx.lineTo(e.clientX - rect.left, e.clientY - rect.top);
        ctx.stroke();
      }
    };

    const stopDrawing = () => {
      if (!signatureRef) return;
      setSignatureDrawing(false);
      // Save signature data
      const dataURL = signatureRef.toDataURL();
      if (onChange) onChange(dataURL);
    };

    const clearSignature = () => {
      if (!signatureRef) return;
      const ctx = signatureRef.getContext('2d');
      if (ctx) {
        ctx.clearRect(0, 0, signatureRef.width, signatureRef.height);
      }
      if (onChange) onChange('');
    };

    return (
      <div className="space-y-3">
        <div className="border-2 border-dashed border-gray-300 rounded-lg overflow-hidden">
          <canvas
            ref={setSignatureRef}
            width={400}
            height={200}
            onMouseDown={startDrawing}
            onMouseMove={draw}
            onMouseUp={stopDrawing}
            onMouseLeave={stopDrawing}
            className={`
              w-full h-32 bg-white
              ${isPreview ? 'cursor-crosshair' : 'cursor-not-allowed opacity-75'}
            `}
            style={{ touchAction: 'none' }}
          />
        </div>
        {isPreview && (
          <div className="flex justify-between items-center">
            <span className="text-sm text-gray-500">
              {value ? 'Signature captured' : 'Sign above'}
            </span>
            <button
              onClick={clearSignature}
              className="px-3 py-1 text-sm text-red-600 border border-red-200 rounded hover:bg-red-50 transition-colors"
            >
              Clear
            </button>
          </div>
        )}
      </div>
    );
  };

  // Divider Field Component
  const renderDivider = () => {
    const dividerStyle = field.styling || {};
    
    return (
      <div className="py-4">
        <hr 
          className="border-gray-300"
          style={{
            borderColor: dividerStyle.borderColor,
            borderWidth: dividerStyle.borderWidth || '1px',
          }}
        />
      </div>
    );
  };

  // Heading Field Component
  const renderHeading = () => {
    const headingLevel = field.validation?.find(rule => rule.type === 'custom')?.value as string || 'h2';
    const HeadingTag = ['h1', 'h2', 'h3', 'h4', 'h5', 'h6'].includes(headingLevel) ? headingLevel as 'h1' | 'h2' | 'h3' | 'h4' | 'h5' | 'h6' : 'h2';
    const headingClasses = `
      font-bold text-gray-900 mb-2
      ${HeadingTag === 'h1' ? 'text-3xl' : ''}
      ${HeadingTag === 'h2' ? 'text-2xl' : ''}
      ${HeadingTag === 'h3' ? 'text-xl' : ''}
      ${HeadingTag === 'h4' ? 'text-lg' : ''}
      ${HeadingTag === 'h5' ? 'text-base' : ''}
      ${HeadingTag === 'h6' ? 'text-sm' : ''}
    `;

    return React.createElement(
      HeadingTag,
      { className: headingClasses },
      field.label || 'Heading Text'
    );
  };

  // Paragraph Field Component
  const renderParagraph = () => {
    return (
      <div className="prose prose-sm max-w-none">
        <p className="text-gray-700 leading-relaxed">
          {field.description || field.label || 'Paragraph text goes here. You can add multiple sentences and format this text as needed.'}
        </p>
      </div>
    );
  };

  // HTML Field Component
  const renderHtml = () => {
    const htmlContent = field.validation?.find(rule => rule.type === 'custom')?.value || 
      '<div class="p-4 bg-gray-100 rounded-lg"><p>Custom HTML content</p></div>';

    if (!isPreview) {
      return (
        <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
          <div className="flex items-center space-x-2 mb-2">
            <Code size={16} className="text-gray-500" />
            <span className="text-sm text-gray-600">HTML Content</span>
          </div>
          <pre className="text-xs text-gray-700 overflow-x-auto">
            {htmlContent}
          </pre>
        </div>
      );
    }

    return (
      <div 
        dangerouslySetInnerHTML={{ __html: htmlContent }}
        className="html-content"
      />
    );
  };

  const renderFieldContent = () => {
    switch (field.type) {
      case 'rating':
        return renderRating();
      case 'signature':
        return renderSignature();
      case 'divider':
        return renderDivider();
      case 'heading':
        return renderHeading();
      case 'paragraph':
        return renderParagraph();
      case 'html':
        return renderHtml();
      default:
        return (
          <div className="p-4 bg-gray-50 border border-gray-200 rounded-lg text-center">
            <p className="text-sm text-gray-500">Unsupported field type: {field.type}</p>
          </div>
        );
    }
  };

  // Layout fields don't need the BaseField wrapper in the same way
  if (['divider', 'heading', 'paragraph', 'html'].includes(field.type)) {
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
        {renderFieldContent()}
      </BaseField>
    );
  }

  // Interactive fields need full BaseField functionality
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
      {renderFieldContent()}
    </BaseField>
  );
};

export default AdvancedFields;