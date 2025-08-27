#!/usr/bin/env python3
"""
FormHub Services Restart Script
Attempts to restart both backend and frontend services via SSH
"""
import subprocess
import time
import sys

def run_ssh_command(command, description):
    """Execute SSH command with error handling"""
    full_cmd = [
        "ssh", 
        "-i", "D:/Mejona Workspace/Product/Mejona complete website/mejonaN.pem",
        "-o", "ConnectTimeout=15",
        "ec2-user@13.127.59.135",
        command
    ]
    
    print(f"Executing: {description}")
    try:
        result = subprocess.run(full_cmd, 
                              capture_output=True, 
                              text=True, 
                              timeout=60)
        print(f"Exit code: {result.returncode}")
        if result.stdout:
            print(f"Output: {result.stdout.strip()}")
        if result.stderr:
            print(f"Error: {result.stderr.strip()}")
        return result.returncode == 0
    except subprocess.TimeoutExpired:
        print("Command timed out")
        return False
    except Exception as e:
        print(f"Command failed: {e}")
        return False

def restart_backend():
    """Restart backend service"""
    print("\n=== Restarting Backend Service ===")
    
    commands = [
        ("sudo pkill -f formhub-api || true", "Kill existing backend process"),
        ("cd /opt/formhub && nohup ./formhub-api > formhub.log 2>&1 &", "Start backend"),
        ("sleep 5", "Wait for startup"),
        ("curl -s http://localhost:9000/health || echo 'Backend not responding'", "Test backend")
    ]
    
    for cmd, desc in commands:
        if not run_ssh_command(cmd, desc):
            print(f"Failed: {desc}")
    
    print("Backend restart completed")

def restart_frontend():
    """Restart frontend service"""
    print("\n=== Restarting Frontend Service ===")
    
    commands = [
        ("sudo systemctl stop formhub-frontend", "Stop frontend service"),
        ("sudo systemctl start formhub-frontend", "Start frontend service"),
        ("sleep 10", "Wait for startup"),
        ("curl -I http://localhost:3000/ || echo 'Frontend not responding'", "Test frontend")
    ]
    
    for cmd, desc in commands:
        if not run_ssh_command(cmd, desc):
            print(f"Warning: {desc} may have issues")
    
    print("Frontend restart completed")

def main():
    print("FormHub Services Restart")
    print("=" * 40)
    
    # Check SSH connectivity first
    if not run_ssh_command("echo 'SSH connection test'", "Test SSH connection"):
        print("SSH connection failed - cannot proceed")
        return False
    
    restart_backend()
    restart_frontend()
    
    print("\n=== Testing Services ===")
    time.sleep(5)
    
    # Final test
    backend_test = run_ssh_command("curl -s http://localhost:9000/health", "Final backend test")
    frontend_test = run_ssh_command("curl -I http://localhost:3000/", "Final frontend test")
    
    print(f"\nFinal Status:")
    print(f"Backend: {'OK' if backend_test else 'FAILED'}")
    print(f"Frontend: {'OK' if frontend_test else 'FAILED'}")
    
    return backend_test and frontend_test

if __name__ == "__main__":
    success = main()
    print(f"\nRestart {'successful' if success else 'completed with issues'}")
    sys.exit(0 if success else 1)