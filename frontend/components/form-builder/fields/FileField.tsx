'use client';

import React, { useRef, useState } from 'react';
import BaseField from './BaseField';
import { FormField } from '@/types/form-builder';
import { Upload, File, Image, X, Download } from 'lucide-react';

interface FileFieldProps {
  field: FormField;
  isSelected?: boolean;
  isPreview?: boolean;
  onSelect?: (fieldId: string) => void;
  onDelete?: (fieldId: string) => void;
  onDuplicate?: (fieldId: string) => void;
  onEdit?: (fieldId: string) => void;
  onChange?: (files: FileList | null) => void;
  value?: File[];
}

const FileField: React.FC<FileFieldProps> = ({
  field,
  isSelected,
  isPreview,
  onSelect,
  onDelete,
  onDuplicate,
  onEdit,
  onChange,
  value = [],
}) => {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [dragOver, setDragOver] = useState(false);
  const [previewUrls, setPreviewUrls] = useState<string[]>([]);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (onChange) {
      onChange(files);
    }
    updatePreviews(files);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    
    if (!isPreview) return;
    
    const files = e.dataTransfer.files;
    if (fileInputRef.current) {
      fileInputRef.current.files = files;
    }
    if (onChange) {
      onChange(files);
    }
    updatePreviews(files);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(true);
  };

  const handleDragLeave = () => {
    setDragOver(false);
  };

  const updatePreviews = (files: FileList | null) => {
    if (!files || field.type !== 'image') return;

    const urls: string[] = [];
    Array.from(files).forEach(file => {
      if (file.type.startsWith('image/')) {
        urls.push(URL.createObjectURL(file));
      }
    });
    setPreviewUrls(urls);
  };

  const removeFile = (index: number) => {
    if (!fileInputRef.current || !isPreview) return;

    const dt = new DataTransfer();
    const files = fileInputRef.current.files;
    
    if (files) {
      Array.from(files).forEach((file, i) => {
        if (i !== index) {
          dt.items.add(file);
        }
      });
      fileInputRef.current.files = dt.files;
      if (onChange) {
        onChange(dt.files);
      }
      updatePreviews(dt.files);
    }
  };

  const openFileDialog = () => {
    if (fileInputRef.current && isPreview) {
      fileInputRef.current.click();
    }
  };

  const getAcceptedFileTypes = () => {
    if (field.type === 'image') {
      return 'image/*';
    }
    // Get from field validation or use default
    const acceptRule = field.validation?.find(rule => rule.type === 'pattern');
    return typeof acceptRule?.value === 'string' ? acceptRule.value : '*';
  };

  const getMaxFileSize = () => {
    const maxSizeRule = field.validation?.find(rule => rule.type === 'max');
    return maxSizeRule?.value ? Number(maxSizeRule.value) : undefined;
  };

  const getMaxFileSizeString = () => {
    const maxSize = getMaxFileSize();
    return maxSize ? formatFileSize(maxSize) : '10MB';
  };

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const renderFileList = () => {
    if (!value.length) return null;

    return (
      <div className="mt-3 space-y-2">
        {value.map((file, index) => (
          <div key={index} className="flex items-center justify-between p-2 bg-gray-50 rounded-lg border">
            <div className="flex items-center space-x-2">
              {field.type === 'image' && file.type.startsWith('image/') ? (
                <Image size={16} className="text-blue-500" />
              ) : (
                <File size={16} className="text-gray-500" />
              )}
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-700 truncate">{file.name}</p>
                <p className="text-xs text-gray-500">{formatFileSize(file.size)}</p>
              </div>
            </div>
            {isPreview && (
              <button
                onClick={() => removeFile(index)}
                className="p-1 text-gray-400 hover:text-red-500 transition-colors"
              >
                <X size={14} />
              </button>
            )}
          </div>
        ))}
      </div>
    );
  };

  const renderImagePreviews = () => {
    if (field.type !== 'image' || !previewUrls.length) return null;

    return (
      <div className="mt-3 grid grid-cols-2 gap-2">
        {previewUrls.map((url, index) => (
          <div key={index} className="relative">
            <img
              src={url}
              alt={`Preview ${index + 1}`}
              className="w-full h-24 object-cover rounded-lg border"
            />
            {isPreview && (
              <button
                onClick={() => removeFile(index)}
                className="absolute -top-1 -right-1 p-1 bg-red-500 text-white rounded-full hover:bg-red-600 transition-colors"
              >
                <X size={12} />
              </button>
            )}
          </div>
        ))}
      </div>
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
      <input
        ref={fileInputRef}
        type="file"
        id={field.id}
        name={field.name}
        multiple={field.validation?.some(rule => rule.type === 'custom' && rule.value === 'multiple')}
        accept={getAcceptedFileTypes()}
        onChange={handleFileChange}
        disabled={!isPreview}
        className="hidden"
      />
      
      <div
        onClick={openFileDialog}
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        className={`
          relative border-2 border-dashed rounded-lg p-6 text-center cursor-pointer
          transition-all duration-200
          ${dragOver 
            ? 'border-blue-400 bg-blue-50' 
            : 'border-gray-300 hover:border-blue-400 hover:bg-gray-50'
          }
          ${!isPreview ? 'cursor-default opacity-75' : ''}
        `}
      >
        <div className="flex flex-col items-center space-y-2">
          {field.type === 'image' ? (
            <Image size={24} className="text-gray-400" />
          ) : (
            <Upload size={24} className="text-gray-400" />
          )}
          
          <div className="text-sm text-gray-600">
            <span className="font-medium text-blue-600 hover:text-blue-500">
              {isPreview ? 'Click to upload' : 'File upload area'}
            </span>
            {isPreview && ' or drag and drop'}
          </div>
          
          <div className="text-xs text-gray-400">
            {field.type === 'image' ? 'PNG, JPG, GIF up to' : 'Files up to'} {getMaxFileSizeString()}
          </div>
        </div>

        {dragOver && (
          <div className="absolute inset-0 bg-blue-500 bg-opacity-10 border-2 border-blue-500 border-dashed rounded-lg flex items-center justify-center">
            <p className="text-blue-600 font-medium">Drop files here</p>
          </div>
        )}
      </div>

      {renderFileList()}
      {renderImagePreviews()}
    </BaseField>
  );
};

export default FileField;