'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'

interface ApiKey {
  id: string
  name: string
  key?: string // Only shown when first created
  permissions: string
  rateLimit: number
  isActive: boolean
  createdAt: string
  lastUsedAt?: string
}

export default function ApiKeysPage() {
  const [apiKeys, setApiKeys] = useState<ApiKey[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [newKeyName, setNewKeyName] = useState('')
  const [creating, setCreating] = useState(false)
  const [newlyCreatedKey, setNewlyCreatedKey] = useState<string | null>(null)
  const router = useRouter()

  useEffect(() => {
    const token = localStorage.getItem('formhub_token')
    if (!token) {
      router.push('/auth/login')
      return
    }

    fetchApiKeys(token)
  }, [router])

  const fetchApiKeys = async (token: string) => {
    try {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://13.127.59.135:9000/api/v1';
      console.log('API URL:', apiUrl);
      const response = await fetch(`${apiUrl}/api-keys`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      })

      if (response.ok) {
        const data = await response.json()
        // Ensure we have a valid array and filter out any invalid items  
        // API returns 'api_keys' but we need to handle both formats
        const apiKeysData = data.api_keys || data.apiKeys || []
        const keys = Array.isArray(apiKeysData) ? apiKeysData.filter(key => 
          key && 
          typeof key === 'object' && 
          key.id && 
          key.name && 
          typeof key.name === 'string'
        ).map(key => ({
          ...key,
          // Handle both is_active and isActive formats
          isActive: key.isActive !== undefined ? key.isActive : key.is_active
        })) : []
        setApiKeys(keys)
      } else if (response.status === 401) {
        router.push('/auth/login')
      }
    } catch (error) {
      console.error('Error fetching API keys:', error)
    } finally {
      setLoading(false)
    }
  }

  const createApiKey = async () => {
    if (!newKeyName.trim()) return

    setCreating(true)
    const token = localStorage.getItem('formhub_token')

    try {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://13.127.59.135:9000/api/v1';
      const response = await fetch(`${apiUrl}/api-keys`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: newKeyName,
          permissions: 'form_submit',
          rateLimit: 1000
        }),
      })

      if (response.ok) {
        const data = await response.json()
        // Handle the actual API response format
        const apiKey = data.api_key || data.apiKey
        if (apiKey && apiKey.key) {
          setNewlyCreatedKey(apiKey.key)
        }
        if (apiKey && apiKey.id && apiKey.name) {
          setApiKeys([...apiKeys, apiKey])
        }
        setShowCreateModal(false)
        setNewKeyName('')
      }
    } catch (error) {
      console.error('Error creating API key:', error)
    } finally {
      setCreating(false)
    }
  }

  const deleteApiKey = async (keyId: string) => {
    if (!confirm('Are you sure you want to delete this API key? This action cannot be undone.')) {
      return
    }

    const token = localStorage.getItem('formhub_token')

    try {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://13.127.59.135:9000/api/v1';
      const response = await fetch(`${apiUrl}/api-keys/${keyId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      })

      if (response.ok) {
        setApiKeys(apiKeys.filter(key => key.id !== keyId))
      }
    } catch (error) {
      console.error('Error deleting API key:', error)
    }
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    alert('API key copied to clipboard!')
  }

  const toggleApiKeyStatus = async (keyId: string, currentStatus: boolean) => {
    try {
      const token = localStorage.getItem('access_token')
      if (!token) {
        router.push('/auth/login')
        return
      }

      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://13.127.59.135:9000/api/v1'
      const response = await fetch(`${apiUrl}/api-keys/${keyId}/toggle`, {
        method: 'PUT',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ is_active: !currentStatus })
      })

      if (response.ok) {
        const token = localStorage.getItem('access_token')
        if (token) {
          fetchApiKeys(token) // Refresh the list
        }
      } else {
        const error = await response.json()
        alert(`Error updating API key status: ${error.error || 'Unknown error'}`)
      }
    } catch (error) {
      console.error('Error toggling API key status:', error)
      alert('Error updating API key status. Please try again.')
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-6">
            <div className="flex items-center">
              <Link href="/" className="text-2xl font-bold text-blue-600">
                FormHub
              </Link>
              <nav className="ml-10 flex space-x-8">
                <Link href="/dashboard" className="text-gray-500 hover:text-blue-600">
                  Dashboard
                </Link>
                <Link href="/dashboard/forms" className="text-gray-500 hover:text-blue-600">
                  Forms
                </Link>
                <Link href="/dashboard/api-keys" className="text-gray-900 hover:text-blue-600">
                  API Keys
                </Link>
                <Link href="/docs" className="text-gray-500 hover:text-blue-600">
                  Docs
                </Link>
              </nav>
            </div>
            <Link href="/dashboard" className="text-gray-500 hover:text-gray-700">
              Back to Dashboard
            </Link>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0">
          <div className="flex justify-between items-center mb-8">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">API Keys</h1>
              <p className="mt-2 text-gray-600">
                Manage your API keys to access FormHub programmatically.
              </p>
            </div>
            <button
              onClick={() => setShowCreateModal(true)}
              className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700"
            >
              Create New Key
            </button>
          </div>

          {/* New API Key Alert */}
          {newlyCreatedKey && (
            <div className="bg-green-50 border border-green-200 rounded-md p-4 mb-8">
              <div className="flex">
                <div className="ml-3">
                  <h3 className="text-sm font-medium text-green-800">
                    API Key Created Successfully!
                  </h3>
                  <div className="mt-2 text-sm text-green-700">
                    <p className="mb-2">Your new API key is:</p>
                    <div className="bg-white border rounded-md p-3 font-mono text-sm flex items-center justify-between">
                      <code className="text-gray-900">{newlyCreatedKey}</code>
                      <button
                        onClick={() => copyToClipboard(newlyCreatedKey)}
                        className="ml-2 text-green-600 hover:text-green-500"
                      >
                        Copy
                      </button>
                    </div>
                    <p className="mt-2 text-xs">
                      <strong>Important:</strong> This is the only time you'll see this key. Store it securely!
                    </p>
                  </div>
                  <div className="mt-4">
                    <button
                      onClick={() => setNewlyCreatedKey(null)}
                      className="text-green-800 text-sm underline"
                    >
                      Dismiss
                    </button>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* API Keys List */}
          <div className="bg-white shadow overflow-hidden sm:rounded-md">
            {apiKeys.length === 0 ? (
              <div className="text-center py-12">
                <div className="w-12 h-12 mx-auto bg-gray-400 rounded-full flex items-center justify-center mb-4">
                  <span className="text-white font-bold">K</span>
                </div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">No API keys</h3>
                <p className="text-gray-500 mb-6">Get started by creating your first API key.</p>
                <button
                  onClick={() => setShowCreateModal(true)}
                  className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700"
                >
                  Create API Key
                </button>
              </div>
            ) : (
              <ul className="divide-y divide-gray-200">
                {apiKeys.filter(apiKey => 
                  apiKey && 
                  typeof apiKey === 'object' && 
                  apiKey.id && 
                  apiKey.name && 
                  typeof apiKey.name === 'string'
                ).map((apiKey) => (
                  <li key={apiKey.id}>
                    <div className="px-6 py-4">
                      <div className="flex items-center justify-between">
                        <div className="flex-1">
                          <div className="flex items-center">
                            <h3 className="text-lg font-medium text-gray-900">
                              {apiKey.name}
                            </h3>
                            <button
                              onClick={() => toggleApiKeyStatus(apiKey.id, apiKey.isActive)}
                              className={`ml-3 px-2 py-1 text-xs rounded-full cursor-pointer transition-colors ${
                                apiKey.isActive 
                                  ? 'bg-green-100 text-green-800 hover:bg-green-200' 
                                  : 'bg-red-100 text-red-800 hover:bg-red-200'
                              }`}
                              title="Click to toggle status"
                            >
                              {apiKey.isActive ? 'Active' : 'Inactive'}
                            </button>
                          </div>
                          <div className="mt-2 text-sm text-gray-600">
                            <p>Permissions: {apiKey.permissions || 'form_submit'}</p>
                            <p>Rate limit: {apiKey.rateLimit || 1000} requests/minute</p>
                            <p>Created: {apiKey.createdAt ? new Date(apiKey.createdAt).toLocaleDateString() : 'Unknown'}</p>
                            {apiKey.lastUsedAt && (
                              <p>Last used: {new Date(apiKey.lastUsedAt).toLocaleDateString()}</p>
                            )}
                          </div>
                        </div>
                        <div className="flex items-center space-x-2">
                          <button
                            onClick={() => deleteApiKey(apiKey.id)}
                            className="text-red-600 hover:text-red-500 text-sm"
                          >
                            Delete
                          </button>
                        </div>
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>

          {/* Usage Instructions */}
          <div className="mt-8 bg-blue-50 border border-blue-200 rounded-md p-6">
            <h3 className="text-lg font-medium text-blue-900 mb-4">How to use your API key</h3>
            <div className="text-sm text-blue-800 space-y-4">
              <div>
                <p className="font-medium mb-2">HTML Form Example:</p>
                <pre className="bg-blue-900 text-green-400 p-4 rounded-md text-xs overflow-x-auto">
{`<form action="${process.env.NEXT_PUBLIC_FORMHUB_URL}/api/v1/submit" method="POST">
  <input type="hidden" name="access_key" value="YOUR_API_KEY">
  <input type="text" name="name" placeholder="Name" required>
  <input type="email" name="email" placeholder="Email" required>
  <textarea name="message" placeholder="Message" required></textarea>
  <button type="submit">Submit</button>
</form>`}
                </pre>
              </div>
              <div>
                <p className="font-medium mb-2">JavaScript (Fetch API) Example:</p>
                <pre className="bg-blue-900 text-green-400 p-4 rounded-md text-xs overflow-x-auto">
{`fetch('${process.env.NEXT_PUBLIC_FORMHUB_URL}/api/v1/submit', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    access_key: 'YOUR_API_KEY',
    name: 'John Doe',
    email: 'john@example.com',
    message: 'Hello FormHub!'
  })
});`}
                </pre>
              </div>
            </div>
          </div>
        </div>
      </main>

      {/* Create API Key Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-gray-600 bg-opacity-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h3 className="text-lg font-medium text-gray-900 mb-4">Create New API Key</h3>
            
            <div className="mb-4">
              <label htmlFor="keyName" className="block text-sm font-medium text-gray-700 mb-2">
                Key Name
              </label>
              <input
                type="text"
                id="keyName"
                value={newKeyName}
                onChange={(e) => setNewKeyName(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md text-gray-900 bg-white placeholder-gray-500 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                placeholder="e.g., Website Contact Form"
              />
            </div>

            <div className="flex justify-end space-x-3">
              <button
                onClick={() => setShowCreateModal(false)}
                className="px-4 py-2 text-gray-700 border border-gray-300 rounded-md hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={createApiKey}
                disabled={creating || !newKeyName.trim()}
                className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {creating ? 'Creating...' : 'Create Key'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}