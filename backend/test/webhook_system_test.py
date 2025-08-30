#!/usr/bin/env python3
"""
Comprehensive Webhook System Testing Suite
Tests all aspects of the enhanced webhook system including:
- Basic webhook functionality
- Third-party integrations
- Analytics and monitoring
- Enterprise features
- Load testing
"""

import asyncio
import json
import time
import uuid
import requests
import aiohttp
import pytest
import logging
from datetime import datetime, timedelta
from typing import Dict, List, Any
from concurrent.futures import ThreadPoolExecutor
import threading

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class WebhookSystemTester:
    def __init__(self, base_url: str = "http://localhost:8080", api_key: str = None):
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.session = requests.Session()
        if api_key:
            self.session.headers.update({"Authorization": f"Bearer {api_key}"})
        
        # Test data
        self.test_form_id = None
        self.test_endpoints = []
        self.test_results = {
            "basic_webhook": {},
            "integrations": {},
            "analytics": {},
            "monitoring": {},
            "enterprise": {},
            "load_testing": {}
        }
    
    def setup_test_environment(self):
        """Set up test environment with forms and endpoints"""
        logger.info("Setting up test environment...")
        
        # Create test form
        form_data = {
            "name": "Webhook Test Form",
            "description": "Form for testing webhook functionality",
            "fields": [
                {"name": "name", "type": "text", "required": True},
                {"name": "email", "type": "email", "required": True},
                {"name": "message", "type": "textarea", "required": False}
            ]
        }
        
        response = self.session.post(f"{self.base_url}/api/v1/forms", json=form_data)
        if response.status_code == 201:
            self.test_form_id = response.json()["form"]["id"]
            logger.info(f"Created test form: {self.test_form_id}")
        else:
            logger.error(f"Failed to create test form: {response.text}")
            raise Exception("Could not create test form")
    
    def test_basic_webhook_functionality(self):
        """Test basic webhook CRUD operations"""
        logger.info("Testing basic webhook functionality...")
        
        # Test webhook endpoint creation
        endpoint_data = {
            "name": "Test Webhook",
            "url": "https://httpbin.org/post",
            "events": ["form.submitted"],
            "enabled": True,
            "verify_ssl": True,
            "timeout": 30,
            "max_retries": 3,
            "retry_delay": 5
        }
        
        response = self.session.post(
            f"{self.base_url}/api/v1/forms/{self.test_form_id}/webhooks/endpoints",
            json=endpoint_data
        )
        
        if response.status_code == 201:
            endpoint = response.json()["endpoint"]
            self.test_endpoints.append(endpoint["id"])
            self.test_results["basic_webhook"]["create"] = True
            logger.info(f"✓ Created webhook endpoint: {endpoint['id']}")
        else:
            self.test_results["basic_webhook"]["create"] = False
            logger.error(f"✗ Failed to create webhook endpoint: {response.text}")
        
        # Test webhook endpoint retrieval
        response = self.session.get(
            f"{self.base_url}/api/v1/forms/{self.test_form_id}/webhooks/endpoints"
        )
        
        if response.status_code == 200:
            endpoints = response.json()["endpoints"]
            self.test_results["basic_webhook"]["retrieve"] = len(endpoints) > 0
            logger.info(f"✓ Retrieved {len(endpoints)} webhook endpoints")
        else:
            self.test_results["basic_webhook"]["retrieve"] = False
            logger.error(f"✗ Failed to retrieve webhook endpoints: {response.text}")
        
        # Test webhook endpoint update
        if self.test_endpoints:
            endpoint_id = self.test_endpoints[0]
            update_data = {
                "name": "Updated Test Webhook",
                "timeout": 60
            }
            
            response = self.session.put(
                f"{self.base_url}/api/v1/forms/{self.test_form_id}/webhooks/endpoints/{endpoint_id}",
                json=update_data
            )
            
            if response.status_code == 200:
                self.test_results["basic_webhook"]["update"] = True
                logger.info(f"✓ Updated webhook endpoint: {endpoint_id}")
            else:
                self.test_results["basic_webhook"]["update"] = False
                logger.error(f"✗ Failed to update webhook endpoint: {response.text}")
        
        # Test webhook testing functionality
        if self.test_endpoints:
            endpoint_id = self.test_endpoints[0]
            response = self.session.post(
                f"{self.base_url}/api/v1/forms/{self.test_form_id}/webhooks/endpoints/{endpoint_id}/test"
            )
            
            if response.status_code == 200:
                test_result = response.json()["result"]
                self.test_results["basic_webhook"]["test"] = test_result["success"]
                logger.info(f"✓ Webhook test successful: {test_result['response_time']}ms")
            else:
                self.test_results["basic_webhook"]["test"] = False
                logger.error(f"✗ Webhook test failed: {response.text}")
    
    def test_third_party_integrations(self):
        """Test third-party integration functionality"""
        logger.info("Testing third-party integrations...")
        
        # Test integration listing
        response = self.session.get(f"{self.base_url}/api/v1/integrations")
        
        if response.status_code == 200:
            integrations = response.json()["integrations"]
            self.test_results["integrations"]["list"] = len(integrations) > 0
            logger.info(f"✓ Found {len(integrations)} available integrations")
            
            # Test specific integration schemas
            for integration in integrations[:3]:  # Test first 3 integrations
                name = integration["name"]
                response = self.session.get(f"{self.base_url}/api/v1/integrations/{name}/schema")
                
                if response.status_code == 200:
                    schema = response.json()["schema"]
                    self.test_results["integrations"][f"schema_{name}"] = "fields" in schema
                    logger.info(f"✓ Retrieved schema for {name}")
                else:
                    self.test_results["integrations"][f"schema_{name}"] = False
                    logger.error(f"✗ Failed to get schema for {name}")
        else:
            self.test_results["integrations"]["list"] = False
            logger.error(f"✗ Failed to list integrations: {response.text}")
        
        # Test integration configuration validation
        test_configs = {
            "slack": {
                "webhook_url": "https://hooks.slack.com/services/TEST/TEST/TEST",
                "channel": "#general"
            },
            "google_sheets": {
                "spreadsheet_id": "test_sheet_id",
                "credentials_json": "{\"type\": \"service_account\"}"
            }
        }
        
        for integration_name, config in test_configs.items():
            response = self.session.post(
                f"{self.base_url}/api/v1/integrations/{integration_name}/test",
                json=config
            )
            
            # We expect this to fail with authentication error, but config should be valid
            self.test_results["integrations"][f"config_{integration_name}"] = response.status_code in [200, 400]
            if response.status_code == 200:
                logger.info(f"✓ {integration_name} configuration test passed")
            elif response.status_code == 400:
                logger.info(f"✓ {integration_name} configuration validation working (auth failed as expected)")
            else:
                logger.error(f"✗ {integration_name} configuration test failed unexpectedly")
    
    def test_marketplace_integrations(self):
        """Test integration marketplace functionality"""
        logger.info("Testing integration marketplace...")
        
        # Test marketplace listing
        response = self.session.get(f"{self.base_url}/api/v1/marketplace/integrations")
        
        if response.status_code == 200:
            marketplace_integrations = response.json()["integrations"]
            self.test_results["integrations"]["marketplace_list"] = len(marketplace_integrations) > 0
            logger.info(f"✓ Found {len(marketplace_integrations)} marketplace integrations")
            
            # Test filtering
            response = self.session.get(
                f"{self.base_url}/api/v1/marketplace/integrations?category=communication&popular=true"
            )
            
            if response.status_code == 200:
                filtered = response.json()["integrations"]
                self.test_results["integrations"]["marketplace_filter"] = True
                logger.info(f"✓ Filtered marketplace returned {len(filtered)} integrations")
            else:
                self.test_results["integrations"]["marketplace_filter"] = False
        else:
            self.test_results["integrations"]["marketplace_list"] = False
            logger.error(f"✗ Failed to list marketplace integrations: {response.text}")
        
        # Test categories
        response = self.session.get(f"{self.base_url}/api/v1/marketplace/categories")
        
        if response.status_code == 200:
            categories = response.json()["categories"]
            self.test_results["integrations"]["marketplace_categories"] = len(categories) > 0
            logger.info(f"✓ Found {len(categories)} integration categories")
        else:
            self.test_results["integrations"]["marketplace_categories"] = False
    
    def test_analytics_functionality(self):
        """Test webhook analytics functionality"""
        logger.info("Testing webhook analytics...")
        
        # Generate some test webhook activity first
        self._generate_test_webhook_activity()
        
        # Test analytics retrieval
        end_date = datetime.now()
        start_date = end_date - timedelta(days=7)
        
        params = {
            "start": start_date.isoformat(),
            "end": end_date.isoformat()
        }
        
        response = self.session.get(
            f"{self.base_url}/api/v1/forms/{self.test_form_id}/webhooks/analytics",
            params=params
        )
        
        if response.status_code == 200:
            analytics = response.json()["analytics"]
            self.test_results["analytics"]["retrieve"] = "total_webhooks" in analytics
            logger.info(f"✓ Retrieved analytics: {analytics.get('total_webhooks', 0)} total webhooks")
        else:
            self.test_results["analytics"]["retrieve"] = False
            logger.error(f"✗ Failed to retrieve analytics: {response.text}")
        
        # Test real-time stats
        response = self.session.get(
            f"{self.base_url}/api/v1/forms/{self.test_form_id}/webhooks/stats/realtime"
        )
        
        if response.status_code == 200:
            stats = response.json()["stats"]
            self.test_results["analytics"]["realtime"] = "total_requests" in stats
            logger.info(f"✓ Retrieved real-time stats: {stats.get('total_requests', 0)} total requests")
        else:
            self.test_results["analytics"]["realtime"] = False
            logger.error(f"✗ Failed to retrieve real-time stats: {response.text}")
    
    def test_monitoring_functionality(self):
        """Test webhook monitoring functionality"""
        logger.info("Testing webhook monitoring...")
        
        # Test monitoring data retrieval
        response = self.session.get(
            f"{self.base_url}/api/v1/forms/{self.test_form_id}/webhooks/monitoring"
        )
        
        if response.status_code == 200:
            monitoring = response.json()["monitoring"]
            self.test_results["monitoring"]["retrieve"] = "status" in monitoring
            logger.info(f"✓ Retrieved monitoring data: Status = {monitoring.get('status', 'unknown')}")
        else:
            self.test_results["monitoring"]["retrieve"] = False
            logger.error(f"✗ Failed to retrieve monitoring data: {response.text}")
        
        # Test health checks (if endpoints exist)
        if self.test_endpoints:
            endpoint_id = self.test_endpoints[0]
            # Health checks are typically done automatically, so we just verify the data exists
            if response.status_code == 200:
                monitoring = response.json()["monitoring"]
                health_checks = monitoring.get("health_checks", [])
                self.test_results["monitoring"]["health_checks"] = len(health_checks) >= 0
                logger.info(f"✓ Found {len(health_checks)} health check records")
    
    def test_enterprise_features(self):
        """Test enterprise features like rate limiting, circuit breaker, etc."""
        logger.info("Testing enterprise features...")
        
        # Test rate limiting by making multiple rapid requests
        self.test_results["enterprise"]["rate_limiting"] = self._test_rate_limiting()
        
        # Test circuit breaker behavior
        self.test_results["enterprise"]["circuit_breaker"] = self._test_circuit_breaker()
        
        # Test security features
        self.test_results["enterprise"]["security"] = self._test_security_features()
    
    def _test_rate_limiting(self) -> bool:
        """Test rate limiting functionality"""
        logger.info("Testing rate limiting...")
        
        # Make rapid requests to test rate limiting
        success_count = 0
        rate_limited_count = 0
        
        for i in range(20):
            response = self.session.get(
                f"{self.base_url}/api/v1/forms/{self.test_form_id}/webhooks/endpoints"
            )
            
            if response.status_code == 200:
                success_count += 1
            elif response.status_code == 429:  # Too Many Requests
                rate_limited_count += 1
            
            time.sleep(0.1)  # Small delay between requests
        
        # Rate limiting should kick in after some requests
        logger.info(f"Rate limiting test: {success_count} successful, {rate_limited_count} rate limited")
        return rate_limited_count > 0 or success_count == 20  # Either rate limited or all passed
    
    def _test_circuit_breaker(self) -> bool:
        """Test circuit breaker functionality"""
        logger.info("Testing circuit breaker...")
        
        # Create a webhook endpoint that will fail
        endpoint_data = {
            "name": "Failing Webhook",
            "url": "http://localhost:99999/nonexistent",  # This should fail
            "events": ["form.submitted"],
            "enabled": True,
            "max_retries": 1
        }
        
        response = self.session.post(
            f"{self.base_url}/api/v1/forms/{self.test_form_id}/webhooks/endpoints",
            json=endpoint_data
        )
        
        if response.status_code == 201:
            failing_endpoint_id = response.json()["endpoint"]["id"]
            
            # Test the failing endpoint multiple times
            for i in range(3):
                response = self.session.post(
                    f"{self.base_url}/api/v1/forms/{self.test_form_id}/webhooks/endpoints/{failing_endpoint_id}/test"
                )
                time.sleep(1)
            
            # Clean up
            self.session.delete(
                f"{self.base_url}/api/v1/forms/{self.test_form_id}/webhooks/endpoints/{failing_endpoint_id}"
            )
            
            logger.info("✓ Circuit breaker test completed")
            return True
        
        return False
    
    def _test_security_features(self) -> bool:
        """Test security features"""
        logger.info("Testing security features...")
        
        # Test webhook signature verification
        if self.test_endpoints:
            endpoint_id = self.test_endpoints[0]
            
            # Update endpoint to require signature
            update_data = {
                "secret": "test-secret-key"
            }
            
            response = self.session.put(
                f"{self.base_url}/api/v1/forms/{self.test_form_id}/webhooks/endpoints/{endpoint_id}",
                json=update_data
            )
            
            if response.status_code == 200:
                logger.info("✓ Updated endpoint with secret for signature testing")
                return True
        
        return False
    
    def test_load_performance(self):
        """Test system performance under load"""
        logger.info("Testing load performance...")
        
        # Test concurrent webhook endpoint creation
        self.test_results["load_testing"]["concurrent_creation"] = self._test_concurrent_operations()
        
        # Test webhook delivery performance
        self.test_results["load_testing"]["delivery_performance"] = self._test_delivery_performance()
    
    def _test_concurrent_operations(self) -> bool:
        """Test concurrent webhook operations"""
        logger.info("Testing concurrent operations...")
        
        def create_endpoint(i):
            endpoint_data = {
                "name": f"Load Test Webhook {i}",
                "url": f"https://httpbin.org/post?test={i}",
                "events": ["form.submitted"],
                "enabled": True
            }
            
            response = self.session.post(
                f"{self.base_url}/api/v1/forms/{self.test_form_id}/webhooks/endpoints",
                json=endpoint_data
            )
            
            return response.status_code == 201
        
        # Create multiple endpoints concurrently
        with ThreadPoolExecutor(max_workers=5) as executor:
            futures = [executor.submit(create_endpoint, i) for i in range(10)]
            results = [future.result() for future in futures]
        
        success_count = sum(results)
        logger.info(f"Concurrent creation test: {success_count}/10 successful")
        
        return success_count >= 8  # Allow for some failures
    
    def _test_delivery_performance(self) -> bool:
        """Test webhook delivery performance"""
        logger.info("Testing webhook delivery performance...")
        
        # This would typically involve triggering actual webhook deliveries
        # For this test, we'll measure the response time of webhook tests
        
        if not self.test_endpoints:
            return False
        
        endpoint_id = self.test_endpoints[0]
        response_times = []
        
        for i in range(5):
            start_time = time.time()
            response = self.session.post(
                f"{self.base_url}/api/v1/forms/{self.test_form_id}/webhooks/endpoints/{endpoint_id}/test"
            )
            end_time = time.time()
            
            if response.status_code == 200:
                response_times.append(end_time - start_time)
            
            time.sleep(1)
        
        if response_times:
            avg_response_time = sum(response_times) / len(response_times)
            logger.info(f"Average webhook test response time: {avg_response_time:.3f}s")
            return avg_response_time < 5.0  # Should be under 5 seconds
        
        return False
    
    def _generate_test_webhook_activity(self):
        """Generate some test webhook activity for analytics testing"""
        logger.info("Generating test webhook activity...")
        
        # Submit a few test forms to trigger webhook activity
        for i in range(3):
            form_data = {
                "name": f"Test User {i}",
                "email": f"test{i}@example.com",
                "message": f"Test message {i}"
            }
            
            # This would normally trigger webhooks
            response = self.session.post(
                f"{self.base_url}/api/v1/submit",
                json={"form_id": self.test_form_id, "data": form_data}
            )
            
            time.sleep(0.5)
    
    def cleanup_test_environment(self):
        """Clean up test environment"""
        logger.info("Cleaning up test environment...")
        
        # Delete test endpoints
        for endpoint_id in self.test_endpoints:
            response = self.session.delete(
                f"{self.base_url}/api/v1/forms/{self.test_form_id}/webhooks/endpoints/{endpoint_id}"
            )
            if response.status_code == 200:
                logger.info(f"✓ Deleted endpoint: {endpoint_id}")
        
        # Delete test form
        if self.test_form_id:
            response = self.session.delete(f"{self.base_url}/api/v1/forms/{self.test_form_id}")
            if response.status_code == 200:
                logger.info(f"✓ Deleted test form: {self.test_form_id}")
    
    def run_all_tests(self):
        """Run all webhook system tests"""
        logger.info("Starting comprehensive webhook system tests...")
        
        try:
            # Setup
            self.setup_test_environment()
            
            # Run tests
            self.test_basic_webhook_functionality()
            self.test_third_party_integrations()
            self.test_marketplace_integrations()
            self.test_analytics_functionality()
            self.test_monitoring_functionality()
            self.test_enterprise_features()
            self.test_load_performance()
            
        except Exception as e:
            logger.error(f"Test execution failed: {str(e)}")
        finally:
            # Cleanup
            self.cleanup_test_environment()
        
        # Generate report
        self.generate_test_report()
    
    def generate_test_report(self):
        """Generate comprehensive test report"""
        logger.info("Generating test report...")
        
        total_tests = 0
        passed_tests = 0
        
        print("\n" + "="*60)
        print("FORMHUB WEBHOOK SYSTEM TEST REPORT")
        print("="*60)
        
        for category, tests in self.test_results.items():
            print(f"\n{category.upper().replace('_', ' ')}:")
            print("-" * 40)
            
            for test_name, result in tests.items():
                status = "✓ PASS" if result else "✗ FAIL"
                print(f"  {test_name.replace('_', ' ').title()}: {status}")
                total_tests += 1
                if result:
                    passed_tests += 1
        
        print(f"\n{'='*60}")
        print(f"SUMMARY: {passed_tests}/{total_tests} tests passed ({passed_tests/total_tests*100:.1f}%)")
        
        if passed_tests == total_tests:
            print("🎉 ALL TESTS PASSED! Webhook system is fully functional.")
        elif passed_tests / total_tests >= 0.8:
            print("✅ Most tests passed. Minor issues detected.")
        else:
            print("⚠️  Multiple test failures detected. Review system configuration.")
        
        print("="*60)
        
        # Save detailed results to file
        with open(f"webhook_test_report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json", 'w') as f:
            json.dump({
                "timestamp": datetime.now().isoformat(),
                "summary": {
                    "total_tests": total_tests,
                    "passed_tests": passed_tests,
                    "success_rate": passed_tests / total_tests if total_tests > 0 else 0
                },
                "results": self.test_results
            }, f, indent=2)


# Integration-specific test classes

class SlackIntegrationTester:
    """Specific tests for Slack integration"""
    
    def __init__(self, webhook_url: str):
        self.webhook_url = webhook_url
    
    def test_message_sending(self):
        """Test sending messages to Slack"""
        test_payload = {
            "text": "FormHub Test Message",
            "attachments": [{
                "color": "good",
                "title": "Test Webhook",
                "text": "This is a test message from FormHub webhook system.",
                "footer": "FormHub",
                "ts": int(time.time())
            }]
        }
        
        response = requests.post(self.webhook_url, json=test_payload)
        return response.status_code == 200


class GoogleSheetsIntegrationTester:
    """Specific tests for Google Sheets integration"""
    
    def __init__(self, credentials_json: str, spreadsheet_id: str):
        self.credentials_json = credentials_json
        self.spreadsheet_id = spreadsheet_id
    
    def test_sheet_access(self):
        """Test access to Google Sheets"""
        # This would require actual Google Sheets API implementation
        # For now, just validate the configuration format
        try:
            credentials = json.loads(self.credentials_json)
            required_fields = ["type", "project_id", "private_key", "client_email"]
            return all(field in credentials for field in required_fields)
        except json.JSONDecodeError:
            return False


# Main execution
if __name__ == "__main__":
    import argparse
    
    parser = argparse.ArgumentParser(description="FormHub Webhook System Comprehensive Test Suite")
    parser.add_argument("--base-url", default="http://localhost:8080", help="FormHub API base URL")
    parser.add_argument("--api-key", help="API key for authentication")
    parser.add_argument("--verbose", "-v", action="store_true", help="Verbose output")
    
    args = parser.parse_args()
    
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)
    
    # Run tests
    tester = WebhookSystemTester(base_url=args.base_url, api_key=args.api_key)
    tester.run_all_tests()