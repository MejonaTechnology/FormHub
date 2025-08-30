'use client';

import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  Type, 
  Mail, 
  Lock, 
  Hash, 
  Phone, 
  Link, 
  Calendar, 
  Clock,
  ChevronDown,
  Circle,
  CheckSquare,
  Upload,
  Image,
  Star,
  Edit3,
  Minus,
  Heading,
  FileText,
  Code,
  Palette,
  Search
} from 'lucide-react';
import { PaletteComponent, ComponentCategory, FieldType } from '@/types/form-builder';

interface ComponentPaletteProps {
  onDragStart?: (component: PaletteComponent) => void;
  onDragEnd?: () => void;
}

const ComponentPalette: React.FC<ComponentPaletteProps> = ({
  onDragStart,
  onDragEnd
}) => {
  const [activeCategory, setActiveCategory] = useState<ComponentCategory>('input');
  const [searchQuery, setSearchQuery] = useState('');
  const [expandedCategories, setExpandedCategories] = useState<ComponentCategory[]>(['input']);

  const paletteComponents: PaletteComponent[] = [
    // Input Fields
    {
      id: 'text',
      type: 'text',
      name: 'Text Input',
      description: 'Single line text input field',
      icon: <Type size={16} />,
      category: 'input',
      preview: <input type="text" className="w-full p-2 border rounded" placeholder="Enter text..." disabled />,
      defaultProps: {
        id: '',
        type: 'text',
        label: 'Text Input',
        name: 'text_field',
        placeholder: 'Enter text...',
        required: false
      }
    },
    {
      id: 'email',
      type: 'email',
      name: 'Email Input',
      description: 'Email address input with validation',
      icon: <Mail size={16} />,
      category: 'input',
      preview: <input type="email" className="w-full p-2 border rounded" placeholder="email@example.com" disabled />,
      defaultProps: {
        id: '',
        type: 'email',
        label: 'Email Address',
        name: 'email',
        placeholder: 'email@example.com',
        required: true
      }
    },
    {
      id: 'password',
      type: 'password',
      name: 'Password Input',
      description: 'Secure password input field',
      icon: <Lock size={16} />,
      category: 'input',
      preview: <input type="password" className="w-full p-2 border rounded" placeholder="••••••••" disabled />,
      defaultProps: {
        id: '',
        type: 'password',
        label: 'Password',
        name: 'password',
        placeholder: 'Enter password',
        required: true
      }
    },
    {
      id: 'textarea',
      type: 'textarea',
      name: 'Textarea',
      description: 'Multi-line text input field',
      icon: <FileText size={16} />,
      category: 'input',
      preview: <textarea className="w-full p-2 border rounded" rows={3} placeholder="Enter multiple lines..." disabled />,
      defaultProps: {
        id: '',
        type: 'textarea',
        label: 'Message',
        name: 'message',
        placeholder: 'Enter your message...',
        required: false
      }
    },
    {
      id: 'number',
      type: 'number',
      name: 'Number Input',
      description: 'Numeric input with validation',
      icon: <Hash size={16} />,
      category: 'input',
      preview: <input type="number" className="w-full p-2 border rounded" placeholder="0" disabled />,
      defaultProps: {
        id: '',
        type: 'number',
        label: 'Number',
        name: 'number',
        placeholder: 'Enter number',
        required: false
      }
    },
    {
      id: 'phone',
      type: 'phone',
      name: 'Phone Input',
      description: 'Phone number input field',
      icon: <Phone size={16} />,
      category: 'input',
      preview: <input type="tel" className="w-full p-2 border rounded" placeholder="+1 (555) 123-4567" disabled />,
      defaultProps: {
        id: '',
        type: 'phone',
        label: 'Phone Number',
        name: 'phone',
        placeholder: '+1 (555) 123-4567',
        required: false
      }
    },
    {
      id: 'url',
      type: 'url',
      name: 'URL Input',
      description: 'Website URL input field',
      icon: <Link size={16} />,
      category: 'input',
      preview: <input type="url" className="w-full p-2 border rounded" placeholder="https://example.com" disabled />,
      defaultProps: {
        id: '',
        type: 'url',
        label: 'Website URL',
        name: 'url',
        placeholder: 'https://example.com',
        required: false
      }
    },
    {
      id: 'date',
      type: 'date',
      name: 'Date Picker',
      description: 'Date selection input',
      icon: <Calendar size={16} />,
      category: 'input',
      preview: <input type="date" className="w-full p-2 border rounded" disabled />,
      defaultProps: {
        id: '',
        type: 'date',
        label: 'Date',
        name: 'date',
        required: false
      }
    },
    {
      id: 'time',
      type: 'time',
      name: 'Time Picker',
      description: 'Time selection input',
      icon: <Clock size={16} />,
      category: 'input',
      preview: <input type="time" className="w-full p-2 border rounded" disabled />,
      defaultProps: {
        id: '',
        type: 'time',
        label: 'Time',
        name: 'time',
        required: false
      }
    },

    // Choice Fields
    {
      id: 'select',
      type: 'select',
      name: 'Dropdown',
      description: 'Single selection dropdown menu',
      icon: <ChevronDown size={16} />,
      category: 'choice',
      preview: (
        <div className="relative">
          <select className="w-full p-2 border rounded appearance-none bg-white" disabled>
            <option>Select an option...</option>
          </select>
          <ChevronDown className="absolute right-2 top-1/2 transform -translate-y-1/2 text-gray-400" size={16} />
        </div>
      ),
      defaultProps: {
        id: '',
        type: 'select',
        label: 'Dropdown',
        name: 'dropdown',
        placeholder: 'Select an option...',
        required: false,
        options: [
          { id: '1', label: 'Option 1', value: 'option1' },
          { id: '2', label: 'Option 2', value: 'option2' },
          { id: '3', label: 'Option 3', value: 'option3' }
        ]
      }
    },
    {
      id: 'radio',
      type: 'radio',
      name: 'Radio Buttons',
      description: 'Single selection radio buttons',
      icon: <Circle size={16} />,
      category: 'choice',
      preview: (
        <div className="space-y-2">
          <label className="flex items-center">
            <input type="radio" name="radio_preview" className="mr-2" disabled />
            <span className="text-sm">Option 1</span>
          </label>
          <label className="flex items-center">
            <input type="radio" name="radio_preview" className="mr-2" disabled />
            <span className="text-sm">Option 2</span>
          </label>
        </div>
      ),
      defaultProps: {
        id: '',
        type: 'radio',
        label: 'Radio Group',
        name: 'radio_group',
        required: false,
        options: [
          { id: '1', label: 'Option 1', value: 'option1' },
          { id: '2', label: 'Option 2', value: 'option2' }
        ]
      }
    },
    {
      id: 'checkbox',
      type: 'checkbox',
      name: 'Checkboxes',
      description: 'Multiple selection checkboxes',
      icon: <CheckSquare size={16} />,
      category: 'choice',
      preview: (
        <div className="space-y-2">
          <label className="flex items-center">
            <input type="checkbox" className="mr-2" disabled />
            <span className="text-sm">Option 1</span>
          </label>
          <label className="flex items-center">
            <input type="checkbox" className="mr-2" disabled />
            <span className="text-sm">Option 2</span>
          </label>
        </div>
      ),
      defaultProps: {
        id: '',
        type: 'checkbox',
        label: 'Checkboxes',
        name: 'checkboxes',
        required: false,
        options: [
          { id: '1', label: 'Option 1', value: 'option1' },
          { id: '2', label: 'Option 2', value: 'option2' }
        ]
      }
    },

    // Media Fields
    {
      id: 'file',
      type: 'file',
      name: 'File Upload',
      description: 'File upload field',
      icon: <Upload size={16} />,
      category: 'media',
      preview: (
        <div className="border-2 border-dashed border-gray-300 rounded-lg p-4 text-center">
          <Upload size={20} className="mx-auto mb-2 text-gray-400" />
          <span className="text-sm text-gray-500">Click to upload</span>
        </div>
      ),
      defaultProps: {
        id: '',
        type: 'file',
        label: 'File Upload',
        name: 'file_upload',
        required: false
      }
    },
    {
      id: 'image',
      type: 'image',
      name: 'Image Upload',
      description: 'Image file upload with preview',
      icon: <Image size={16} />,
      category: 'media',
      preview: (
        <div className="border-2 border-dashed border-gray-300 rounded-lg p-4 text-center">
          <Image size={20} className="mx-auto mb-2 text-gray-400" />
          <span className="text-sm text-gray-500">Upload image</span>
        </div>
      ),
      defaultProps: {
        id: '',
        type: 'image',
        label: 'Image Upload',
        name: 'image_upload',
        required: false
      }
    },

    // Advanced Fields
    {
      id: 'rating',
      type: 'rating',
      name: 'Star Rating',
      description: 'Star rating input field',
      icon: <Star size={16} />,
      category: 'advanced',
      preview: (
        <div className="flex">
          {[1, 2, 3, 4, 5].map(star => (
            <Star key={star} size={16} className="text-gray-300" />
          ))}
        </div>
      ),
      defaultProps: {
        id: '',
        type: 'rating',
        label: 'Rating',
        name: 'rating',
        required: false
      }
    },
    {
      id: 'signature',
      type: 'signature',
      name: 'Signature',
      description: 'Digital signature pad',
      icon: <Edit3 size={16} />,
      category: 'advanced',
      preview: (
        <div className="border border-gray-300 rounded p-2 bg-gray-50">
          <span className="text-xs text-gray-500">Signature pad</span>
        </div>
      ),
      defaultProps: {
        id: '',
        type: 'signature',
        label: 'Signature',
        name: 'signature',
        required: false
      }
    },

    // Layout Fields
    {
      id: 'divider',
      type: 'divider',
      name: 'Divider',
      description: 'Horizontal divider line',
      icon: <Minus size={16} />,
      category: 'layout',
      preview: <hr className="border-gray-300" />,
      defaultProps: {
        id: '',
        type: 'divider',
        label: '',
        name: 'divider',
        required: false
      }
    },
    {
      id: 'heading',
      type: 'heading',
      name: 'Heading',
      description: 'Section heading text',
      icon: <Heading size={16} />,
      category: 'layout',
      preview: <h3 className="text-lg font-semibold">Section Heading</h3>,
      defaultProps: {
        id: '',
        type: 'heading',
        label: 'Section Heading',
        name: 'heading',
        required: false
      }
    },
    {
      id: 'paragraph',
      type: 'paragraph',
      name: 'Paragraph',
      description: 'Descriptive text paragraph',
      icon: <FileText size={16} />,
      category: 'layout',
      preview: <p className="text-sm text-gray-600">Descriptive text goes here...</p>,
      defaultProps: {
        id: '',
        type: 'paragraph',
        label: 'Description',
        name: 'paragraph',
        required: false
      }
    },
    {
      id: 'html',
      type: 'html',
      name: 'Custom HTML',
      description: 'Custom HTML content block',
      icon: <Code size={16} />,
      category: 'layout',
      preview: (
        <div className="bg-gray-100 border rounded p-2 text-center">
          <Code size={16} className="mx-auto text-gray-500" />
          <span className="text-xs text-gray-500">HTML</span>
        </div>
      ),
      defaultProps: {
        id: '',
        type: 'html',
        label: 'Custom HTML',
        name: 'html_content',
        required: false
      }
    }
  ];

  const categories: { id: ComponentCategory; name: string; icon: React.ReactNode }[] = [
    { id: 'input', name: 'Input Fields', icon: <Type size={16} /> },
    { id: 'choice', name: 'Choice Fields', icon: <Circle size={16} /> },
    { id: 'media', name: 'Media', icon: <Upload size={16} /> },
    { id: 'advanced', name: 'Advanced', icon: <Star size={16} /> },
    { id: 'layout', name: 'Layout', icon: <Palette size={16} /> }
  ];

  const filteredComponents = paletteComponents.filter(component => {
    const matchesSearch = component.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         component.description.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesCategory = activeCategory === component.category;
    return matchesSearch && matchesCategory;
  });

  const toggleCategory = (categoryId: ComponentCategory) => {
    setExpandedCategories(prev => 
      prev.includes(categoryId) 
        ? prev.filter(id => id !== categoryId)
        : [...prev, categoryId]
    );
  };

  const handleDragStart = (e: React.DragEvent, component: PaletteComponent) => {
    e.dataTransfer.setData('application/json', JSON.stringify(component));
    if (onDragStart) {
      onDragStart(component);
    }
  };

  const handleDragEnd = () => {
    if (onDragEnd) {
      onDragEnd();
    }
  };

  return (
    <div className="w-80 bg-white border-r border-gray-200 flex flex-col h-full">
      {/* Header */}
      <div className="p-4 border-b border-gray-200">
        <h2 className="text-lg font-semibold text-gray-900 mb-3">Form Components</h2>
        
        {/* Search */}
        <div className="relative mb-3">
          <Search size={16} className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            placeholder="Search components..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-9 pr-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          />
        </div>

        {/* Category Tabs */}
        <div className="flex flex-wrap gap-1">
          {categories.map(category => (
            <button
              key={category.id}
              onClick={() => setActiveCategory(category.id)}
              className={`
                flex items-center space-x-1 px-3 py-1.5 text-xs font-medium rounded-lg transition-colors
                ${activeCategory === category.id 
                  ? 'bg-blue-100 text-blue-700' 
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                }
              `}
            >
              {category.icon}
              <span>{category.name}</span>
            </button>
          ))}
        </div>
      </div>

      {/* Components List */}
      <div className="flex-1 overflow-y-auto">
        <div className="p-2">
          <AnimatePresence mode="wait">
            <motion.div
              key={activeCategory}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              transition={{ duration: 0.2 }}
              className="space-y-2"
            >
              {filteredComponents.map(component => (
                <motion.div
                  key={component.id}
                  layout
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                  draggable
                  onDragStart={(e: any) => handleDragStart(e, component)}
                  onDragEnd={handleDragEnd}
                  className="
                    bg-white border border-gray-200 rounded-lg p-3 cursor-grab active:cursor-grabbing
                    hover:border-blue-300 hover:shadow-sm transition-all duration-200
                    group
                  "
                >
                  <div className="flex items-start space-x-3">
                    <div className="flex-shrink-0 w-8 h-8 bg-gray-100 rounded-lg flex items-center justify-center text-gray-600">
                      {component.icon}
                    </div>
                    
                    <div className="flex-1 min-w-0">
                      <h3 className="text-sm font-medium text-gray-900 truncate">
                        {component.name}
                      </h3>
                      <p className="text-xs text-gray-500 mt-1">
                        {component.description}
                      </p>
                      
                      {/* Preview */}
                      <div className="mt-2 opacity-75 group-hover:opacity-100 transition-opacity">
                        {component.preview}
                      </div>
                    </div>
                  </div>
                </motion.div>
              ))}
            </motion.div>
          </AnimatePresence>
          
          {filteredComponents.length === 0 && (
            <div className="text-center py-8 text-gray-500">
              <Search size={24} className="mx-auto mb-2 opacity-50" />
              <p className="text-sm">No components found</p>
              <p className="text-xs mt-1">Try adjusting your search or category filter</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default ComponentPalette;