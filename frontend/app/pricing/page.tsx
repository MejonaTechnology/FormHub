import Link from 'next/link'

const plans = [
  {
    name: 'Free',
    price: 0,
    description: 'Perfect for testing and small projects',
    features: [
      '1,000 submissions/month',
      'Email notifications',
      'Basic spam protection',
      'API access',
      'Community support'
    ],
    buttonText: 'Get Started',
    buttonHref: '/auth/register',
    popular: false
  },
  {
    name: 'Pro',
    price: 19,
    description: 'Great for growing businesses',
    features: [
      '10,000 submissions/month',
      'Email notifications',
      'Advanced spam protection',
      'File uploads (up to 10MB)',
      'Webhooks',
      'Custom thank you pages',
      'Priority support'
    ],
    buttonText: 'Upgrade to Pro',
    buttonHref: '/dashboard',
    popular: true
  },
  {
    name: 'Business',
    price: 49,
    description: 'For high-volume applications',
    features: [
      '50,000 submissions/month',
      'Everything in Pro',
      'File uploads (up to 25MB)',
      'Custom domains',
      'Advanced analytics',
      'Team collaboration',
      'Phone support',
      '99.9% uptime SLA'
    ],
    buttonText: 'Upgrade to Business',
    buttonHref: '/dashboard',
    popular: false
  },
  {
    name: 'Enterprise',
    price: null,
    description: 'Custom solutions for large organizations',
    features: [
      'Unlimited submissions',
      'Everything in Business',
      'Dedicated infrastructure',
      'Custom integrations',
      'On-premise deployment option',
      'Dedicated account manager',
      '24/7 premium support',
      'Custom SLA'
    ],
    buttonText: 'Contact Sales',
    buttonHref: 'mailto:sales@mejona.tech',
    popular: false
  }
]

export default function PricingPage() {
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
              <Link href="/docs" className="text-gray-500 hover:text-blue-600">
                Documentation
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

      {/* Hero Section */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-24">
        <div className="text-center">
          <h1 className="text-4xl font-bold text-gray-900 sm:text-5xl">
            Simple, transparent pricing
          </h1>
          <p className="mt-4 text-xl text-gray-600 max-w-3xl mx-auto">
            Choose the perfect plan for your needs. Start free and upgrade as you grow.
          </p>
        </div>

        {/* Pricing Cards */}
        <div className="mt-16 grid grid-cols-1 gap-8 lg:grid-cols-4">
          {plans.map((plan) => (
            <div
              key={plan.name}
              className={`relative rounded-2xl border ${
                plan.popular
                  ? 'border-blue-500 shadow-2xl scale-105'
                  : 'border-gray-200 shadow-lg'
              } bg-white p-8`}
            >
              {plan.popular && (
                <div className="absolute -top-5 left-0 right-0 flex justify-center">
                  <span className="bg-blue-500 text-white px-4 py-1 text-sm font-medium rounded-full">
                    Most Popular
                  </span>
                </div>
              )}

              <div className="text-center">
                <h3 className="text-2xl font-bold text-gray-900">{plan.name}</h3>
                <p className="mt-2 text-gray-600">{plan.description}</p>
                
                <div className="mt-6">
                  {plan.price === null ? (
                    <span className="text-4xl font-bold text-gray-900">Custom</span>
                  ) : plan.price === 0 ? (
                    <span className="text-4xl font-bold text-gray-900">Free</span>
                  ) : (
                    <>
                      <span className="text-4xl font-bold text-gray-900">${plan.price}</span>
                      <span className="text-gray-600">/month</span>
                    </>
                  )}
                </div>

                <a
                  href={plan.buttonHref}
                  className={`mt-8 block w-full rounded-md px-6 py-3 text-center text-sm font-semibold ${
                    plan.popular
                      ? 'bg-blue-600 text-white hover:bg-blue-700'
                      : 'bg-blue-50 text-blue-700 hover:bg-blue-100'
                  } transition-colors`}
                >
                  {plan.buttonText}
                </a>
              </div>

              <ul className="mt-8 space-y-3">
                {plan.features.map((feature) => (
                  <li key={feature} className="flex items-center">
                    <svg
                      className="h-5 w-5 text-green-500 mr-3"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M5 13l4 4L19 7"
                      />
                    </svg>
                    <span className="text-gray-700">{feature}</span>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        {/* FAQ Section */}
        <div className="mt-24">
          <h2 className="text-3xl font-bold text-center text-gray-900 mb-12">
            Frequently Asked Questions
          </h2>
          
          <div className="max-w-3xl mx-auto">
            <div className="space-y-8">
              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-2">
                  What happens if I exceed my plan limits?
                </h3>
                <p className="text-gray-600">
                  We'll notify you when you're approaching your limits. You can upgrade your plan 
                  at any time to avoid service interruption.
                </p>
              </div>
              
              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-2">
                  Can I change or cancel my plan at any time?
                </h3>
                <p className="text-gray-600">
                  Yes! You can upgrade, downgrade, or cancel your subscription at any time from 
                  your dashboard. Changes take effect at your next billing cycle.
                </p>
              </div>
              
              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-2">
                  Do you offer refunds?
                </h3>
                <p className="text-gray-600">
                  We offer a 30-day money-back guarantee for all paid plans. Contact our support 
                  team if you're not satisfied with the service.
                </p>
              </div>
              
              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-2">
                  Is my data secure?
                </h3>
                <p className="text-gray-600">
                  Absolutely! We use industry-standard encryption and security measures to protect 
                  your data. All form submissions are encrypted in transit and at rest.
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* CTA Section */}
        <div className="mt-24 text-center">
          <h2 className="text-3xl font-bold text-gray-900">
            Ready to get started?
          </h2>
          <p className="mt-4 text-xl text-gray-600">
            Join thousands of developers who trust FormHub for their form handling needs.
          </p>
          <div className="mt-8 flex justify-center space-x-4">
            <Link
              href="/auth/register"
              className="bg-blue-600 text-white px-8 py-3 rounded-md text-lg font-semibold hover:bg-blue-700 transition-colors"
            >
              Start Free Trial
            </Link>
            <Link
              href="/docs"
              className="bg-gray-100 text-gray-900 px-8 py-3 rounded-md text-lg font-semibold hover:bg-gray-200 transition-colors"
            >
              View Documentation
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}