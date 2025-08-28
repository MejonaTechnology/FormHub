'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'

interface User {
  id: string
  email: string
  first_name: string
  last_name: string
  company?: string
  plan_type: string
}

interface ApiKey {
  id: string
  name: string
  permissions: string
  rateLimit: number
  isActive: boolean
  createdAt: string
  lastUsedAt?: string
}

interface Form {
  id: string
  name: string
  description?: string
  targetEmail: string
  submissionCount: number
  isActive: boolean
  createdAt: string
}

export default function DashboardPage() {
  const [user, setUser] = useState<User | null>(null)
  const [apiKeys, setApiKeys] = useState<ApiKey[]>([])
  const [forms, setForms] = useState<Form[]>([])
  const [loading, setLoading] = useState(true)
  const router = useRouter()

  useEffect(() => {
    const token = localStorage.getItem('formhub_token')
    if (!token) {
      router.push('/auth/login')
      return
    }

    const userData = localStorage.getItem('formhub_user')
    if (userData) {
      setUser(JSON.parse(userData))
    }

    fetchDashboardData(token)
  }, [router])

  const fetchDashboardData = async (token: string) => {
    try {
      const headers = {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      }

      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'https://formhub.mejona.in/api/v1';
      
      // Fetch API keys
      const apiKeysResponse = await fetch(`${apiUrl}/api-keys`, {
        headers,
      })
      if (apiKeysResponse.ok) {
        const data = await apiKeysResponse.json()
        console.log('Dashboard API Keys Response:', data)
        // Use same robust handling as API keys page
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
        console.log('Dashboard Processed Keys:', keys)
        setApiKeys(keys)
      } else {
        console.error('Dashboard API Keys fetch failed:', apiKeysResponse.status, apiKeysResponse.statusText)
      }

      // Fetch forms
      const formsResponse = await fetch(`${apiUrl}/forms`, {
        headers,
      })
      if (formsResponse.ok) {
        const formsData = await formsResponse.json()
        setForms(formsData.forms || [])
      }
    } catch (error) {
      console.error('Error fetching dashboard data:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleLogout = () => {
    localStorage.removeItem('formhub_token')
    localStorage.removeItem('formhub_user')
    router.push('/')
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
                <Link href="/dashboard" className="text-gray-900 hover:text-blue-600">
                  Dashboard
                </Link>
                <Link href="/dashboard/forms" className="text-gray-500 hover:text-blue-600">
                  Forms
                </Link>
                <Link href="/dashboard/api-keys" className="text-gray-500 hover:text-blue-600">
                  API Keys
                </Link>
                <Link href="/docs" className="text-gray-500 hover:text-blue-600">
                  Docs
                </Link>
                <Link href="/pricing" className="text-gray-500 hover:text-blue-600">
                  Pricing
                </Link>
              </nav>
            </div>
            <div className="flex items-center space-x-4">
              <span className="text-sm text-gray-700">
                Welcome, {user?.first_name} {user?.last_name}
              </span>
              <button
                onClick={handleLogout}
                className="text-gray-500 hover:text-gray-700 text-sm"
              >
                Sign out
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0">
          {/* Welcome Section */}
          <div className="bg-white overflow-hidden shadow rounded-lg mb-8">
            <div className="px-4 py-5 sm:p-6">
              <h1 className="text-3xl font-bold text-gray-900 mb-4">
                Welcome to FormHub Dashboard
              </h1>
              <p className="text-gray-600 mb-6">
                Manage your forms, view submissions, and configure API keys from here.
              </p>
              
              {/* Plan Badge */}
              <div className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-blue-100 text-blue-800">
                Current Plan: {user?.plan_type?.toUpperCase() || 'FREE'}
              </div>
            </div>
          </div>

          {/* Stats Cards */}
          <div className="grid grid-cols-1 gap-5 sm:grid-cols-3 mb-8">
            <div className="bg-white overflow-hidden shadow rounded-lg">
              <div className="p-5">
                <div className="flex items-center">
                  <div className="flex-shrink-0">
                    <div className="w-8 h-8 bg-blue-500 rounded-md flex items-center justify-center">
                      <span className="text-white font-bold">F</span>
                    </div>
                  </div>
                  <div className="ml-5 w-0 flex-1">
                    <dl>
                      <dt className="text-sm font-medium text-gray-500 truncate">Total Forms</dt>
                      <dd className="text-lg font-medium text-gray-900">{forms.length}</dd>
                    </dl>
                  </div>
                </div>
              </div>
            </div>

            <div className="bg-white overflow-hidden shadow rounded-lg">
              <div className="p-5">
                <div className="flex items-center">
                  <div className="flex-shrink-0">
                    <div className="w-8 h-8 bg-green-500 rounded-md flex items-center justify-center">
                      <span className="text-white font-bold">S</span>
                    </div>
                  </div>
                  <div className="ml-5 w-0 flex-1">
                    <dl>
                      <dt className="text-sm font-medium text-gray-500 truncate">Total Submissions</dt>
                      <dd className="text-lg font-medium text-gray-900">
                        {forms.reduce((total, form) => total + form.submissionCount, 0)}
                      </dd>
                    </dl>
                  </div>
                </div>
              </div>
            </div>

            <div className="bg-white overflow-hidden shadow rounded-lg">
              <div className="p-5">
                <div className="flex items-center">
                  <div className="flex-shrink-0">
                    <div className="w-8 h-8 bg-purple-500 rounded-md flex items-center justify-center">
                      <span className="text-white font-bold">K</span>
                    </div>
                  </div>
                  <div className="ml-5 w-0 flex-1">
                    <dl>
                      <dt className="text-sm font-medium text-gray-500 truncate">API Keys</dt>
                      <dd className="text-lg font-medium text-gray-900">{apiKeys.length}</dd>
                    </dl>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Quick Actions */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            {/* Recent Forms */}
            <div className="bg-white shadow rounded-lg">
              <div className="px-4 py-5 sm:p-6">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-medium text-gray-900">Recent Forms</h3>
                  <Link
                    href="/dashboard/forms"
                    className="text-blue-600 hover:text-blue-500 text-sm"
                  >
                    View all
                  </Link>
                </div>
                
                {forms.length === 0 ? (
                  <div className="text-center py-8">
                    <p className="text-gray-500 mb-4">No forms created yet</p>
                    <Link
                      href="/dashboard/forms"
                      className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700"
                    >
                      Create your first form
                    </Link>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {forms.slice(0, 3).map((form) => (
                      <div key={form.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-md">
                        <div>
                          <p className="font-medium text-gray-900">{form.name}</p>
                          <p className="text-sm text-gray-500">{form.submissionCount} submissions</p>
                        </div>
                        <span className={`px-2 py-1 text-xs rounded-full ${
                          form.isActive ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                        }`}>
                          {form.isActive ? 'Active' : 'Inactive'}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

            {/* API Keys */}
            <div className="bg-white shadow rounded-lg">
              <div className="px-4 py-5 sm:p-6">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-medium text-gray-900">API Keys</h3>
                  <Link
                    href="/dashboard/api-keys"
                    className="text-blue-600 hover:text-blue-500 text-sm"
                  >
                    Manage keys
                  </Link>
                </div>
                
                {apiKeys.length === 0 ? (
                  <div className="text-center py-8">
                    <p className="text-gray-500 mb-4">No API keys created yet</p>
                    <Link
                      href="/dashboard/api-keys"
                      className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700"
                    >
                      Create API key
                    </Link>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {apiKeys.slice(0, 3).map((key) => (
                      <div key={key.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-md">
                        <div>
                          <p className="font-medium text-gray-900">{key.name}</p>
                          <p className="text-sm text-gray-500">Rate limit: {key.rateLimit}/min</p>
                        </div>
                        <span className={`px-2 py-1 text-xs rounded-full ${
                          key.isActive ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                        }`}>
                          {key.isActive ? 'Active' : 'Inactive'}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}