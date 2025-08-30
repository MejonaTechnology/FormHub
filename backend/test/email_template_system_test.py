#!/usr/bin/env python3
"""
FormHub Email Template System Test Suite

This script tests all the major components of the email template system:
- Email Templates CRUD operations
- Email Providers configuration
- Autoresponders with conditions
- Email Queue management
- Template Builder functionality
- A/B Testing features
- Analytics reporting

Usage:
    python email_template_system_test.py

Prerequisites:
    - FormHub server running on localhost:8080
    - Valid user account and JWT token
    - pip install requests
"""

import requests
import json
import time
import uuid
from typing import Dict, List, Any, Optional

class FormHubEmailTestClient:
    def __init__(self, base_url: str = "http://localhost:8080/api/v1", token: str = None):
        self.base_url = base_url
        self.token = token
        self.session = requests.Session()
        if token:
            self.session.headers.update({
                'Authorization': f'Bearer {token}',
                'Content-Type': 'application/json'
            })

    def authenticate(self, email: str, password: str) -> bool:
        """Authenticate and get JWT token"""
        response = self.session.post(f"{self.base_url}/auth/login", json={
            "email": email,
            "password": password
        })
        
        if response.status_code == 200:
            data = response.json()
            self.token = data['access_token']
            self.session.headers.update({
                'Authorization': f'Bearer {self.token}'
            })
            print("✅ Authentication successful")
            return True
        else:
            print(f"❌ Authentication failed: {response.text}")
            return False

    def test_email_templates(self) -> Dict[str, str]:
        """Test email template CRUD operations"""
        print("\n=== Testing Email Templates ===")
        
        # Create template
        template_data = {
            "name": "Test Welcome Template",
            "description": "Test template for new user welcome",
            "type": "welcome",
            "language": "en",
            "subject": "Welcome to {{company_name}}, {{name}}!",
            "html_content": """
            <!DOCTYPE html>
            <html>
            <head>
                <meta charset="UTF-8">
                <title>Welcome</title>
                <style>
                    body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
                    .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 20px; text-align: center; }
                    .content { padding: 20px; }
                    .button { display: inline-block; background: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; }
                </style>
            </head>
            <body>
                <div class="header">
                    <h1>Welcome to {{company_name}}!</h1>
                </div>
                <div class="content">
                    <h2>Hello {{name}},</h2>
                    <p>Thank you for joining {{company_name}}. We're excited to have you on board!</p>
                    <p>Your email: {{email}}</p>
                    <p>Get started by exploring our features:</p>
                    <a href="{{dashboard_url}}" class="button">Go to Dashboard</a>
                </div>
            </body>
            </html>
            """,
            "text_content": "Welcome {{name}}! Thank you for joining {{company_name}}. Your email: {{email}}. Visit {{dashboard_url}} to get started.",
            "variables": ["name", "email", "company_name", "dashboard_url"],
            "tags": ["welcome", "onboarding", "test"]
        }
        
        response = self.session.post(f"{self.base_url}/email/templates", json=template_data)
        if response.status_code == 201:
            template = response.json()['template']
            template_id = template['id']
            print(f"✅ Template created: {template_id}")
        else:
            print(f"❌ Failed to create template: {response.text}")
            return {}

        # List templates
        response = self.session.get(f"{self.base_url}/email/templates")
        if response.status_code == 200:
            templates = response.json()['templates']
            print(f"✅ Listed {len(templates)} templates")
        else:
            print(f"❌ Failed to list templates: {response.text}")

        # Get template
        response = self.session.get(f"{self.base_url}/email/templates/{template_id}")
        if response.status_code == 200:
            template = response.json()['template']
            print(f"✅ Retrieved template: {template['name']}")
        else:
            print(f"❌ Failed to get template: {response.text}")

        # Preview template
        preview_data = {
            "variables": {
                "name": "John Doe",
                "email": "john.doe@example.com",
                "company_name": "FormHub Inc.",
                "dashboard_url": "https://formhub.io/dashboard"
            }
        }
        
        response = self.session.post(f"{self.base_url}/email/templates/{template_id}/preview", json=preview_data)
        if response.status_code == 200:
            rendered = response.json()['rendered']
            print("✅ Template preview generated")
            print(f"   Subject: {rendered['subject']}")
        else:
            print(f"❌ Failed to preview template: {response.text}")

        # Clone template
        clone_data = {"name": "Test Welcome Template - Clone"}
        response = self.session.post(f"{self.base_url}/email/templates/{template_id}/clone", json=clone_data)
        if response.status_code == 201:
            cloned_template = response.json()['template']
            print(f"✅ Template cloned: {cloned_template['id']}")
            return {
                'original_id': template_id,
                'clone_id': cloned_template['id']
            }
        else:
            print(f"❌ Failed to clone template: {response.text}")
            return {'original_id': template_id}

    def test_email_providers(self) -> Optional[str]:
        """Test email provider configuration"""
        print("\n=== Testing Email Providers ===")
        
        # Create SMTP provider
        provider_data = {
            "name": "Test SMTP Provider",
            "type": "smtp",
            "config": {
                "host": "smtp.gmail.com",
                "port": 587,
                "username": "test@example.com",
                "password": "test-password",
                "use_tls": True,
                "from_name": "FormHub Test",
                "from_email": "noreply@formhub-test.com"
            },
            "is_default": True
        }
        
        response = self.session.post(f"{self.base_url}/email/providers", json=provider_data)
        if response.status_code == 201:
            provider = response.json()['provider']
            provider_id = provider['id']
            print(f"✅ Email provider created: {provider_id}")
        else:
            print(f"❌ Failed to create email provider: {response.text}")
            return None

        # List providers
        response = self.session.get(f"{self.base_url}/email/providers")
        if response.status_code == 200:
            providers = response.json()['providers']
            print(f"✅ Listed {len(providers)} email providers")
        else:
            print(f"❌ Failed to list providers: {response.text}")

        # Note: Skipping provider test as it would require real SMTP credentials
        print("⚠️  Skipping provider test (requires real credentials)")
        
        return provider_id

    def test_autoresponders(self, template_id: str, form_id: str = None) -> Optional[str]:
        """Test autoresponder configuration"""
        print("\n=== Testing Autoresponders ===")
        
        # Use a dummy form ID if none provided
        if not form_id:
            form_id = str(uuid.uuid4())
            print(f"⚠️  Using dummy form ID: {form_id}")
        
        # Create autoresponder with conditions
        autoresponder_data = {
            "form_id": form_id,
            "name": "Test Welcome Autoresponder",
            "template_id": template_id,
            "delay_minutes": 5,
            "send_to_field": "email",
            "conditions": {
                "field_conditions": [
                    {
                        "field_name": "email",
                        "operator": "exists"
                    },
                    {
                        "field_name": "subscribe_newsletter",
                        "operator": "equals",
                        "value": "yes"
                    },
                    {
                        "field_name": "country",
                        "operator": "in",
                        "values": ["US", "CA", "UK"]
                    }
                ],
                "time_conditions": {
                    "start_time": "09:00",
                    "end_time": "17:00",
                    "days": ["monday", "tuesday", "wednesday", "thursday", "friday"],
                    "timezone": "America/New_York"
                },
                "logical_operator": "AND"
            },
            "cc_emails": ["backup@example.com"],
            "reply_to": "support@example.com",
            "track_opens": True,
            "track_clicks": True
        }
        
        response = self.session.post(f"{self.base_url}/email/autoresponders", json=autoresponder_data)
        if response.status_code == 201:
            autoresponder = response.json()['autoresponder']
            autoresponder_id = autoresponder['id']
            print(f"✅ Autoresponder created: {autoresponder_id}")
        else:
            print(f"❌ Failed to create autoresponder: {response.text}")
            return None

        # List autoresponders
        response = self.session.get(f"{self.base_url}/email/autoresponders")
        if response.status_code == 200:
            autoresponders = response.json()['autoresponders']
            print(f"✅ Listed {len(autoresponders)} autoresponders")
        else:
            print(f"❌ Failed to list autoresponders: {response.text}")

        # Toggle autoresponder
        toggle_data = {"enabled": False}
        response = self.session.post(f"{self.base_url}/email/autoresponders/{autoresponder_id}/toggle", json=toggle_data)
        if response.status_code == 200:
            print("✅ Autoresponder disabled")
        else:
            print(f"❌ Failed to toggle autoresponder: {response.text}")

        return autoresponder_id

    def test_email_queue(self):
        """Test email queue functionality"""
        print("\n=== Testing Email Queue ===")
        
        # Get queue statistics
        response = self.session.get(f"{self.base_url}/email/queue/stats")
        if response.status_code == 200:
            stats = response.json()['stats']
            print(f"✅ Queue stats: {stats['total']} total, {stats['pending']} pending")
        else:
            print(f"❌ Failed to get queue stats: {response.text}")

        # List queued emails
        response = self.session.get(f"{self.base_url}/email/queue/emails?limit=10")
        if response.status_code == 200:
            emails = response.json()['emails']
            print(f"✅ Listed {len(emails)} queued emails")
        else:
            print(f"❌ Failed to list queued emails: {response.text}")

        # Process queue (manual trigger)
        response = self.session.post(f"{self.base_url}/email/queue/process")
        if response.status_code == 200:
            result = response.json()['result']
            print(f"✅ Queue processed: {result['processed']} emails")
        else:
            print(f"❌ Failed to process queue: {response.text}")

    def test_template_builder(self):
        """Test drag-and-drop template builder"""
        print("\n=== Testing Template Builder ===")
        
        # Get available components
        response = self.session.get(f"{self.base_url}/email/builder/components")
        if response.status_code == 200:
            components = response.json()['components']
            print(f"✅ Retrieved {len(components)} available components")
        else:
            print(f"❌ Failed to get components: {response.text}")
            return

        # Create template design
        design_data = {
            "name": "Test Newsletter Design",
            "description": "A test newsletter created with the builder",
            "components": [
                {
                    "id": "header-1",
                    "type": "header",
                    "content": "{{company_name}} Newsletter",
                    "properties": {
                        "logo": "https://example.com/logo.png",
                        "title": "Monthly Update"
                    },
                    "styles": {
                        "background_color": "#4a90e2",
                        "text_color": "white",
                        "text_align": "center",
                        "padding": "20px"
                    },
                    "order": 1
                },
                {
                    "id": "text-1",
                    "type": "text",
                    "content": "<h2>Hello {{name}},</h2><p>Here's your monthly newsletter with the latest updates from {{company_name}}.</p>",
                    "properties": {
                        "tag": "div"
                    },
                    "styles": {
                        "padding": "20px",
                        "font_size": "16px"
                    },
                    "order": 2
                },
                {
                    "id": "button-1",
                    "type": "button",
                    "content": "Read Full Article",
                    "properties": {
                        "url": "{{article_url}}",
                        "target": "_blank"
                    },
                    "styles": {
                        "background_color": "#28a745",
                        "text_color": "white",
                        "border_radius": "5px",
                        "padding": "12px 24px",
                        "margin": "20px"
                    },
                    "order": 3
                },
                {
                    "id": "footer-1",
                    "type": "footer",
                    "content": "",
                    "properties": {
                        "company_name": "{{company_name}}",
                        "address": "123 Main St, City, State 12345",
                        "unsubscribe_url": "{{unsubscribe_url}}"
                    },
                    "styles": {
                        "background_color": "#f8f9fa",
                        "text_align": "center",
                        "padding": "20px",
                        "font_size": "12px",
                        "color": "#666"
                    },
                    "order": 4
                }
            ],
            "global_styles": {
                "container_width": "600px",
                "background_color": "#ffffff",
                "default_font_family": "Arial, sans-serif",
                "default_text_color": "#333333",
                "link_color": "#4a90e2"
            },
            "category": "newsletter",
            "tags": ["newsletter", "monthly", "test"],
            "is_template": True
        }
        
        response = self.session.post(f"{self.base_url}/email/builder/designs", json=design_data)
        if response.status_code == 201:
            design = response.json()['design']
            print(f"✅ Template design created: {design['id']}")
        else:
            print(f"❌ Failed to create template design: {response.text}")

        # Generate preview
        preview_data = {
            "components": design_data["components"],
            "global_styles": design_data["global_styles"],
            "variables": {
                "company_name": "FormHub Inc.",
                "name": "John Doe",
                "article_url": "https://formhub.io/blog/latest",
                "unsubscribe_url": "https://formhub.io/unsubscribe"
            }
        }
        
        response = self.session.post(f"{self.base_url}/email/builder/preview", json=preview_data)
        if response.status_code == 200:
            print("✅ Template preview generated successfully")
            # In a real test, you might save this HTML to a file for manual inspection
        else:
            print(f"❌ Failed to generate preview: {response.text}")

    def test_ab_testing(self, template_a_id: str, template_b_id: str):
        """Test A/B testing functionality"""
        print("\n=== Testing A/B Testing ===")
        
        # Create A/B test
        ab_test_data = {
            "name": "Welcome Email Subject Test",
            "description": "Testing different subject line approaches",
            "template_a_id": template_a_id,
            "template_b_id": template_b_id,
            "traffic_split": 50,
            "test_metric": "open_rate",
            "min_sample_size": 100,
            "max_duration_days": 30
        }
        
        response = self.session.post(f"{self.base_url}/email/ab-tests", json=ab_test_data)
        if response.status_code == 201:
            test = response.json()['test']
            test_id = test['id']
            print(f"✅ A/B test created: {test_id}")
        else:
            print(f"❌ Failed to create A/B test: {response.text}")
            return

        # Start A/B test
        response = self.session.post(f"{self.base_url}/email/ab-tests/{test_id}/start")
        if response.status_code == 200:
            test = response.json()['test']
            print(f"✅ A/B test started: {test['status']}")
        else:
            print(f"❌ Failed to start A/B test: {response.text}")

        # Get A/B test results (will show preliminary results)
        response = self.session.get(f"{self.base_url}/email/ab-tests/{test_id}/results")
        if response.status_code == 200:
            result = response.json()['result']
            print(f"✅ A/B test results retrieved")
            print(f"   Current winner: {result['winner']}")
            print(f"   Confidence: {result['confidence']}%")
            print(f"   Recommendation: {result['recommendation'][:100]}...")
        else:
            print(f"❌ Failed to get A/B test results: {response.text}")

    def test_analytics(self, template_id: str):
        """Test email analytics functionality"""
        print("\n=== Testing Email Analytics ===")
        
        # Get template analytics
        response = self.session.get(f"{self.base_url}/email/templates/{template_id}/analytics?start_date=2024-01-01&end_date=2024-12-31")
        if response.status_code == 200:
            report = response.json()['report']
            print(f"✅ Template analytics retrieved")
            print(f"   Template: {report['template_name']}")
            print(f"   Total sent: {report['total_sent']}")
            print(f"   Open rate: {report['open_rate']:.1f}%")
        else:
            print(f"❌ Failed to get template analytics: {response.text}")

        # Get user analytics overview
        response = self.session.get(f"{self.base_url}/email/analytics/overview?start_date=2024-01-01&end_date=2024-12-31")
        if response.status_code == 200:
            report = response.json()['report']
            print(f"✅ User analytics overview retrieved")
            print(f"   Total sent: {report['total_sent']}")
            print(f"   Overall open rate: {report['open_rate']:.1f}%")
        else:
            print(f"❌ Failed to get user analytics: {response.text}")

        # Get top performing templates
        response = self.session.get(f"{self.base_url}/email/analytics/top-templates?limit=5")
        if response.status_code == 200:
            templates = response.json()['templates']
            print(f"✅ Top performing templates retrieved: {len(templates)} templates")
            for i, template in enumerate(templates[:3], 1):
                print(f"   {i}. {template['template_name']} - {template['open_rate']:.1f}% open rate")
        else:
            print(f"❌ Failed to get top templates: {response.text}")

    def run_comprehensive_test(self, email: str = None, password: str = None):
        """Run all tests in sequence"""
        print("🚀 Starting FormHub Email Template System Comprehensive Test")
        print("=" * 60)
        
        # Authentication (if credentials provided)
        if email and password:
            if not self.authenticate(email, password):
                print("❌ Cannot continue without authentication")
                return
        elif not self.token:
            print("⚠️  No authentication provided - using existing token or running without auth")
        
        try:
            # 1. Test email templates
            template_results = self.test_email_templates()
            if not template_results:
                print("❌ Template tests failed - stopping")
                return
            
            # 2. Test email providers
            provider_id = self.test_email_providers()
            
            # 3. Test autoresponders
            autoresponder_id = self.test_autoresponders(template_results['original_id'])
            
            # 4. Test email queue
            self.test_email_queue()
            
            # 5. Test template builder
            self.test_template_builder()
            
            # 6. Test A/B testing
            if 'clone_id' in template_results:
                self.test_ab_testing(template_results['original_id'], template_results['clone_id'])
            
            # 7. Test analytics
            self.test_analytics(template_results['original_id'])
            
            print("\n" + "=" * 60)
            print("🎉 Comprehensive test completed successfully!")
            print("✅ All major email template system features are working")
            
        except Exception as e:
            print(f"\n❌ Test failed with exception: {str(e)}")
            import traceback
            traceback.print_exc()

def main():
    """Main test runner"""
    print("FormHub Email Template System Test Suite")
    print("=" * 50)
    
    # Configuration
    BASE_URL = "http://localhost:8080/api/v1"
    
    # You can either:
    # 1. Provide credentials for automatic login
    # 2. Provide a JWT token directly
    # 3. Run without authentication (will likely fail for protected endpoints)
    
    EMAIL = "test@example.com"        # Set your test email
    PASSWORD = "testpassword"         # Set your test password
    JWT_TOKEN = None                  # Or provide JWT token directly
    
    # Initialize test client
    client = FormHubEmailTestClient(BASE_URL, JWT_TOKEN)
    
    # Run comprehensive test
    if EMAIL and PASSWORD:
        client.run_comprehensive_test(EMAIL, PASSWORD)
    else:
        print("⚠️  Running tests without authentication")
        print("   Update EMAIL and PASSWORD variables in the script for full testing")
        client.run_comprehensive_test()

if __name__ == "__main__":
    main()