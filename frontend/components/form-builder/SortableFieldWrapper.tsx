'use client';

import React from 'react';
import {
  useSortable,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { FormField } from '@/types/form-builder';
import { GripVertical } from 'lucide-react';

interface SortableFieldWrapperProps {
  field: FormField;
  children: React.ReactNode;
  isPreviewMode?: boolean;
  disabled?: boolean;
}

const SortableFieldWrapper: React.FC<SortableFieldWrapperProps> = ({
  field,
  children,
  isPreviewMode = false,
  disabled = false,
}) => {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ 
    id: field.id,
    disabled: disabled || isPreviewMode,
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    zIndex: isDragging ? 1000 : 'auto',
  };

  if (isPreviewMode || disabled) {
    return (
      <div ref={setNodeRef} style={style}>
        {children}
      </div>
    );
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`
        relative group
        ${isDragging ? 'opacity-50' : ''}
      `}
    >
      {/* Drag Handle */}
      <div
        {...attributes}
        {...listeners}
        className="
          absolute -left-8 top-2 opacity-0 group-hover:opacity-100
          flex items-center justify-center w-6 h-6 
          text-gray-400 hover:text-gray-600 cursor-grab active:cursor-grabbing
          transition-opacity duration-200 z-10
        "
        title="Drag to reorder"
      >
        <GripVertical size={16} />
      </div>
      
      {children}
    </div>
  );
};

export default SortableFieldWrapper;