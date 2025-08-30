#!/usr/bin/env python3
"""
Security Test Suite for FormHub API
Tests all critical security fixes and validates proper functionality.
"""

import requests
import json
import time
import sys

BASE_URL = "http://localhost:9000/api/v1"

# Test data
VALID_ACCESS_KEY = "f6c7a044-4b28-4cfb-8c02-c24c2cece786-c496113b-176d-433c-972d-596f5028d91f"
INVALID_ACCESS_KEY = "invalid-key-123"
TEST_TOKEN = "test-token-12345"

def test_access_key_validation():
    """Test critical access key validation"""
    print("\n[TESTING] Access Key Validation...")
    
    # Test with valid access key
    valid_submission = {
        "access_key": VALID_ACCESS_KEY,
        "email": "test@example.com",
        "message": "Test message"
    }
    
    try:
        response = requests.post(f"{BASE_URL}/submit", json=valid_submission)
        print(f"Valid access key test: Status {response.status_code}")
        if response.status_code == 200:
            data = response.json()
            print(f"  Response: {data.get('success')} - {data.get('message')}")
        else:
            print(f"  Error: {response.text}")
    except Exception as e:
        print(f"  Request failed: {e}")
    
    # Test with invalid access key
    invalid_submission = {
        "access_key": INVALID_ACCESS_KEY,
        "email": "test@example.com", 
        "message": "Test message"
    }
    
    try:
        response = requests.post(f"{BASE_URL}/submit", json=invalid_submission)
        print(f"Invalid access key test: Status {response.status_code}")
        if response.status_code == 401:
            print("  PASS: Invalid access key correctly rejected")
        else:
            print(f"  FAIL: Expected 401, got {response.status_code}")
            print(f"  Response: {response.text}")
    except Exception as e:
        print(f"  Request failed: {e}")

def test_input_sanitization():
    """Test input sanitization"""
    print("\n[TESTING] Input Sanitization...")
    
    # Test XSS prevention
    xss_submission = {
        "access_key": VALID_ACCESS_KEY,
        "email": "test@example.com",
        "message": "<script>alert('xss')</script>",
        "name": "<img src=x onerror=alert('xss')>"
    }
    
    try:
        response = requests.post(f"{BASE_URL}/submit", json=xss_submission)
        print(f"XSS payload test: Status {response.status_code}")
        if response.status_code == 200:
            data = response.json()
            sanitized_data = data.get("data", {}).get("sanitized_data", {})
            print(f"  Sanitized data: {sanitized_data}")
            
            # Check if XSS was prevented
            sanitized_str = str(sanitized_data)
            if "<script>" not in sanitized_str and "<img" not in sanitized_str:
                print("  PASS: XSS payload sanitized")
            else:
                print("  FAIL: XSS payload not properly sanitized")
        else:
            print(f"  Error: {response.text}")
    except Exception as e:
        print(f"  Request failed: {e}")

def test_security_headers():
    """Test security headers"""
    print("\n[TESTING] Security Headers...")
    
    try:
        response = requests.get(f"{BASE_URL}/health")
        headers = response.headers
        
        security_headers = [
            "X-Content-Type-Options",
            "X-Frame-Options", 
            "X-XSS-Protection",
            "Referrer-Policy",
            "Content-Security-Policy",
            "Strict-Transport-Security"
        ]
        
        for header in security_headers:
            if header in headers:
                print(f"  PASS: {header} present - {headers[header]}")
            else:
                print(f"  FAIL: {header} missing")
                
    except Exception as e:
        print(f"  Request failed: {e}")

def test_authentication():
    """Test authentication requirements"""
    print("\n[TESTING] Authentication...")
    
    # Test without authorization
    try:
        response = requests.get(f"{BASE_URL}/forms")
        print(f"Unauthenticated request: Status {response.status_code}")
        if response.status_code == 401:
            print("  PASS: Unauthenticated request correctly rejected")
        else:
            print(f"  FAIL: Expected 401, got {response.status_code}")
    except Exception as e:
        print(f"  Request failed: {e}")
    
    # Test with authorization
    try:
        headers = {"Authorization": f"Bearer {TEST_TOKEN}"}
        response = requests.get(f"{BASE_URL}/forms", headers=headers)
        print(f"Authenticated request: Status {response.status_code}")
        if response.status_code == 200:
            print("  PASS: Authenticated request accepted")
        else:
            print(f"  FAIL: Expected 200, got {response.status_code}")
    except Exception as e:
        print(f"  Request failed: {e}")

def main():
    print("FormHub Security Test Suite")
    print("=" * 40)
    
    # Check if server is running
    try:
        response = requests.get(f"{BASE_URL}/health", timeout=5)
        if response.status_code == 200:
            print("Server is running and responsive")
        else:
            print("Server is not responding correctly")
            return
    except requests.RequestException:
        print("Cannot connect to FormHub API. Make sure it's running on localhost:9000")
        return
    
    # Run tests
    test_access_key_validation()
    test_input_sanitization() 
    test_security_headers()
    test_authentication()
    
    print("\n" + "=" * 40)
    print("Security tests completed!")

if __name__ == "__main__":
    main()