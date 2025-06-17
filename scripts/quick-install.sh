#!/bin/bash

# STunnel Pro v1.0 - Quick Installation Script
# This script provides multiple installation methods

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
REPO_URL="https://github.com/SalehMonfared/stunnel-pro"
INSTALL_DIR="/opt/stunnel-pro"

print_banner() {
    echo -e "${PURPLE}"
    cat << "EOF"
    ╔══════════════════════════════════════════════════════════════╗
    ║                  🚀 STunnel Pro v1.0 🚀                      ║
    ║              Quick Installation Script                       ║
    ╚══════════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
}

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
    exit 1
}

info() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $1${NC}"
}

# Check if running as root
check_root() {
    if [[ $EUID -eq 0 ]]; then
        error "This script should not be run as root for security reasons. Please run as a regular user with sudo access."
    fi
}

# Detect OS and architecture
detect_system() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        armv7l) ARCH="arm" ;;
        i386|i686) ARCH="386" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac
    
    log "Detected system: $OS-$ARCH"
}

# Show installation options
show_options() {
    echo -e "${CYAN}Choose installation method:${NC}"
    echo "1) 🐳 Docker Compose (Recommended)"
    echo "2) 📦 Binary Installation"
    echo "3) 🔧 Development Setup"
    echo "4) ☸️  Kubernetes"
    echo "5) 🛠️  Custom Installation"
    echo ""
    read -p "Enter your choice (1-5): " choice
}

# Docker Compose installation
install_docker_compose() {
    info "Installing with Docker Compose..."
    
    # Check if Docker is installed
    if ! command -v docker &> /dev/null; then
        info "Installing Docker..."
        curl -fsSL https://get.docker.com | sh
        sudo usermod -aG docker $USER
        log "Docker installed. Please log out and log back in, then run this script again."
        exit 0
    fi
    
    # Check if Docker Compose is installed
    if ! command -v docker-compose &> /dev/null; then
        info "Installing Docker Compose..."
        sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
        sudo chmod +x /usr/local/bin/docker-compose
    fi
    
    # Clone repository
    if [ -d "$INSTALL_DIR" ]; then
        log "Updating existing installation..."
        cd $INSTALL_DIR
        git pull
    else
        log "Cloning repository..."
        git clone $REPO_URL $INSTALL_DIR
        cd $INSTALL_DIR
    fi
    
    # Start services
    log "Starting STunnel Pro v1.0 services..."
    docker-compose up -d

    # Wait for services to be ready
    log "Waiting for services to start..."
    sleep 30

    # Check if services are running
    if docker-compose ps | grep -q "Up"; then
        log "✅ STunnel Pro v1.0 installed successfully!"
        show_access_info
    else
        error "❌ Failed to start services. Check logs with: docker-compose logs"
    fi
}

# Binary installation
install_binary() {
    info "Installing binary version..."
    
    # Get latest release
    LATEST_RELEASE=$(curl -s https://api.github.com/repos/SalehMonfared/stunnel-pro/releases/latest | grep "tag_name" | cut -d '"' -f 4)
    
    if [ -z "$LATEST_RELEASE" ]; then
        error "Could not fetch latest release information"
    fi
    
    # Download binary
    DOWNLOAD_URL="https://github.com/SalehMonfared/stunnel-pro/releases/download/${LATEST_RELEASE}/stunnel-pro-${OS}-${ARCH}.tar.gz"

    log "Downloading STunnel Pro v1.0 ${LATEST_RELEASE}..."
    curl -L $DOWNLOAD_URL -o /tmp/stunnel-pro.tar.gz

    # Extract and install
    sudo mkdir -p $INSTALL_DIR
    sudo tar -xzf /tmp/stunnel-pro.tar.gz -C $INSTALL_DIR
    sudo chmod +x $INSTALL_DIR/stunnel-pro
    
    # Create systemd service
    create_systemd_service
    
    # Start service
    sudo systemctl enable stunnel-pro
    sudo systemctl start stunnel-pro

    log "✅ STunnel Pro v1.0 installed successfully!"
    show_access_info
}

# Development setup
install_development() {
    info "Setting up development environment..."
    
    # Check dependencies
    check_dev_dependencies
    
    # Clone repository
    git clone $REPO_URL $HOME/stunnel-pro-dev
    cd $HOME/stunnel-pro-dev
    
    # Setup backend
    log "Setting up backend..."
    cd backend
    go mod download
    
    # Setup frontend
    log "Setting up frontend..."
    cd ../frontend
    npm install
    
    # Start development services
    log "Starting development services..."
    cd ..
    docker-compose -f docker-compose.dev.yml up -d
    
    log "✅ Development environment ready!"
    echo -e "${CYAN}To start development:${NC}"
    echo "Backend: cd $HOME/stunnel-pro-dev/backend && go run cmd/server/main.go"
    echo "Frontend: cd $HOME/stunnel-pro-dev/frontend && npm run dev"
}

# Kubernetes installation
install_kubernetes() {
    info "Installing on Kubernetes..."
    
    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        error "kubectl is not installed. Please install kubectl first."
    fi
    
    # Clone repository
    git clone $REPO_URL /tmp/stunnel-pro
    cd /tmp/stunnel-pro
    
    # Apply Kubernetes manifests
    log "Applying Kubernetes manifests..."
    kubectl apply -f k8s/
    
    # Wait for deployment
    log "Waiting for deployment to be ready..."
    kubectl wait --for=condition=available --timeout=300s deployment/stunnel-pro-backend -n stunnel-pro

    log "✅ STunnel Pro v1.0 deployed to Kubernetes!"

    # Show access information
    echo -e "${CYAN}Access information:${NC}"
    kubectl get services -n stunnel-pro
}

# Custom installation
install_custom() {
    info "Custom installation..."
    
    echo -e "${CYAN}Custom installation options:${NC}"
    echo "1) Install specific version"
    echo "2) Install from source"
    echo "3) Install with custom configuration"
    
    read -p "Enter your choice (1-3): " custom_choice
    
    case $custom_choice in
        1) install_specific_version ;;
        2) install_from_source ;;
        3) install_with_config ;;
        *) error "Invalid choice" ;;
    esac
}

# Helper functions
check_dev_dependencies() {
    # Check Go
    if ! command -v go &> /dev/null; then
        error "Go is not installed. Please install Go 1.21 or higher."
    fi
    
    # Check Node.js
    if ! command -v node &> /dev/null; then
        error "Node.js is not installed. Please install Node.js 18 or higher."
    fi
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        error "Docker is not installed. Please install Docker first."
    fi
}

create_systemd_service() {
    sudo tee /etc/systemd/system/stunnel-pro.service > /dev/null << EOF
[Unit]
Description=STunnel Pro v1.0 Service
After=network.target

[Service]
Type=simple
User=stunnel
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/stunnel-pro
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    sudo systemctl daemon-reload
}

show_access_info() {
    echo -e "${GREEN}"
    echo "🎉 Installation completed successfully!"
    echo -e "${NC}"
    echo -e "${CYAN}Access URLs:${NC}"
    echo "🌐 Web Dashboard: http://localhost:3000"
    echo "🔧 API Documentation: http://localhost:8080/swagger"
    echo "📊 Grafana: http://localhost:3001 (admin/admin)"
    echo ""
    echo -e "${YELLOW}Next steps:${NC}"
    echo "1. Open http://localhost:3000 in your browser"
    echo "2. Create your admin account"
    echo "3. Configure your first tunnel"
    echo ""
    echo -e "${CYAN}Useful commands:${NC}"
    echo "📋 View logs: docker-compose logs -f"
    echo "🔄 Restart: docker-compose restart"
    echo "⏹️  Stop: docker-compose down"
}

# Main execution
main() {
    print_banner
    check_root
    detect_system
    show_options
    
    case $choice in
        1) install_docker_compose ;;
        2) install_binary ;;
        3) install_development ;;
        4) install_kubernetes ;;
        5) install_custom ;;
        *) error "Invalid choice" ;;
    esac
}

# Run main function
main "$@"
