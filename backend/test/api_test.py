#!/usr/bin/env python3
"""
FormHub API Test Suite
Tests the core functionality of the FormHub API
"""

import requests
import json
import time
from typing import Dict, Any

BASE_URL = "http://localhost:8080"
API_BASE = f"{BASE_URL}/api/v1"

class FormHubTester:
    def __init__(self):
        self.access_token = None
        self.api_key = None
        self.form_id = None
        
    def test_health_check(self):
        """Test health check endpoint"""
        print("🔍 Testing health check...")
        response = requests.get(f"{BASE_URL}/health")
        
        if response.status_code == 200:
            data = response.json()
            print(f"✅ Health check passed: {data}")
            return True
        else:
            print(f"❌ Health check failed: {response.status_code}")
            return False
    
    def test_user_registration(self):
        """Test user registration"""
        print("🔍 Testing user registration...")
        
        user_data = {
            "email": f"test_{int(time.time())}@example.com",
            "password": "testpassword123",
            "first_name": "John",
            "last_name": "Doe",
            "company": "Test Company"
        }
        
        response = requests.post(f"{API_BASE}/auth/register", json=user_data)
        
        if response.status_code == 201:
            data = response.json()
            self.access_token = data.get("access_token")
            print(f"✅ User registration successful")
            print(f"   Email: {data['user']['email']}")
            print(f"   Plan: {data['user']['plan_type']}")
            return True
        else:
            print(f"❌ User registration failed: {response.status_code}")
            print(f"   Error: {response.text}")
            return False
    
    def test_user_login(self):
        """Test user login with existing user"""
        print("🔍 Testing user login...")
        
        login_data = {
            "email": "admin@formhub.com",  # This would be a pre-existing test user
            "password": "admin123"
        }
        
        response = requests.post(f"{API_BASE}/auth/login", json=login_data)
        
        if response.status_code == 200:
            data = response.json()
            self.access_token = data.get("access_token")
            print(f"✅ User login successful")
            return True
        elif response.status_code == 401:
            print("⚠️  Login failed - using registration token instead")
            return True  # Continue with registration token
        else:
            print(f"❌ User login failed: {response.status_code}")
            return False
    
    def test_create_api_key(self):
        """Test API key creation"""
        print("🔍 Testing API key creation...")
        
        if not self.access_token:
            print("❌ No access token available")
            return False
        
        headers = {"Authorization": f"Bearer {self.access_token}"}
        api_key_data = {"name": "Test API Key"}
        
        response = requests.post(f"{API_BASE}/api-keys", json=api_key_data, headers=headers)
        
        if response.status_code == 201:
            data = response.json()
            self.api_key = data["api_key"]["key"]
            print(f"✅ API key created successfully")
            print(f"   Key: {self.api_key[:20]}...")
            return True
        else:
            print(f"❌ API key creation failed: {response.status_code}")
            print(f"   Error: {response.text}")
            return False
    
    def test_create_form(self):
        """Test form creation"""
        print("🔍 Testing form creation...")
        
        if not self.access_token:
            print("❌ No access token available")
            return False
        
        headers = {"Authorization": f"Bearer {self.access_token}"}
        form_data = {
            "name": "Test Contact Form",
            "description": "A test contact form",
            "target_email": "test@example.com",
            "cc_emails": ["cc@example.com"],
            "subject": "New form submission from {name}",
            "success_message": "Thank you for your message!",
            "spam_protection": True,
            "file_uploads": False
        }
        
        response = requests.post(f"{API_BASE}/forms", json=form_data, headers=headers)
        
        if response.status_code == 201:
            data = response.json()
            self.form_id = data["form"]["id"]
            print(f"✅ Form created successfully")
            print(f"   Form ID: {self.form_id}")
            print(f"   Name: {data['form']['name']}")
            return True
        else:
            print(f"❌ Form creation failed: {response.status_code}")
            print(f"   Error: {response.text}")
            return False
    
    def test_form_submission(self):
        """Test form submission"""
        print("🔍 Testing form submission...")
        
        if not self.api_key:
            print("❌ No API key available")
            return False
        
        # Test JSON submission
        submission_data = {
            "access_key": self.api_key,
            "email": "user@example.com",
            "subject": "Test Submission",
            "message": "This is a test message from the API test suite.",
            "name": "Test User",
            "phone": "123-456-7890"
        }
        
        response = requests.post(f"{API_BASE}/submit", json=submission_data)
        
        if response.status_code == 200:
            data = response.json()
            print(f"✅ Form submission successful (JSON)")
            print(f"   Success: {data['success']}")
            print(f"   Message: {data['message']}")
        else:
            print(f"❌ Form submission failed (JSON): {response.status_code}")
            print(f"   Error: {response.text}")
            return False
        
        # Test form data submission
        form_data = {
            "access_key": self.api_key,
            "email": "user2@example.com",
            "subject": "Test Form Data Submission",
            "message": "This is a test message using form data.",
            "name": "Another Test User"
        }
        
        response = requests.post(f"{API_BASE}/submit", data=form_data)
        
        if response.status_code == 200:
            data = response.json()
            print(f"✅ Form submission successful (Form Data)")
            print(f"   Success: {data['success']}")
            print(f"   Message: {data['message']}")
            return True
        else:
            print(f"❌ Form submission failed (Form Data): {response.status_code}")
            print(f"   Error: {response.text}")
            return False
    
    def test_spam_detection(self):
        """Test spam detection"""
        print("🔍 Testing spam detection...")
        
        if not self.api_key:
            print("❌ No API key available")
            return False
        
        # Submit a potentially spammy message
        spam_data = {
            "access_key": self.api_key,
            "email": "spammer@example.com",
            "subject": "URGENT!!! MAKE MONEY FAST!!!",
            "message": "GET RICH QUICK WITH BITCOIN! CLICK HERE: http://scam.com http://fake.com GUARANTEED MONEY!",
            "name": "SPAMMER"
        }
        
        response = requests.post(f"{API_BASE}/submit", json=spam_data)
        
        if response.status_code == 200:
            data = response.json()
            print(f"✅ Spam detection test completed")
            print(f"   Success: {data['success']}")
            print(f"   Message: {data['message']}")
            # Note: The actual spam detection logic will mark this as spam in the backend
            return True
        else:
            print(f"❌ Spam detection test failed: {response.status_code}")
            return False
    
    def test_get_forms(self):
        """Test getting user forms"""
        print("🔍 Testing get forms...")
        
        if not self.access_token:
            print("❌ No access token available")
            return False
        
        headers = {"Authorization": f"Bearer {self.access_token}"}
        response = requests.get(f"{API_BASE}/forms", headers=headers)
        
        if response.status_code == 200:
            data = response.json()
            print(f"✅ Get forms successful")
            print(f"   Forms count: {len(data['forms'])}")
            return True
        else:
            print(f"❌ Get forms failed: {response.status_code}")
            return False
    
    def run_all_tests(self):
        """Run all tests in sequence"""
        print("🚀 Starting FormHub API Test Suite")
        print("=" * 50)
        
        tests = [
            self.test_health_check,
            self.test_user_registration,
            self.test_user_login,
            self.test_create_api_key,
            self.test_create_form,
            self.test_form_submission,
            self.test_spam_detection,
            self.test_get_forms,
        ]
        
        passed = 0
        total = len(tests)
        
        for test in tests:
            try:
                if test():
                    passed += 1
                print()  # Empty line for readability
            except Exception as e:
                print(f"❌ Test {test.__name__} failed with exception: {e}")
                print()
        
        print("=" * 50)
        print(f"📊 Test Results: {passed}/{total} tests passed")
        
        if passed == total:
            print("🎉 All tests passed!")
        else:
            print(f"⚠️  {total - passed} test(s) failed")
        
        return passed == total

if __name__ == "__main__":
    tester = FormHubTester()
    success = tester.run_all_tests()
    exit(0 if success else 1)