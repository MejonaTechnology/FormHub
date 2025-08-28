import Link from 'next/link'

export default function DocsPage() {
  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <header className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-6">
            <Link href="/" className="text-2xl font-bold text-blue-600">
              FormHub
            </Link>
            <nav className="flex space-x-8">
              <Link href="/pricing" className="text-gray-500 hover:text-blue-600">
                Pricing
              </Link>
              <Link href="/auth/login" className="text-gray-500 hover:text-blue-600">
                Sign In
              </Link>
              <Link
                href="/auth/register"
                className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700"
              >
                Get Started
              </Link>
            </nav>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-8">
          {/* Sidebar */}
          <nav className="lg:col-span-1">
            <div className="bg-gray-50 rounded-lg p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Documentation</h3>
              <ul className="space-y-2">
                <li><a href="#getting-started" className="text-blue-600 hover:text-blue-500">Getting Started</a></li>
                <li><a href="#html-forms" className="text-gray-600 hover:text-blue-600">HTML Forms</a></li>
                <li><a href="#javascript" className="text-gray-600 hover:text-blue-600">JavaScript</a></li>
                <li><a href="#react" className="text-gray-600 hover:text-blue-600">React</a></li>
                <li><a href="#api-reference" className="text-gray-600 hover:text-blue-600">API Reference</a></li>
                <li><a href="#webhooks" className="text-gray-600 hover:text-blue-600">Webhooks</a></li>
                <li><a href="#troubleshooting" className="text-gray-600 hover:text-blue-600">Troubleshooting</a></li>
              </ul>
            </div>
          </nav>

          {/* Main Content */}
          <div className="lg:col-span-3">
            <div className="prose prose-blue max-w-none">
              <h1>FormHub Documentation</h1>
              <p className="text-lg text-gray-600">
                FormHub is a powerful form backend service that handles form submissions, 
                email notifications, and more. Get started in minutes!
              </p>

              <section id="getting-started" className="mt-12">
                <h2>Getting Started</h2>
                <ol>
                  <li><Link href="/auth/register" className="text-blue-600">Create a free account</Link></li>
                  <li>Generate your first API key from the dashboard</li>
                  <li>Add the API key to your forms</li>
                  <li>Start receiving form submissions!</li>
                </ol>
              </section>

              <section id="html-forms" className="mt-12">
                <h2>HTML Forms</h2>
                <p>The easiest way to get started is with plain HTML forms:</p>
                
                <div className="bg-gray-900 text-green-400 p-6 rounded-lg text-sm font-mono overflow-x-auto">
                  <pre>{`<form action="https://formhub.mejona.in/api/v1/submit" method="POST">
  <input type="hidden" name="access_key" value="YOUR_API_KEY">
  <input type="hidden" name="subject" value="New Contact Form Submission">
  
  <label for="name">Name:</label>
  <input type="text" name="name" id="name" required>
  
  <label for="email">Email:</label>
  <input type="email" name="email" id="email" required>
  
  <label for="message">Message:</label>
  <textarea name="message" id="message" required></textarea>
  
  <button type="submit">Send Message</button>
</form>`}</pre>
                </div>

                <h3>Required Fields</h3>
                <ul>
                  <li><code>access_key</code> - Your API key from the dashboard</li>
                </ul>

                <h3>Optional Fields</h3>
                <ul>
                  <li><code>subject</code> - Custom email subject line</li>
                  <li><code>_redirect</code> - Redirect URL after successful submission</li>
                  <li><code>_template</code> - Custom email template</li>
                </ul>
              </section>

              <section id="javascript" className="mt-12">
                <h2>JavaScript Integration</h2>
                <p>For more control, use JavaScript to submit forms programmatically:</p>
                
                <div className="bg-gray-900 text-green-400 p-6 rounded-lg text-sm font-mono overflow-x-auto">
                  <pre>{`async function submitForm(formData) {
  try {
    const response = await fetch('https://formhub.mejona.in/api/v1/submit', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        access_key: 'YOUR_API_KEY',
        name: formData.name,
        email: formData.email,
        message: formData.message
      })
    });

    const result = await response.json();
    
    if (result.success) {
      alert('Message sent successfully!');
    } else {
      alert('Error: ' + result.message);
    }
  } catch (error) {
    alert('Network error. Please try again.');
  }
}`}</pre>
                </div>
              </section>

              <section id="react" className="mt-12">
                <h2>React Example</h2>
                <p>Here's a complete React form component:</p>
                
                <div className="bg-gray-900 text-green-400 p-6 rounded-lg text-sm font-mono overflow-x-auto">
                  <pre>{`import { useState } from 'react';

function ContactForm() {
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    message: ''
  });
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);

    try {
      const response = await fetch('https://formhub.mejona.in/api/v1/submit', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          access_key: 'YOUR_API_KEY',
          ...formData
        })
      });

      const result = await response.json();
      
      if (result.success) {
        alert('Message sent!');
        setFormData({ name: '', email: '', message: '' });
      }
    } catch (error) {
      alert('Error sending message');
    } finally {
      setLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <input
        type="text"
        placeholder="Name"
        value={formData.name}
        onChange={(e) => setFormData({...formData, name: e.target.value})}
        required
      />
      <input
        type="email"
        placeholder="Email"
        value={formData.email}
        onChange={(e) => setFormData({...formData, email: e.target.value})}
        required
      />
      <textarea
        placeholder="Message"
        value={formData.message}
        onChange={(e) => setFormData({...formData, message: e.target.value})}
        required
      />
      <button type="submit" disabled={loading}>
        {loading ? 'Sending...' : 'Send'}
      </button>
    </form>
  );
}`}</pre>
                </div>
              </section>

              <section id="api-reference" className="mt-12">
                <h2>API Reference</h2>
                
                <h3>Submit Form</h3>
                <div className="bg-blue-50 p-4 rounded-lg">
                  <code>POST https://formhub.mejona.in/api/v1/submit</code>
                </div>

                <h4>Request Body</h4>
                <div className="bg-gray-900 text-green-400 p-6 rounded-lg text-sm font-mono overflow-x-auto">
                  <pre>{`{
  "access_key": "your-api-key",
  "name": "John Doe",
  "email": "john@example.com",
  "message": "Hello FormHub!",
  "subject": "New Contact Form Submission" // optional
}`}</pre>
                </div>

                <h4>Response</h4>
                <div className="bg-gray-900 text-green-400 p-6 rounded-lg text-sm font-mono overflow-x-auto">
                  <pre>{`{
  "success": true,
  "message": "Form submitted successfully"
}`}</pre>
                </div>
              </section>

              <section id="webhooks" className="mt-12">
                <h2>Webhooks</h2>
                <p>
                  Configure webhooks to receive real-time notifications when forms are submitted. 
                  Set up webhook URLs in your dashboard.
                </p>

                <h3>Webhook Payload</h3>
                <div className="bg-gray-900 text-green-400 p-6 rounded-lg text-sm font-mono overflow-x-auto">
                  <pre>{`{
  "event": "form_submission",
  "timestamp": "2024-01-01T00:00:00Z",
  "form_id": "form-uuid",
  "data": {
    "name": "John Doe",
    "email": "john@example.com",
    "message": "Hello FormHub!"
  }
}`}</pre>
                </div>
              </section>

              <section id="troubleshooting" className="mt-12">
                <h2>Troubleshooting</h2>
                
                <h3>Common Issues</h3>
                <div className="space-y-4">
                  <div className="border-l-4 border-yellow-400 pl-4">
                    <h4 className="font-semibold">Form not submitting</h4>
                    <p>Check that your API key is correct and active in your dashboard.</p>
                  </div>
                  
                  <div className="border-l-4 border-yellow-400 pl-4">
                    <h4 className="font-semibold">Not receiving emails</h4>
                    <p>Verify your email settings in the form configuration and check spam folder.</p>
                  </div>
                  
                  <div className="border-l-4 border-yellow-400 pl-4">
                    <h4 className="font-semibold">CORS errors</h4>
                    <p>Make sure your domain is added to the allowed origins in your form settings.</p>
                  </div>
                </div>

                <h3>Need Help?</h3>
                <p>
                  Can't find what you're looking for? <a href="mailto:support@mejona.tech" className="text-blue-600">Contact our support team</a> and we'll help you get set up.
                </p>
              </section>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}