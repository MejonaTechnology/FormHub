'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'

interface Form {
  id: string
  name: string
  description: string
  target_email: string
  cc_emails: string[]
  subject: string
  success_message: string
  redirect_url: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export default function FormsPage() {
  const [forms, setForms] = useState<Form[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [newForm, setNewForm] = useState({
    name: '',
    description: '',
    target_email: '',
    cc_emails: '',
    subject: 'New Form Submission',
    success_message: 'Thank you for your submission!',
    redirect_url: ''
  })
  const router = useRouter()

  useEffect(() => {
    fetchForms()
  }, [])

  const fetchForms = async () => {
    try {
      const token = localStorage.getItem('formhub_token')
      if (!token) {
        router.push('/auth/login')
        return
      }

      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'https://formhub.mejona.in/api/v1'
      const response = await fetch(`${apiUrl}/forms`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      })

      if (response.ok) {
        const data = await response.json()
        console.log('Forms API Response:', data)
        setForms(data.forms || [])
      } else if (response.status === 401) {
        localStorage.removeItem('formhub_token')
        router.push('/auth/login')
      }
    } catch (error) {
      console.error('Error fetching forms:', error)
    } finally {
      setLoading(false)
    }
  }

  const createForm = async (e: React.FormEvent) => {
    e.preventDefault()
    
    try {
      const token = localStorage.getItem('formhub_token')
      if (!token) {
        router.push('/auth/login')
        return
      }

      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'https://formhub.mejona.in/api/v1'
      const formData = {
        ...newForm,
        cc_emails: newForm.cc_emails ? newForm.cc_emails.split(',').map(email => email.trim()) : []
      }

      const response = await fetch(`${apiUrl}/forms`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(formData)
      })

      if (response.ok) {
        const data = await response.json()
        setForms([...forms, data.form])
        setShowCreateForm(false)
        setNewForm({
          name: '',
          description: '',
          target_email: '',
          cc_emails: '',
          subject: 'New Form Submission',
          success_message: 'Thank you for your submission!',
          redirect_url: ''
        })
      } else {
        const error = await response.json()
        alert(`Error creating form: ${error.error || 'Unknown error'}`)
      }
    } catch (error) {
      console.error('Error creating form:', error)
      alert('Error creating form. Please try again.')
    }
  }

  const toggleFormStatus = async (formId: string, isActive: boolean) => {
    try {
      const token = localStorage.getItem('formhub_token')
      if (!token) return

      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'https://formhub.mejona.in/api/v1'
      const response = await fetch(`${apiUrl}/forms/${formId}`, {
        method: 'PUT',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ is_active: !isActive })
      })

      if (response.ok) {
        fetchForms()
      }
    } catch (error) {
      console.error('Error updating form status:', error)
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading forms...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="mb-8">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">Forms</h1>
              <p className="mt-2 text-gray-600">
                Manage your forms and configure email recipients for submissions.
              </p>
            </div>
            <button
              onClick={() => setShowCreateForm(true)}
              className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              Create New Form
            </button>
          </div>
        </div>

        {/* Create Form Modal */}
        {showCreateForm && (
          <div className="fixed inset-0 bg-gray-600 bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg p-6 w-full max-w-2xl">
              <h2 className="text-lg font-medium text-gray-900 mb-4">Create New Form</h2>
              
              <form onSubmit={createForm} className="space-y-4">
                <div>
                  <label htmlFor="name" className="block text-sm font-medium text-gray-700">
                    Form Name *
                  </label>
                  <input
                    type="text"
                    id="name"
                    required
                    value={newForm.name}
                    onChange={(e) => setNewForm({...newForm, name: e.target.value})}
                    className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm text-gray-900 bg-white placeholder-gray-400 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                    placeholder="e.g., Contact Form"
                  />
                </div>

                <div>
                  <label htmlFor="description" className="block text-sm font-medium text-gray-700">
                    Description
                  </label>
                  <textarea
                    id="description"
                    value={newForm.description}
                    onChange={(e) => setNewForm({...newForm, description: e.target.value})}
                    rows={3}
                    className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm text-gray-900 bg-white placeholder-gray-400 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                    placeholder="Description of your form"
                  />
                </div>

                <div>
                  <label htmlFor="target_email" className="block text-sm font-medium text-gray-700">
                    Target Email Address *
                  </label>
                  <input
                    type="email"
                    id="target_email"
                    required
                    value={newForm.target_email}
                    onChange={(e) => setNewForm({...newForm, target_email: e.target.value})}
                    className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm text-gray-900 bg-white placeholder-gray-400 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                    placeholder="your-email@example.com"
                  />
                  <p className="mt-1 text-sm text-gray-500">
                    This is where form submissions will be sent
                  </p>
                </div>

                <div>
                  <label htmlFor="cc_emails" className="block text-sm font-medium text-gray-700">
                    CC Email Addresses
                  </label>
                  <input
                    type="text"
                    id="cc_emails"
                    value={newForm.cc_emails}
                    onChange={(e) => setNewForm({...newForm, cc_emails: e.target.value})}
                    className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm text-gray-900 bg-white placeholder-gray-400 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                    placeholder="email1@example.com, email2@example.com"
                  />
                  <p className="mt-1 text-sm text-gray-500">
                    Separate multiple emails with commas
                  </p>
                </div>

                <div>
                  <label htmlFor="subject" className="block text-sm font-medium text-gray-700">
                    Email Subject
                  </label>
                  <input
                    type="text"
                    id="subject"
                    value={newForm.subject}
                    onChange={(e) => setNewForm({...newForm, subject: e.target.value})}
                    className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm text-gray-900 bg-white placeholder-gray-400 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>

                <div>
                  <label htmlFor="success_message" className="block text-sm font-medium text-gray-700">
                    Success Message
                  </label>
                  <input
                    type="text"
                    id="success_message"
                    value={newForm.success_message}
                    onChange={(e) => setNewForm({...newForm, success_message: e.target.value})}
                    className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm text-gray-900 bg-white placeholder-gray-400 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>

                <div className="flex justify-end space-x-3 pt-4">
                  <button
                    type="button"
                    onClick={() => setShowCreateForm(false)}
                    className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700"
                  >
                    Create Form
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Forms List */}
        {forms.length === 0 ? (
          <div className="text-center py-12">
            <div className="mx-auto h-12 w-12 text-gray-400">
              <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <h3 className="mt-2 text-sm font-medium text-gray-900">No forms</h3>
            <p className="mt-1 text-sm text-gray-500">Get started by creating your first form.</p>
            <div className="mt-6">
              <button
                onClick={() => setShowCreateForm(true)}
                className="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700"
              >
                Create your first form
              </button>
            </div>
          </div>
        ) : (
          <div className="bg-white shadow overflow-hidden sm:rounded-md">
            <ul className="divide-y divide-gray-200">
              {forms.map((form) => (
                <li key={form.id}>
                  <div className="px-4 py-4 flex items-center justify-between">
                    <div className="flex-1">
                      <div className="flex items-center justify-between">
                        <div>
                          <p className="text-sm font-medium text-blue-600 truncate">
                            {form.name}
                          </p>
                          <p className="text-sm text-gray-500">
                            Target: {form.target_email}
                          </p>
                          {form.description && (
                            <p className="text-sm text-gray-400 mt-1">
                              {form.description}
                            </p>
                          )}
                        </div>
                        <div className="flex items-center space-x-3">
                          <button
                            onClick={() => toggleFormStatus(form.id, form.is_active)}
                            className={`px-3 py-1 text-xs rounded-full ${
                              form.is_active
                                ? 'bg-green-100 text-green-800 hover:bg-green-200'
                                : 'bg-red-100 text-red-800 hover:bg-red-200'
                            }`}
                          >
                            {form.is_active ? 'Active' : 'Inactive'}
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          </div>
        )}

        {/* Instructions */}
        <div className="mt-8 bg-blue-50 border border-blue-200 rounded-md p-4">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-blue-400" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-blue-800">
                How to use your forms
              </h3>
              <div className="mt-2 text-sm text-blue-700">
                <p>
                  1. Create a form with your email address as the target<br/>
                  2. Use your API key from the API Keys page<br/>
                  3. Submit data to <code className="bg-blue-100 px-1 rounded">https://formhub.mejona.in/api/v1/submit</code><br/>
                  4. Include the <code className="bg-blue-100 px-1 rounded">access_key</code> field with your API key
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}