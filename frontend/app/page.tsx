import Link from 'next/link'

export default function HomePage() {
  return (
    <div className="min-h-screen bg-white">
      {/* Navigation */}
      <header className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-6">
            <Link href="/" className="text-2xl font-bold text-blue-600">FormHub</Link>
            <nav className="flex space-x-8">
              <Link href="/docs" className="text-gray-500 hover:text-blue-600">Documentation</Link>
              <Link href="/pricing" className="text-gray-500 hover:text-blue-600">Pricing</Link>
              <Link href="/auth/login" className="text-gray-500 hover:text-blue-600">Sign In</Link>
              <Link href="/auth/register" className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700">
                Get Started
              </Link>
            </nav>
          </div>
        </div>
      </header>

      <div className="container mx-auto px-6 py-16">
        <div className="text-center">
          <h1 className="text-5xl font-bold text-gray-900 mb-8">
            Welcome to <span className="text-blue-600">FormHub</span>
          </h1>
          <p className="text-xl text-gray-700 mb-12 max-w-3xl mx-auto">
            Professional form backend service for static websites and modern applications. 
            Handle contact forms, user registrations, and more with ease.
          </p>
          
          <div className="grid md:grid-cols-2 gap-8 max-w-4xl mx-auto mt-16">
            <div className="bg-blue-50 border border-blue-200 p-8 rounded-lg shadow-lg">
              <h2 className="text-2xl font-semibold text-blue-900 mb-4">For Developers</h2>
              <p className="text-blue-800 mb-6">
                Easy-to-use API endpoints for form handling. No server-side code required.
              </p>
              <div className="bg-gray-800 text-green-400 p-4 rounded-md text-sm font-mono">
                &lt;form action="https://formhub.mejona.in/api/v1/submit" method="POST"&gt;
                <br />
                &nbsp;&nbsp;&lt;input name="name" placeholder="Name" required&gt;
                <br />
                &nbsp;&nbsp;&lt;button type="submit"&gt;Submit&lt;/button&gt;
                <br />
                &lt;/form&gt;
              </div>
            </div>
            
            <div className="bg-green-50 border border-green-200 p-8 rounded-lg shadow-lg">
              <h2 className="text-2xl font-semibold text-green-900 mb-4">For Businesses</h2>
              <p className="text-green-800 mb-6">
                Manage form submissions with our intuitive dashboard. Get notified instantly.
              </p>
              <div className="space-y-3">
                <div className="flex items-center text-green-700">
                  <div className="w-2 h-2 bg-green-600 rounded-full mr-3"></div>
                  Real-time notifications
                </div>
                <div className="flex items-center text-green-700">
                  <div className="w-2 h-2 bg-green-600 rounded-full mr-3"></div>
                  Spam protection built-in
                </div>
                <div className="flex items-center text-green-700">
                  <div className="w-2 h-2 bg-green-600 rounded-full mr-3"></div>
                  Export to CSV/Excel
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="bg-gray-900 text-white py-16">
        <div className="container mx-auto px-6">
          <div className="text-center">
            <h2 className="text-4xl font-bold mb-8">Ready to Get Started?</h2>
            <p className="text-xl text-gray-300 mb-8 max-w-2xl mx-auto">
              Join thousands of developers who trust FormHub for their form handling needs.
            </p>
            <div className="flex justify-center space-x-4">
              <Link href="/auth/register" className="bg-blue-600 hover:bg-blue-700 text-white font-semibold py-3 px-8 rounded-lg transition-colors">
                Start Free Trial
              </Link>
              <Link href="/docs" className="bg-gray-800 hover:bg-gray-700 text-white font-semibold py-3 px-8 rounded-lg transition-colors">
                View Documentation
              </Link>
            </div>
          </div>

          <div className="mt-16 grid md:grid-cols-3 gap-8">
            <div className="text-center">
              <div className="bg-blue-600 w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4">
                <span className="text-2xl font-bold">1</span>
              </div>
              <h3 className="text-xl font-semibold mb-2">Create Account</h3>
              <p className="text-gray-400">Sign up for your free FormHub account in seconds.</p>
            </div>
            <div className="text-center">
              <div className="bg-green-600 w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4">
                <span className="text-2xl font-bold">2</span>
              </div>
              <h3 className="text-xl font-semibold mb-2">Get API Key</h3>
              <p className="text-gray-400">Generate your unique API key from the dashboard.</p>
            </div>
            <div className="text-center">
              <div className="bg-purple-600 w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4">
                <span className="text-2xl font-bold">3</span>
              </div>
              <h3 className="text-xl font-semibold mb-2">Add to Forms</h3>
              <p className="text-gray-400">Point your HTML forms to our endpoint and you're done!</p>
            </div>
          </div>

          <div className="mt-16 text-center">
            <div className="bg-gray-800 rounded-lg p-8 max-w-4xl mx-auto">
              <h3 className="text-2xl font-bold mb-6">Live API Endpoint</h3>
              <p className="text-gray-300 mb-4">
                Our backend is currently running and ready to handle your form submissions.
              </p>
              
              <div className="bg-gray-900 text-green-400 px-6 py-4 rounded-md text-sm font-mono overflow-x-auto mb-4">
                curl -X POST https://formhub.mejona.in/api/v1/submit -H "Content-Type: application/json" -d {JSON.stringify({"access_key": "test-key-123", "name": "Test User", "email": "test@example.com", "message": "Hello FormHub!"})}
              </div>
              <div className="mt-6 p-4 bg-green-100 rounded-lg">
                <p className="text-green-800 text-sm">
                  <strong>HTTPS API:</strong> https://formhub.mejona.in/api/v1<br/>
                  <strong>Test API Key:</strong> test-key-123<br/>
                  <strong>SSL Status:</strong> Let&apos;s Encrypt Certificate<br/>
                  <strong>Status:</strong> Ready for secure form submissions
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <footer className="bg-white py-12">
        <div className="container mx-auto px-6">
          <div className="grid md:grid-cols-4 gap-8">
            <div>
              <h3 className="text-xl font-bold text-blue-600 mb-4">FormHub</h3>
              <p className="text-gray-600 mb-4">
                Professional form backend service for modern web applications.
              </p>
              <div className="flex space-x-4">
                <a href="#" className="text-gray-400 hover:text-blue-600">Twitter</a>
                <a href="#" className="text-gray-400 hover:text-blue-600">GitHub</a>
                <a href="#" className="text-gray-400 hover:text-blue-600">LinkedIn</a>
              </div>
            </div>
            
            <div>
              <h4 className="font-semibold text-gray-900 mb-4">Product</h4>
              <ul className="space-y-2">
                <li><Link href="/pricing" className="text-gray-600 hover:text-blue-600">Pricing</Link></li>
                <li><Link href="/docs" className="text-gray-600 hover:text-blue-600">Documentation</Link></li>
                <li><a href="#" className="text-gray-600 hover:text-blue-600">API Reference</a></li>
                <li><a href="#" className="text-gray-600 hover:text-blue-600">Status</a></li>
              </ul>
            </div>
            
            <div>
              <h4 className="font-semibold text-gray-900 mb-4">Company</h4>
              <ul className="space-y-2">
                <li><a href="#" className="text-gray-600 hover:text-blue-600">About</a></li>
                <li><a href="#" className="text-gray-600 hover:text-blue-600">Blog</a></li>
                <li><a href="#" className="text-gray-600 hover:text-blue-600">Careers</a></li>
                <li><a href="#" className="text-gray-600 hover:text-blue-600">Contact</a></li>
              </ul>
            </div>
            
            <div>
              <h4 className="font-semibold text-gray-900 mb-4">Support</h4>
              <ul className="space-y-2">
                <li><a href="#" className="text-gray-600 hover:text-blue-600">Help Center</a></li>
                <li><a href="#" className="text-gray-600 hover:text-blue-600">Community</a></li>
                <li><a href="#" className="text-gray-600 hover:text-blue-600">Privacy Policy</a></li>
                <li><a href="#" className="text-gray-600 hover:text-blue-600">Terms of Service</a></li>
              </ul>
            </div>
          </div>
          
          <div className="border-t border-gray-200 mt-8 pt-8 text-center">
            <p className="text-gray-600">
              © 2024 FormHub by Mejona Technology. All rights reserved.
            </p>
          </div>
        </div>
      </footer>
    </div>
  )
}