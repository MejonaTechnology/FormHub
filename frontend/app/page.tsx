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
              <h2 className="text-2xl font-semibold text-green-900 mb-4">Features</h2>
              <ul className="text-green-800 space-y-3">
                <li className="flex items-center">
                  <span className="text-green-600 font-bold mr-3 text-lg">✓</span>
                  Email notifications
                </li>
                <li className="flex items-center">
                  <span className="text-green-600 font-bold mr-3 text-lg">✓</span>
                  Spam protection
                </li>
                <li className="flex items-center">
                  <span className="text-green-600 font-bold mr-3 text-lg">✓</span>
                  File uploads
                </li>
                <li className="flex items-center">
                  <span className="text-green-600 font-bold mr-3 text-lg">✓</span>
                  Webhook integration
                </li>
                <li className="flex items-center">
                  <span className="text-green-600 font-bold mr-3 text-lg">✓</span>
                  Rate limiting
                </li>
                <li className="flex items-center">
                  <span className="text-green-600 font-bold mr-3 text-lg">✓</span>
                  RESTful API
                </li>
              </ul>
            </div>
          </div>
          
          <div className="mt-16">
            <div className="bg-green-50 border border-green-300 p-8 rounded-lg shadow-lg max-w-4xl mx-auto">
              <h3 className="text-2xl font-semibold text-green-900 mb-4">✅ HTTPS Backend Ready</h3>
              <p className="text-green-800 mb-6">
                FormHub backend is deployed with SSL certificate and fully operational on AWS EC2. 
                All API endpoints are secured and ready for GitHub Pages integration.
              </p>
              
              <div className="grid md:grid-cols-2 gap-4 mb-6">
                <div className="bg-white p-4 rounded-lg border border-green-200">
                  <h4 className="font-semibold text-green-900 mb-2">🔐 HTTPS Endpoints</h4>
                  <ul className="text-green-700 text-sm space-y-1">
                    <li>• Form Submission API</li>
                    <li>• User Authentication</li>
                    <li>• Form Management</li>
                    <li>• API Key Management</li>
                  </ul>
                </div>
                <div className="bg-white p-4 rounded-lg border border-green-200">
                  <h4 className="font-semibold text-green-900 mb-2">🔧 System Status</h4>
                  <ul className="text-green-700 text-sm space-y-1">
                    <li>• AWS EC2: Online</li>
                    <li>• MariaDB: Connected</li>
                    <li>• Redis Cache: Active</li>
                    <li>• SSL: Let's Encrypt</li>
                  </ul>
                </div>
              </div>
              
              <div className="bg-green-100 p-4 rounded-lg mb-4">
                <p className="text-green-800 text-sm">
                  <strong>HTTPS API Endpoint:</strong> https://formhub.mejona.in/api/v1<br/>
                  <strong>SSL Certificate:</strong> Let's Encrypt (Auto-renewing)<br/>
                  <strong>Status:</strong> Ready for secure GitHub Pages integration
                </p>
              </div>
              
              <div className="bg-gray-900 text-green-400 px-6 py-4 rounded-md text-sm font-mono overflow-x-auto">
                curl -X POST https://formhub.mejona.in/api/v1/submit -H "Content-Type: application/json" -d '{"access_key": "test-key-123", "name": "Test User", "email": "test@example.com", "message": "Hello FormHub!"}'
              </div>
            </div>
          </div>
          
          <footer className="mt-16 text-center bg-gray-800 py-8 rounded-lg">
            <p className="text-gray-200 text-lg">© 2025 FormHub by <span className="text-blue-400 font-semibold">Mejona Technology LLP</span>. All rights reserved.</p>
          </footer>
        </div>
      </div>
    </div>
  )
}