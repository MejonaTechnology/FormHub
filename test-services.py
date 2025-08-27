#!/usr/bin/env python3
"""
FormHub Services Testing Script
Tests both backend and frontend availability
"""
import requests
import time
import json

def test_backend():
    """Test backend API endpoints"""
    print("Testing Backend Services...")
    
    # Test health endpoint
    try:
        response = requests.get("http://13.127.59.135:9000/health", timeout=10)
        print(f"Backend Health: {response.status_code} - {response.json()}")
    except Exception as e:
        print(f"Backend Health Failed: {e}")
        return False
    
    # Test Web3Forms API
    try:
        data = {
            "access_key": "b73fb689-7901-4122-8cb0-3dfcc2498235-4621361e-a4a1-448d-bf81-8a2302d452ea",
            "email": "test@example.com",
            "subject": "E2E Test Submission",
            "message": "Testing Web3Forms functionality after deployment"
        }
        response = requests.post("http://13.127.59.135:9000/api/v1/submit", 
                               json=data, timeout=10)
        print(f"Web3Forms API: {response.status_code} - {response.json()}")
        return response.status_code == 200
    except Exception as e:
        print(f"Web3Forms API Failed: {e}")
        return False

def test_frontend():
    """Test frontend pages"""
    print("\nTesting Frontend Services...")
    
    pages = [
        "/",
        "/dashboard",
        "/auth/login", 
        "/auth/register",
        "/pricing",
        "/docs"
    ]
    
    frontend_working = False
    for page in pages:
        try:
            url = f"http://13.127.59.135:3000{page}"
            response = requests.get(url, timeout=10)
            status = "OK" if response.status_code in [200, 307] else "FAIL"
            print(f"{status} {page}: {response.status_code}")
            if response.status_code in [200, 307]:
                frontend_working = True
        except Exception as e:
            print(f"FAIL {page}: Connection failed - {e}")
    
    return frontend_working

def main():
    print("FormHub End-to-End Testing")
    print("=" * 50)
    
    backend_ok = test_backend()
    frontend_ok = test_frontend()
    
    print(f"\nTest Results Summary:")
    print(f"Backend API: {'WORKING' if backend_ok else 'FAILED'}")
    print(f"Frontend UI: {'WORKING' if frontend_ok else 'FAILED'}")
    
    if backend_ok and frontend_ok:
        print(f"\nFormHub System: FULLY OPERATIONAL")
        return True
    else:
        print(f"\nFormHub System: PARTIALLY FUNCTIONAL")
        return False

if __name__ == "__main__":
    success = main()
    exit(0 if success else 1)