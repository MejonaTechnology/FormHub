#!/usr/bin/env python3
"""
Simple Web3Forms Bug Fix Test
Tests the auto-form creation functionality with the provided API key
"""

import requests
import json

# Production backend URL
BASE_URL = "http://13.127.59.135:9000"
API_BASE = f"{BASE_URL}/api/v1"

# User's actual API key
API_KEY = "ee48ba7c-a5f6-4a6d-a560-4b02bd0a3bdd-c133f5d0-cb9b-4798-8f15-5b47fa0e726a"

def test_health():
    print("Testing backend health...")
    try:
        response = requests.get(f"{BASE_URL}/health", timeout=10)
        if response.status_code == 200:
            print("[OK] Backend is running")
            return True
        else:
            print(f"[ERROR] Health check failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"[ERROR] Cannot connect: {e}")
        return False

def test_submission():
    print("Testing form submission with auto-form creation...")
    
    submission_data = {
        "access_key": API_KEY,
        "email": "test@example.com",
        "subject": "Test Auto-Form Creation",
        "message": "Testing the bug fix for auto-form creation",
        "name": "Test User"
    }
    
    try:
        response = requests.post(f"{API_BASE}/submit", json=submission_data, timeout=15)
        
        print(f"Response Status: {response.status_code}")
        
        if response.status_code == 200:
            data = response.json()
            print("[SUCCESS] Form submission completed!")
            print(f"Success: {data.get('success', False)}")
            print(f"Message: {data.get('message', 'No message')}")
            return True
        else:
            print(f"[ERROR] Submission failed: {response.status_code}")
            try:
                error_data = response.json()
                print(f"Error: {error_data}")
            except:
                print(f"Error text: {response.text}")
            return False
            
    except Exception as e:
        print(f"[ERROR] Request failed: {e}")
        return False

def main():
    print("=" * 50)
    print("Web3Forms Bug Fix Test")
    print("=" * 50)
    
    # Test 1: Health check
    if not test_health():
        print("Backend is not accessible. Exiting.")
        return False
    
    print()
    
    # Test 2: Form submission
    result = test_submission()
    
    print()
    print("=" * 50)
    
    if result:
        print("SUCCESS: Bug fix appears to be working!")
        print("Auto-form creation is functional.")
    else:
        print("FAILED: Bug still exists.")
        print("Auto-form creation needs more work.")
    
    return result

if __name__ == "__main__":
    success = main()
    exit(0 if success else 1)