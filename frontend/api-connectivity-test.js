// FormHub API Connectivity Test
// Run with: node api-connectivity-test.js

const API_BASE_URL = 'http://13.127.59.135:9000/api/v1';

async function testApiEndpoint(endpoint, method = 'GET', data = null) {
  const url = `${API_BASE_URL}${endpoint}`;
  const options = {
    method,
    headers: {
      'Content-Type': 'application/json',
    },
  };

  if (data) {
    options.body = JSON.stringify(data);
  }

  try {
    console.log(`🔍 Testing ${method} ${endpoint}...`);
    const response = await fetch(url, options);
    const responseText = await response.text();
    
    let responseData;
    try {
      responseData = JSON.parse(responseText);
    } catch {
      responseData = responseText;
    }

    console.log(`   Status: ${response.status} ${response.statusText}`);
    console.log(`   Response: ${JSON.stringify(responseData, null, 2)}`);
    console.log('');
    
    return {
      success: response.ok,
      status: response.status,
      data: responseData
    };
  } catch (error) {
    console.log(`   ❌ Error: ${error.message}`);
    console.log('');
    return {
      success: false,
      error: error.message
    };
  }
}

async function runConnectivityTests() {
  console.log('=====================================');
  console.log('   FormHub API Connectivity Test');
  console.log('=====================================');
  console.log('');

  const tests = [
    // Basic connectivity
    { endpoint: '/health', description: 'Health Check' },
    { endpoint: '/status', description: 'Status Check' },
    
    // Authentication endpoints
    { endpoint: '/auth/register', method: 'POST', data: {
        email: 'test@example.com',
        password: 'testpassword123',
        first_name: 'Test',
        last_name: 'User'
      }, description: 'User Registration Test' 
    },
    
    { endpoint: '/auth/login', method: 'POST', data: {
        email: 'test@example.com', 
        password: 'testpassword123'
      }, description: 'User Login Test'
    },
    
    // Form submission test
    { endpoint: '/submit', method: 'POST', data: {
        name: 'Test User',
        email: 'test@example.com',
        message: 'Test message from connectivity test'
      }, description: 'Form Submission Test'
    }
  ];

  let passedTests = 0;
  let totalTests = tests.length;

  for (const test of tests) {
    console.log(`📋 ${test.description}`);
    const result = await testApiEndpoint(
      test.endpoint, 
      test.method || 'GET', 
      test.data || null
    );
    
    if (result.success) {
      passedTests++;
      console.log('   ✅ PASSED');
    } else {
      console.log('   ❌ FAILED');
    }
    
    console.log('   ' + '-'.repeat(40));
    console.log('');
    
    // Small delay between tests
    await new Promise(resolve => setTimeout(resolve, 1000));
  }

  console.log('=====================================');
  console.log(`📊 Test Results: ${passedTests}/${totalTests} passed`);
  
  if (passedTests === totalTests) {
    console.log('🎉 All API connectivity tests PASSED!');
    console.log('✅ Frontend can successfully connect to backend');
  } else {
    console.log('⚠️  Some tests failed. Check backend service status.');
    console.log('🔧 Troubleshooting steps:');
    console.log('   1. Verify backend is running on port 9000');
    console.log('   2. Check firewall settings');
    console.log('   3. Verify database connectivity');
    console.log('   4. Check backend logs for errors');
  }
  
  console.log('=====================================');
}

// Add fetch polyfill for Node.js if needed
if (typeof fetch === 'undefined') {
  global.fetch = require('node-fetch');
}

// Run the tests
runConnectivityTests().catch(console.error);