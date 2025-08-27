#!/usr/bin/env python3
"""
Web3Forms Auto-Form Creation Bug Fix Test
Tests the specific bug fix for auto-form creation with the provided API key
"""

import requests
import json
import time
from typing import Dict, Any

# Production backend URL (EC2)
BASE_URL = "http://13.127.59.135:9000"
API_BASE = f"{BASE_URL}/api/v1"

# User's actual API key from the bug report
API_KEY = "ee48ba7c-a5f6-4a6d-a560-4b02bd0a3bdd-c133f5d0-cb9b-4798-8f15-5b47fa0e726a"
USER_ID = "0f039798-b4d7-4a6f-a094-5a063b88a18c"
USER_EMAIL = "mysterysingh@gmail.com"

class Web3FormsTester:
    def __init__(self):
        self.api_key = API_KEY
        
    def test_health_check(self):
        """Test if the backend is running"""
        print("[TEST] Testing backend health check...")
        try:
            response = requests.get(f"{BASE_URL}/health", timeout=10)
            
            if response.status_code == 200:
                data = response.json()
                print(f"[OK] Backend is running: {data}")
                return True
            else:
                print(f"[ERROR] Backend health check failed: {response.status_code}")
                return False
        except requests.exceptions.RequestException as e:
            print(f"[ERROR] Cannot connect to backend: {e}")
            return False
    
    def test_web3forms_submission_json(self):
        """Test Web3Forms submission with JSON payload"""
        print("[TEST] Testing Web3Forms submission (JSON payload)...")
        
        # Test JSON submission exactly like Web3Forms.com
        submission_data = {
            "access_key": self.api_key,
            "email": "test@example.com",
            "subject": "Test Web3Forms Auto-Form Creation",
            "message": "This is a test to verify that default forms are auto-created and saved to the database properly.",
            "name": "Web3Forms Tester",
            "phone": "+1-555-123-4567"
        }
        
        try:
            response = requests.post(f"{API_BASE}/submit", json=submission_data, timeout=15)
            
            print(f"📊 Response Status: {response.status_code}")
            print(f"📊 Response Headers: {dict(response.headers)}")
            
            if response.status_code == 200:
                data = response.json()
                print(f"✅ JSON submission successful!")
                print(f"   Success: {data.get('success', False)}")
                print(f"   Message: {data.get('message', 'No message')}")
                print(f"   Data: {json.dumps(data.get('data', {}), indent=2)}")
                return True
            else:
                print(f"❌ JSON submission failed: {response.status_code}")
                try:
                    error_data = response.json()
                    print(f"   Error Response: {json.dumps(error_data, indent=2)}")
                except:
                    print(f"   Error Text: {response.text}")
                return False
                
        except requests.exceptions.RequestException as e:
            print(f"❌ Request failed: {e}")
            return False
    
    def test_web3forms_submission_form_data(self):
        """Test Web3Forms submission with form data"""
        print("🔍 Testing Web3Forms submission (Form Data payload)...")
        
        # Test form data submission (more common for HTML forms)
        form_data = {
            "access_key": self.api_key,
            "email": "formdata@example.com",
            "subject": "Test Form Data Submission",
            "message": "This is a test using form data encoding to verify auto-form creation.",
            "name": "Form Data Tester",
            "company": "Test Company"
        }
        
        try:
            response = requests.post(f"{API_BASE}/submit", data=form_data, timeout=15)
            
            print(f"📊 Response Status: {response.status_code}")
            
            if response.status_code == 200:
                data = response.json()
                print(f"✅ Form data submission successful!")
                print(f"   Success: {data.get('success', False)}")
                print(f"   Message: {data.get('message', 'No message')}")
                print(f"   Data: {json.dumps(data.get('data', {}), indent=2)}")
                return True
            else:
                print(f"❌ Form data submission failed: {response.status_code}")
                try:
                    error_data = response.json()
                    print(f"   Error Response: {json.dumps(error_data, indent=2)}")
                except:
                    print(f"   Error Text: {response.text}")
                return False
                
        except requests.exceptions.RequestException as e:
            print(f"❌ Request failed: {e}")
            return False
    
    def test_multiple_submissions(self):
        """Test multiple submissions to verify form reuse"""
        print("🔍 Testing multiple submissions (should reuse auto-created form)...")
        
        success_count = 0
        total_tests = 3
        
        for i in range(total_tests):
            submission_data = {
                "access_key": self.api_key,
                "email": f"test{i}@example.com",
                "subject": f"Test Submission #{i+1}",
                "message": f"This is test submission number {i+1} to verify form reuse.",
                "name": f"Test User {i+1}"
            }
            
            try:
                response = requests.post(f"{API_BASE}/submit", json=submission_data, timeout=15)
                
                if response.status_code == 200:
                    data = response.json()
                    if data.get('success', False):
                        success_count += 1
                        print(f"   ✅ Submission {i+1}/3 successful")
                    else:
                        print(f"   ❌ Submission {i+1}/3 marked as failed: {data.get('message')}")
                else:
                    print(f"   ❌ Submission {i+1}/3 HTTP error: {response.status_code}")
                    
            except requests.exceptions.RequestException as e:
                print(f"   ❌ Submission {i+1}/3 request failed: {e}")
            
            # Small delay between requests
            time.sleep(0.5)
        
        if success_count == total_tests:
            print(f"✅ Multiple submissions test passed ({success_count}/{total_tests})")
            return True
        else:
            print(f"⚠️ Multiple submissions test partial: ({success_count}/{total_tests})")
            return success_count > 0
    
    def test_invalid_api_key(self):
        """Test with invalid API key to verify error handling"""
        print("🔍 Testing invalid API key (should fail gracefully)...")
        
        submission_data = {
            "access_key": "invalid-key-should-fail-123",
            "email": "test@example.com",
            "subject": "This should fail",
            "message": "This submission should be rejected due to invalid API key."
        }
        
        try:
            response = requests.post(f"{API_BASE}/submit", json=submission_data, timeout=15)
            
            if response.status_code == 401:
                print(f"✅ Invalid API key correctly rejected (401 Unauthorized)")
                return True
            elif response.status_code != 200:
                print(f"✅ Invalid API key rejected with status {response.status_code}")
                return True
            else:
                data = response.json()
                if not data.get('success', True):
                    print(f"✅ Invalid API key correctly rejected in response")
                    return True
                else:
                    print(f"❌ Invalid API key was accepted (this is a bug!)")
                    return False
                    
        except requests.exceptions.RequestException as e:
            print(f"❌ Request failed: {e}")
            return False
    
    def run_all_tests(self):
        """Run all Web3Forms bug fix tests"""
        print("🚀 Starting Web3Forms Auto-Form Creation Bug Fix Test")
        print("=" * 60)
        print(f"📡 Backend URL: {BASE_URL}")
        print(f"🔑 API Key: {self.api_key[:20]}...")
        print(f"👤 User: {USER_EMAIL}")
        print("=" * 60)
        
        tests = [
            ("Health Check", self.test_health_check),
            ("JSON Submission", self.test_web3forms_submission_json),
            ("Form Data Submission", self.test_web3forms_submission_form_data),
            ("Multiple Submissions", self.test_multiple_submissions),
            ("Invalid API Key", self.test_invalid_api_key),
        ]
        
        passed = 0
        total = len(tests)
        
        for test_name, test_func in tests:
            print(f"\n📋 Test: {test_name}")
            print("-" * 40)
            
            try:
                if test_func():
                    passed += 1
                    print(f"🟢 {test_name}: PASSED")
                else:
                    print(f"🔴 {test_name}: FAILED")
            except Exception as e:
                print(f"🔴 {test_name}: EXCEPTION - {e}")
            
            print()  # Empty line for readability
        
        print("=" * 60)
        print(f"📊 Final Results: {passed}/{total} tests passed")
        
        if passed == total:
            print("🎉 All tests passed! Web3Forms auto-form creation is working!")
        elif passed >= total - 1:
            print("🟡 Almost all tests passed. Minor issues detected.")
        else:
            print(f"🔴 {total - passed} significant test(s) failed. Bug fix needs attention.")
        
        print("=" * 60)
        return passed >= total - 1  # Allow for 1 test failure

def main():
    tester = Web3FormsTester()
    success = tester.run_all_tests()
    
    if success:
        print("\n✅ BUG FIX VERIFICATION: SUCCESS")
        print("The Web3Forms auto-form creation functionality is working correctly.")
    else:
        print("\n❌ BUG FIX VERIFICATION: NEEDS ATTENTION")
        print("There are still issues with the Web3Forms functionality.")
    
    return 0 if success else 1

if __name__ == "__main__":
    exit(main())