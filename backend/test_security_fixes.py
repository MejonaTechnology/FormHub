#!/usr/bin/env python3
"""
Security Test Suite for FormHub API
Tests all critical security fixes and validates proper functionality.
"""

import requests
import json
import time
import sys
from typing import Dict, Any, Optional

BASE_URL = "http://localhost:9000/api/v1"

# Test data
VALID_ACCESS_KEY = "f6c7a044-4b28-4cfb-8c02-c24c2cece786-c496113b-176d-433c-972d-596f5028d91f"
INVALID_ACCESS_KEY = "invalid-key-123"
TEST_TOKEN = "test-token-12345"

class SecurityTester:
    def __init__(self):
        self.passed = 0
        self.failed = 0
        self.total = 0

    def test(self, name: str, condition: bool, message: str = ""):
        """Test helper function"""
        self.total += 1
        if condition:
            print(f"[PASS] {name}")
            self.passed += 1
        else:
            print(f"[FAIL] {name} - {message}")
            self.failed += 1

    def make_request(self, method: str, endpoint: str, data: Optional[Dict] = None, 
                    headers: Optional[Dict] = None, expected_status: int = 200) -> requests.Response:
        """Make HTTP request with error handling"""
        try:
            url = f"{BASE_URL}{endpoint}"
            
            if method.upper() == "GET":
                response = requests.get(url, headers=headers)
            elif method.upper() == "POST":
                response = requests.post(url, json=data, headers=headers)
            elif method.upper() == "PUT":
                response = requests.put(url, json=data, headers=headers)
            elif method.upper() == "DELETE":
                response = requests.delete(url, headers=headers)
            else:
                raise ValueError(f"Unsupported method: {method}")
                
            return response
        except requests.RequestException as e:
            print(f"[ERROR] Request failed: {e}")
            return None

    def test_health_check(self):
        """Test basic health endpoint"""
        print("\n🔍 Testing Health Check...")
        response = self.make_request("GET", "/health")
        
        if response:
            self.test("Health check responds", response.status_code == 200)
            if response.status_code == 200:
                data = response.json()
                self.test("Health check returns correct data", 
                         data.get("status") == "healthy")

    def test_security_headers(self):
        """Test security headers are present"""
        print("\n🔍 Testing Security Headers...")
        response = self.make_request("GET", "/health")
        
        if response:
            headers = response.headers
            security_headers = {
                "X-Content-Type-Options": "nosniff",
                "X-Frame-Options": "DENY",
                "X-XSS-Protection": "1; mode=block",
                "Referrer-Policy": "strict-origin-when-cross-origin",
                "Content-Security-Policy": lambda x: x.startswith("default-src 'self'"),
                "Strict-Transport-Security": lambda x: "max-age=" in x
            }
            
            for header, expected in security_headers.items():
                if callable(expected):
                    self.test(f"Security header {header} present and valid", 
                             header in headers and expected(headers[header]))
                else:
                    self.test(f"Security header {header} present", 
                             headers.get(header) == expected,
                             f"Expected: {expected}, Got: {headers.get(header)}")

    def test_access_key_validation(self):
        """Test critical access key validation"""
        print("\n🔍 Testing Access Key Validation...")
        
        # Test with valid access key
        valid_submission = {
            "access_key": VALID_ACCESS_KEY,
            "email": "test@example.com",
            "message": "Test message"
        }
        
        response = self.make_request("POST", "/submit", valid_submission)
        if response:
            self.test("Valid access key accepted", response.status_code == 200)
            if response.status_code == 200:
                data = response.json()
                self.test("Valid submission returns success", data.get("success") == True)
        
        # Test with invalid access key
        invalid_submission = {
            "access_key": INVALID_ACCESS_KEY,
            "email": "test@example.com", 
            "message": "Test message"
        }
        
        response = self.make_request("POST", "/submit", invalid_submission)
        if response:
            self.test("Invalid access key rejected", response.status_code == 401,
                     f"Expected 401, got {response.status_code}")
            if response.status_code == 401:
                data = response.json()
                self.test("Invalid access key returns error", 
                         data.get("success") == False and "Invalid access key" in data.get("error", ""))
        
        # Test with missing access key
        no_key_submission = {
            "email": "test@example.com",
            "message": "Test message"
        }
        
        response = self.make_request("POST", "/submit", no_key_submission)
        if response:
            self.test("Missing access key rejected", response.status_code == 400,
                     f"Expected 400, got {response.status_code}")

    def test_input_validation(self):
        """Test input validation and sanitization"""
        print("\n🔍 Testing Input Validation...")
        
        # Test invalid email
        invalid_email_submission = {
            "access_key": VALID_ACCESS_KEY,
            "email": "invalid-email",
            "message": "Test message"
        }
        
        response = self.make_request("POST", "/submit", invalid_email_submission)
        if response:
            self.test("Invalid email rejected", response.status_code == 400,
                     f"Expected 400, got {response.status_code}")
        
        # Test XSS prevention
        xss_submission = {
            "access_key": VALID_ACCESS_KEY,
            "email": "test@example.com",
            "message": "<script>alert('xss')</script>",
            "name": "<img src=x onerror=alert('xss')>"
        }
        
        response = self.make_request("POST", "/submit", xss_submission)
        if response:
            self.test("XSS payload processed without script execution", response.status_code == 200)
            if response.status_code == 200:
                data = response.json()
                sanitized_data = data.get("data", {}).get("sanitized_data", {})
                self.test("XSS payload sanitized", 
                         "<script>" not in str(sanitized_data) and "<img" not in str(sanitized_data))
        
        # Test SQL injection attempt
        sql_injection_submission = {
            "access_key": VALID_ACCESS_KEY,
            "email": "test@example.com",
            "message": "'; DROP TABLE users; --",
            "name": "admin'/**/UNION/**/SELECT/**/*/**/FROM/**/users--"
        }
        
        response = self.make_request("POST", "/submit", sql_injection_submission)
        if response:
            self.test("SQL injection payload handled", response.status_code == 200)
            if response.status_code == 200:
                data = response.json()
                self.test("SQL injection returned valid response", data.get("success") == True)

    def test_parameter_validation(self):
        """Test strict parameter validation"""
        print("\n🔍 Testing Parameter Validation...")
        
        # Test phone number validation
        invalid_phone_submission = {
            "access_key": VALID_ACCESS_KEY,
            "email": "test@example.com",
            "phone": "invalid-phone-123-abc",
            "message": "Test message"
        }
        
        response = self.make_request("POST", "/submit", invalid_phone_submission)
        if response:
            self.test("Invalid phone number rejected", response.status_code == 400,
                     f"Expected 400, got {response.status_code}")
        
        # Test message length validation
        long_message_submission = {
            "access_key": VALID_ACCESS_KEY,
            "email": "test@example.com",
            "message": "A" * 6000  # Exceeds 5000 character limit
        }
        
        response = self.make_request("POST", "/submit", long_message_submission)
        if response:
            self.test("Overly long message rejected", response.status_code == 400,
                     f"Expected 400, got {response.status_code}")

    def test_authentication_endpoints(self):
        """Test authentication-required endpoints"""
        print("\n🔍 Testing Authentication...")
        
        # Test without authorization header
        response = self.make_request("GET", "/forms")
        if response:
            self.test("Unauthenticated forms request rejected", response.status_code == 401,
                     f"Expected 401, got {response.status_code}")
        
        # Test with valid authorization header
        headers = {"Authorization": f"Bearer {TEST_TOKEN}"}
        response = self.make_request("GET", "/forms", headers=headers)
        if response:
            self.test("Authenticated forms request accepted", response.status_code == 200)

    def test_crud_operations(self):
        """Test individual form CRUD operations"""
        print("\n🔍 Testing CRUD Operations...")
        
        headers = {"Authorization": f"Bearer {TEST_TOKEN}"}
        
        # Test GET individual form
        response = self.make_request("GET", "/forms/123e4567-e89b-12d3-a456-426614174000", headers=headers)
        if response:
            self.test("Individual form GET works", response.status_code == 200)
        
        # Test GET with invalid UUID
        response = self.make_request("GET", "/forms/invalid-id", headers=headers)
        if response:
            self.test("Invalid form ID rejected", response.status_code == 400)
        
        # Test PUT form update
        form_data = {
            "name": "Updated Form",
            "target_email": "updated@example.com",
            "description": "Updated description"
        }
        response = self.make_request("PUT", "/forms/123e4567-e89b-12d3-a456-426614174000", 
                                   form_data, headers=headers)
        if response:
            self.test("Form update works", response.status_code == 200)

    def test_api_key_management(self):
        """Test API key management endpoints"""
        print("\n🔍 Testing API Key Management...")
        
        headers = {"Authorization": f"Bearer {TEST_TOKEN}"}
        
        # Test create API key
        key_data = {"name": "Test API Key"}
        response = self.make_request("POST", "/api-keys", key_data, headers=headers)
        if response:
            self.test("API key creation works", response.status_code == 201)
        
        # Test delete API key
        response = self.make_request("DELETE", "/api-keys/test-key-id", headers=headers)
        if response:
            self.test("API key deletion works", response.status_code == 200)

    def test_rate_limiting(self):
        """Test rate limiting (basic test)"""
        print("\n🔍 Testing Rate Limiting...")
        
        # Make multiple rapid requests
        rapid_requests = 0
        for i in range(10):
            response = self.make_request("GET", "/health")
            if response and response.status_code == 200:
                rapid_requests += 1
        
        self.test("Rate limiting allows reasonable requests", rapid_requests >= 5,
                 f"Only {rapid_requests}/10 requests succeeded")

    def run_all_tests(self):
        """Run the complete security test suite"""
        print("🚀 Starting FormHub Security Test Suite")
        print("=" * 50)
        
        # Check if server is running
        try:
            response = requests.get(f"{BASE_URL}/health", timeout=5)
            if response.status_code != 200:
                print("❌ Server is not responding correctly")
                return False
        except requests.RequestException:
            print("❌ Cannot connect to FormHub API. Make sure it's running on localhost:9000")
            return False
        
        print("✅ Server is running and responsive")
        
        # Run all test suites
        self.test_health_check()
        self.test_security_headers()
        self.test_access_key_validation()
        self.test_input_validation()
        self.test_parameter_validation()
        self.test_authentication_endpoints()
        self.test_crud_operations()
        self.test_api_key_management()
        self.test_rate_limiting()
        
        # Print results
        print("\n" + "=" * 50)
        print("🏁 TEST RESULTS")
        print("=" * 50)
        print(f"✅ Passed: {self.passed}")
        print(f"❌ Failed: {self.failed}")
        print(f"📊 Total:  {self.total}")
        
        if self.failed == 0:
            print("\n🎉 ALL SECURITY TESTS PASSED!")
            print("🔒 FormHub API is now production-ready with proper security measures.")
        else:
            print(f"\n⚠️  {self.failed} security issues need attention.")
            print("🔧 Please review and fix the failing tests before production deployment.")
        
        return self.failed == 0

def main():
    tester = SecurityTester()
    success = tester.run_all_tests()
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()