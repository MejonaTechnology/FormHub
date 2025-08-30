'use client';

import React, { useState, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import {
  DndContext,
  DragEndEvent,
  DragOverEvent,
  DragStartEvent,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
// import { restrictToVerticalAxis } from '@dnd-kit/modifiers';

import { FormField, PaletteComponent } from '@/types/form-builder';
import FieldRenderer from './fields/FieldRenderer';
import SortableFieldWrapper from './SortableFieldWrapper';
import { Plus, Grid, Eye, Settings, Smartphone, Tablet, Monitor } from 'lucide-react';

interface FormCanvasProps {
  fields: FormField[];
  selectedFieldId?: string;
  isPreviewMode?: boolean;
  onFieldsChange: (fields: FormField[]) => void;
  onFieldSelect: (fieldId: string) => void;
  onFieldDelete: (fieldId: string) => void;
  onFieldDuplicate: (fieldId: string) => void;
  onFieldEdit: (fieldId: string) => void;
  onPreviewToggle: () => void;
  formValues?: Record<string, any>;
  onFormValueChange?: (fieldId: string, value: any) => void;
}

type ViewMode = 'desktop' | 'tablet' | 'mobile';

const FormCanvas: React.FC<FormCanvasProps> = ({
  fields,
  selectedFieldId,
  isPreviewMode,
  onFieldsChange,
  onFieldSelect,
  onFieldDelete,
  onFieldDuplicate,
  onFieldEdit,
  onPreviewToggle,
  formValues = {},
  onFormValueChange,
}) => {
  const [viewMode, setViewMode] = useState<ViewMode>('desktop');
  const [showGrid, setShowGrid] = useState(false);
  const [isDragOver, setIsDragOver] = useState(false);

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8,
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  const generateFieldId = () => {
    return `field_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  };

  const createFieldFromComponent = useCallback((component: PaletteComponent): FormField => {
    return {
      ...component.defaultProps,
      id: generateFieldId(),
      type: component.type,
      label: component.defaultProps.label || component.name,
      name: component.defaultProps.name || `field_${Date.now()}`,
      required: component.defaultProps.required || false,
    } as FormField;
  }, []);

  const handleDragStart = (event: DragStartEvent) => {
    // Handle drag start if needed
  };

  const handleDragOver = (event: DragOverEvent) => {
    const { over } = event;
    setIsDragOver(!!over);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    setIsDragOver(false);

    if (!over) return;

    // Handle dropping from palette
    if (active.id.toString().startsWith('palette_')) {
      try {
        const componentData = active.data.current as PaletteComponent;
        if (componentData) {
          const newField = createFieldFromComponent(componentData);
          
          // Find insertion index based on drop target
          let insertIndex = fields.length;
          if (over.id !== 'form-canvas') {
            const targetIndex = fields.findIndex(field => field.id === over.id);
            if (targetIndex !== -1) {
              insertIndex = targetIndex + 1;
            }
          }

          const newFields = [...fields];
          newFields.splice(insertIndex, 0, newField);
          onFieldsChange(newFields);
        }
      } catch (error) {
        console.error('Error creating field from component:', error);
      }
      return;
    }

    // Handle reordering existing fields
    if (active.id !== over.id) {
      const oldIndex = fields.findIndex(field => field.id === active.id);
      const newIndex = fields.findIndex(field => field.id === over.id);

      if (oldIndex !== -1 && newIndex !== -1) {
        onFieldsChange(arrayMove(fields, oldIndex, newIndex));
      }
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(false);
    
    try {
      const componentData = JSON.parse(e.dataTransfer.getData('application/json')) as PaletteComponent;
      if (componentData) {
        const newField = createFieldFromComponent(componentData);
        onFieldsChange([...fields, newField]);
      }
    } catch (error) {
      console.error('Error handling drop:', error);
    }
  };

  const handleDragOverCanvas = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(true);
  };

  const handleDragLeave = () => {
    setIsDragOver(false);
  };

  const getCanvasStyles = () => {
    const baseStyles = "mx-auto bg-white shadow-sm border rounded-lg transition-all duration-200";
    
    switch (viewMode) {
      case 'mobile':
        return `${baseStyles} max-w-sm`;
      case 'tablet':
        return `${baseStyles} max-w-2xl`;
      default:
        return `${baseStyles} max-w-4xl`;
    }
  };

  const getCanvasHeight = () => {
    switch (viewMode) {
      case 'mobile':
        return 'min-h-[640px]';
      case 'tablet':
        return 'min-h-[768px]';
      default:
        return 'min-h-[600px]';
    }
  };

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      onDragStart={handleDragStart}
      onDragOver={handleDragOver}
      onDragEnd={handleDragEnd}
    >
      <div className="flex-1 bg-gray-50 flex flex-col">
        {/* Toolbar */}
        <div className="bg-white border-b border-gray-200 px-4 py-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <h3 className="text-lg font-medium text-gray-900">
                {isPreviewMode ? 'Form Preview' : 'Form Builder'}
              </h3>
              
              {/* View Mode Selector */}
              <div className="flex items-center space-x-1 bg-gray-100 rounded-lg p-1">
                <button
                  onClick={() => setViewMode('desktop')}
                  className={`
                    flex items-center space-x-1 px-3 py-1.5 text-xs font-medium rounded-md transition-colors
                    ${viewMode === 'desktop' ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-600 hover:text-gray-900'}
                  `}
                >
                  <Monitor size={14} />
                  <span>Desktop</span>
                </button>
                <button
                  onClick={() => setViewMode('tablet')}
                  className={`
                    flex items-center space-x-1 px-3 py-1.5 text-xs font-medium rounded-md transition-colors
                    ${viewMode === 'tablet' ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-600 hover:text-gray-900'}
                  `}
                >
                  <Tablet size={14} />
                  <span>Tablet</span>
                </button>
                <button
                  onClick={() => setViewMode('mobile')}
                  className={`
                    flex items-center space-x-1 px-3 py-1.5 text-xs font-medium rounded-md transition-colors
                    ${viewMode === 'mobile' ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-600 hover:text-gray-900'}
                  `}
                >
                  <Smartphone size={14} />
                  <span>Mobile</span>
                </button>
              </div>
            </div>

            <div className="flex items-center space-x-2">
              {!isPreviewMode && (
                <button
                  onClick={() => setShowGrid(!showGrid)}
                  className={`
                    p-2 rounded-lg transition-colors
                    ${showGrid 
                      ? 'bg-blue-100 text-blue-600' 
                      : 'text-gray-500 hover:text-gray-700 hover:bg-gray-100'
                    }
                  `}
                  title="Toggle grid"
                >
                  <Grid size={16} />
                </button>
              )}
              
              <button
                onClick={onPreviewToggle}
                className={`
                  flex items-center space-x-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors
                  ${isPreviewMode 
                    ? 'bg-gray-100 text-gray-700 hover:bg-gray-200' 
                    : 'bg-blue-600 text-white hover:bg-blue-700'
                  }
                `}
              >
                {isPreviewMode ? (
                  <>
                    <Settings size={16} />
                    <span>Edit Form</span>
                  </>
                ) : (
                  <>
                    <Eye size={16} />
                    <span>Preview</span>
                  </>
                )}
              </button>
            </div>
          </div>
        </div>

        {/* Canvas Area */}
        <div className="flex-1 p-6 overflow-auto">
          <div
            className={getCanvasStyles()}
            onDrop={handleDrop}
            onDragOver={handleDragOverCanvas}
            onDragLeave={handleDragLeave}
          >
            <div 
              className={`
                ${getCanvasHeight()} p-6 relative
                ${showGrid && !isPreviewMode ? 'bg-grid-pattern' : ''}
                ${isDragOver && !isPreviewMode ? 'bg-blue-50 border-blue-300 border-2 border-dashed' : ''}
              `}
              id="form-canvas"
            >
              {fields.length === 0 ? (
                <div className="flex flex-col items-center justify-center h-full text-center">
                  <div className="mb-4">
                    <Plus size={48} className="mx-auto text-gray-300 mb-4" />
                    <h3 className="text-lg font-medium text-gray-900 mb-2">
                      {isPreviewMode ? 'No Form Fields' : 'Build Your Form'}
                    </h3>
                    <p className="text-gray-500 max-w-sm">
                      {isPreviewMode 
                        ? 'This form doesn\'t have any fields yet. Switch to edit mode to add fields.'
                        : 'Drag components from the palette to start building your form.'
                      }
                    </p>
                  </div>
                </div>
              ) : (
                <SortableContext items={fields.map(field => field.id)} strategy={verticalListSortingStrategy}>
                  <AnimatePresence mode="popLayout">
                    {fields.map(field => (
                      <SortableFieldWrapper
                        key={field.id}
                        field={field}
                        isPreviewMode={isPreviewMode}
                        disabled={isPreviewMode}
                      >
                        <FieldRenderer
                          field={field}
                          isSelected={selectedFieldId === field.id}
                          isPreview={isPreviewMode}
                          onSelect={onFieldSelect}
                          onDelete={onFieldDelete}
                          onDuplicate={onFieldDuplicate}
                          onEdit={onFieldEdit}
                          onChange={onFormValueChange}
                          value={formValues[field.id]}
                        />
                      </SortableFieldWrapper>
                    ))}
                  </AnimatePresence>
                </SortableContext>
              )}

              {/* Drop zone indicator */}
              {isDragOver && !isPreviewMode && (
                <motion.div
                  initial={{ opacity: 0, scale: 0.9 }}
                  animate={{ opacity: 1, scale: 1 }}
                  exit={{ opacity: 0, scale: 0.9 }}
                  className="absolute inset-4 border-2 border-blue-400 border-dashed rounded-lg bg-blue-50 flex items-center justify-center pointer-events-none"
                >
                  <div className="text-center">
                    <Plus size={24} className="mx-auto text-blue-500 mb-2" />
                    <p className="text-blue-600 font-medium">Drop component here</p>
                  </div>
                </motion.div>
              )}
            </div>

            {/* Form Submit Button (Preview Mode) */}
            {isPreviewMode && fields.length > 0 && (
              <div className="px-6 pb-6">
                <button
                  type="submit"
                  className="w-full bg-blue-600 text-white py-3 px-4 rounded-lg font-medium hover:bg-blue-700 transition-colors"
                >
                  Submit Form
                </button>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Custom CSS for grid pattern */}
      <style jsx>{`
        .bg-grid-pattern {
          background-image: 
            linear-gradient(rgba(0, 0, 0, 0.1) 1px, transparent 1px),
            linear-gradient(90deg, rgba(0, 0, 0, 0.1) 1px, transparent 1px);
          background-size: 20px 20px;
        }
      `}</style>
    </DndContext>
  );
};

export default FormCanvas;