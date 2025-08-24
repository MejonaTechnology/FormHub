#!/bin/bash

# FormHub Production Deployment Script
# This script automates the deployment process for FormHub

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="FormHub"
DEPLOY_DIR="/opt/formhub"
BACKUP_DIR="/opt/formhub/backups"
DOCKER_COMPOSE_FILE="docker-compose.prod.yml"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if Docker is installed
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    # Check if Docker Compose is installed
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    # Check if running as root or with sudo
    if [ "$EUID" -ne 0 ]; then
        log_error "This script must be run as root or with sudo."
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

create_directories() {
    log_info "Creating deployment directories..."
    
    mkdir -p "$DEPLOY_DIR"
    mkdir -p "$BACKUP_DIR"
    mkdir -p "$DEPLOY_DIR/nginx/ssl"
    mkdir -p "$DEPLOY_DIR/logs"
    
    log_success "Directories created"
}

backup_existing_deployment() {
    if [ -d "$DEPLOY_DIR" ] && [ "$(ls -A $DEPLOY_DIR)" ]; then
        log_info "Backing up existing deployment..."
        
        BACKUP_NAME="formhub_backup_$(date +%Y%m%d_%H%M%S)"
        tar -czf "$BACKUP_DIR/$BACKUP_NAME.tar.gz" -C "$DEPLOY_DIR" . 2>/dev/null || true
        
        log_success "Backup created: $BACKUP_DIR/$BACKUP_NAME.tar.gz"
    fi
}

copy_deployment_files() {
    log_info "Copying deployment files..."
    
    # Copy docker-compose and configuration files
    cp -r deploy/* "$DEPLOY_DIR/"
    cp -r backend "$DEPLOY_DIR/"
    cp -r frontend "$DEPLOY_DIR/"
    
    # Copy environment file if it doesn't exist
    if [ ! -f "$DEPLOY_DIR/.env" ]; then
        cp "$DEPLOY_DIR/.env.production" "$DEPLOY_DIR/.env"
        log_warning "Environment file copied. Please edit $DEPLOY_DIR/.env with your actual values."
    fi
    
    log_success "Deployment files copied"
}

generate_ssl_certificate() {
    log_info "Setting up SSL certificate..."
    
    if [ ! -f "$DEPLOY_DIR/nginx/ssl/cert.pem" ]; then
        # Generate self-signed certificate for development
        # In production, replace with Let's Encrypt or proper SSL certificate
        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout "$DEPLOY_DIR/nginx/ssl/key.pem" \
            -out "$DEPLOY_DIR/nginx/ssl/cert.pem" \
            -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"
        
        log_warning "Self-signed SSL certificate generated. Replace with proper SSL certificate for production."
    else
        log_info "SSL certificate already exists"
    fi
}

deploy_application() {
    log_info "Deploying FormHub application..."
    
    cd "$DEPLOY_DIR"
    
    # Pull latest images and build
    docker-compose -f "$DOCKER_COMPOSE_FILE" pull
    docker-compose -f "$DOCKER_COMPOSE_FILE" build --no-cache
    
    # Stop existing containers
    docker-compose -f "$DOCKER_COMPOSE_FILE" down
    
    # Start new containers
    docker-compose -f "$DOCKER_COMPOSE_FILE" up -d
    
    log_success "FormHub deployed successfully"
}

wait_for_services() {
    log_info "Waiting for services to be ready..."
    
    # Wait for database
    timeout=60
    while [ $timeout -gt 0 ]; do
        if docker-compose -f "$DEPLOY_DIR/$DOCKER_COMPOSE_FILE" exec -T postgres pg_isready -q; then
            break
        fi
        sleep 2
        timeout=$((timeout - 2))
    done
    
    if [ $timeout -le 0 ]; then
        log_error "Database did not start within expected time"
        exit 1
    fi
    
    # Wait for API
    timeout=60
    while [ $timeout -gt 0 ]; do
        if curl -f http://localhost:8080/health >/dev/null 2>&1; then
            break
        fi
        sleep 2
        timeout=$((timeout - 2))
    done
    
    if [ $timeout -le 0 ]; then
        log_error "API did not start within expected time"
        exit 1
    fi
    
    log_success "All services are ready"
}

run_health_checks() {
    log_info "Running health checks..."
    
    # Check API health
    if curl -f http://localhost:8080/health >/dev/null 2>&1; then
        log_success "API health check passed"
    else
        log_error "API health check failed"
        exit 1
    fi
    
    # Check database connection
    if docker-compose -f "$DEPLOY_DIR/$DOCKER_COMPOSE_FILE" exec -T postgres pg_isready -q; then
        log_success "Database health check passed"
    else
        log_error "Database health check failed"
        exit 1
    fi
    
    # Check Redis connection
    if docker-compose -f "$DEPLOY_DIR/$DOCKER_COMPOSE_FILE" exec -T redis redis-cli ping >/dev/null 2>&1; then
        log_success "Redis health check passed"
    else
        log_error "Redis health check failed"
        exit 1
    fi
    
    log_success "All health checks passed"
}

setup_monitoring() {
    log_info "Setting up monitoring and logging..."
    
    # Create log rotation configuration
    cat > /etc/logrotate.d/formhub << EOF
$DEPLOY_DIR/logs/*.log {
    daily
    missingok
    rotate 52
    compress
    delaycompress
    notifempty
    sharedscripts
    postrotate
        docker-compose -f $DEPLOY_DIR/$DOCKER_COMPOSE_FILE restart api
    endscript
}
EOF
    
    log_success "Monitoring and logging configured"
}

setup_systemd_service() {
    log_info "Setting up systemd service..."
    
    cat > /etc/systemd/system/formhub.service << EOF
[Unit]
Description=FormHub Service
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=$DEPLOY_DIR
ExecStart=/usr/bin/docker-compose -f $DOCKER_COMPOSE_FILE up -d
ExecStop=/usr/bin/docker-compose -f $DOCKER_COMPOSE_FILE down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable formhub.service
    
    log_success "Systemd service configured"
}

cleanup() {
    log_info "Cleaning up old Docker images..."
    
    # Remove unused Docker images to free up space
    docker image prune -f
    
    log_success "Cleanup completed"
}

show_deployment_info() {
    log_success "FormHub deployment completed successfully!"
    echo
    echo "=== Deployment Information ==="
    echo "Project: $PROJECT_NAME"
    echo "Deploy Directory: $DEPLOY_DIR"
    echo "Backup Directory: $BACKUP_DIR"
    echo
    echo "=== Service URLs ==="
    echo "API: http://localhost:8080"
    echo "Frontend: http://localhost:3000"
    echo "Health Check: http://localhost:8080/health"
    echo
    echo "=== Management Commands ==="
    echo "Start services: docker-compose -f $DEPLOY_DIR/$DOCKER_COMPOSE_FILE up -d"
    echo "Stop services: docker-compose -f $DEPLOY_DIR/$DOCKER_COMPOSE_FILE down"
    echo "View logs: docker-compose -f $DEPLOY_DIR/$DOCKER_COMPOSE_FILE logs -f"
    echo "Restart service: systemctl restart formhub"
    echo
    echo "=== Next Steps ==="
    echo "1. Edit $DEPLOY_DIR/.env with your production values"
    echo "2. Configure your domain and SSL certificates"
    echo "3. Set up proper database backups"
    echo "4. Configure monitoring and alerting"
    echo
    log_warning "Remember to secure your server and configure firewall rules!"
}

# Main deployment process
main() {
    log_info "Starting FormHub deployment..."
    
    check_prerequisites
    create_directories
    backup_existing_deployment
    copy_deployment_files
    generate_ssl_certificate
    deploy_application
    wait_for_services
    run_health_checks
    setup_monitoring
    setup_systemd_service
    cleanup
    show_deployment_info
    
    log_success "FormHub deployment completed successfully!"
}

# Handle script interruption
trap 'log_error "Deployment interrupted"; exit 1' INT TERM

# Run main function
main "$@"