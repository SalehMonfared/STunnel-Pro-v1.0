#!/bin/bash

# STunnel Pro v1.0 - Advanced Installation Script
# Version: 1.0.0
# Author: SalehMonfared
# Repository: https://github.com/SalehMonfared/stunnel-pro

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
INSTALL_DIR="/opt/stunnel-pro"
CONFIG_DIR="/etc/stunnel-pro"
LOG_DIR="/var/log/stunnel-pro"
SERVICE_NAME="stunnel-pro"
GITHUB_REPO="https://github.com/SalehMonfared/stunnel-pro"
DOCKER_COMPOSE_VERSION="2.21.0"

# System requirements
MIN_RAM_GB=2
MIN_DISK_GB=10
REQUIRED_PORTS=(80 443 8080 5432 6379)

# Functions
print_banner() {
    echo -e "${PURPLE}"
    cat << "EOF"
    ╔════════════════════════════════════════════════════════════════════╗
    ║                                                                    ║
    ║    ███████╗████████╗██╗   ██╗███╗   ██╗███╗   ██╗███████╗          ║
    ║    ██╔════╝╚══██╔══╝██║   ██║████╗  ██║████╗  ██║██╔════╝          ║
    ║     █████╗    ██║   ██║   ██║██╔██╗ ██║██╔██╗ ██║█████╗            ║
    ║         ██╔   ██║   ██║   ██║██║╚██╗██║██║╚██╗██║██╔══╝            ║
    ║    ███████╗   ██║   ╚██████╔╝██║ ╚████║██║ ╚████║███████╗          ║
    ║    ╚══════╝   ╚═╝    ╚═════╝ ╚═╝  ╚═══╝╚═╝  ╚═══╝╚══════╝          ║
    ║                                                                    ║
    ║                    🚀 STunnel Pro v1.0 🚀                         ║
    ║               Advanced Tunnel Management System                    ║
    ║                                                                    ║
    ╚════════════════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
}

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
    exit 1
}

info() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $1${NC}"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        error "This script must be run as root. Please use sudo."
    fi
}

detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$NAME
        VER=$VERSION_ID
    else
        error "Cannot detect operating system"
    fi
    
    log "Detected OS: $OS $VER"
}

check_architecture() {
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l)
            ARCH="arm"
            ;;
        i386|i686)
            ARCH="386"
            ;;
        *)
            error "Unsupported architecture: $ARCH"
            ;;
    esac
    
    log "Architecture: $ARCH"
}

check_system_requirements() {
    info "Checking system requirements..."
    
    # Check RAM
    TOTAL_RAM=$(free -g | awk '/^Mem:/{print $2}')
    if [[ $TOTAL_RAM -lt $MIN_RAM_GB ]]; then
        warn "System has ${TOTAL_RAM}GB RAM, minimum ${MIN_RAM_GB}GB recommended"
    fi
    
    # Check disk space
    AVAILABLE_DISK=$(df / | awk 'NR==2{print int($4/1024/1024)}')
    if [[ $AVAILABLE_DISK -lt $MIN_DISK_GB ]]; then
        error "Insufficient disk space. Available: ${AVAILABLE_DISK}GB, Required: ${MIN_DISK_GB}GB"
    fi
    
    # Check ports
    for port in "${REQUIRED_PORTS[@]}"; do
        if netstat -tuln | grep -q ":$port "; then
            warn "Port $port is already in use"
        fi
    done
    
    log "System requirements check completed"
}

install_dependencies() {
    info "Installing dependencies..."
    
    case $OS in
        *"Ubuntu"*|*"Debian"*)
            apt-get update
            apt-get install -y curl wget git unzip software-properties-common apt-transport-https ca-certificates gnupg lsb-release
            ;;
        *"CentOS"*|*"Red Hat"*|*"Rocky"*|*"AlmaLinux"*)
            yum update -y
            yum install -y curl wget git unzip yum-utils device-mapper-persistent-data lvm2
            ;;
        *"Fedora"*)
            dnf update -y
            dnf install -y curl wget git unzip dnf-plugins-core
            ;;
        *)
            error "Unsupported operating system: $OS"
            ;;
    esac
    
    log "Dependencies installed successfully"
}

install_docker() {
    if command -v docker &> /dev/null; then
        log "Docker is already installed"
        return
    fi
    
    info "Installing Docker..."
    
    case $OS in
        *"Ubuntu"*|*"Debian"*)
            curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
            echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
            apt-get update
            apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            ;;
        *"CentOS"*|*"Red Hat"*|*"Rocky"*|*"AlmaLinux"*)
            yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
            yum install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            ;;
        *)
            error "Docker installation not supported for $OS"
            ;;
    esac
    
    systemctl enable docker
    systemctl start docker
    
    log "Docker installed successfully"
}

install_docker_compose() {
    if command -v docker-compose &> /dev/null; then
        log "Docker Compose is already installed"
        return
    fi
    
    info "Installing Docker Compose..."
    
    curl -L "https://github.com/docker/compose/releases/download/v${DOCKER_COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose
    ln -sf /usr/local/bin/docker-compose /usr/bin/docker-compose
    
    log "Docker Compose installed successfully"
}

create_directories() {
    info "Creating directories..."
    
    mkdir -p $INSTALL_DIR
    mkdir -p $CONFIG_DIR
    mkdir -p $LOG_DIR
    mkdir -p $INSTALL_DIR/{backend,frontend,tunnel-core,monitoring,nginx}
    
    log "Directories created successfully"
}

download_stunnel_pro() {
    info "Downloading STunnel Pro v1.0..."

    cd $INSTALL_DIR

    # Download latest release
    LATEST_RELEASE=$(curl -s https://api.github.com/repos/SalehMonfared/stunnel-pro/releases/latest | grep "tag_name" | cut -d '"' -f 4)

    if [[ -z "$LATEST_RELEASE" ]]; then
        warn "Could not fetch latest release, using development version"
        git clone $GITHUB_REPO.git .
    else
        wget -O stunnel-pro.tar.gz "https://github.com/SalehMonfared/stunnel-pro/archive/refs/tags/${LATEST_RELEASE}.tar.gz"
        tar -xzf stunnel-pro.tar.gz --strip-components=1
        rm stunnel-pro.tar.gz
    fi

    log "STunnel Pro v1.0 downloaded successfully"
}

configure_environment() {
    info "Configuring environment..."
    
    # Generate secure passwords and keys
    DB_PASSWORD=$(openssl rand -base64 32)
    JWT_SECRET=$(openssl rand -base64 64)
    API_KEY=$(openssl rand -hex 32)
    
    # Create environment file
    cat > $CONFIG_DIR/.env << EOF
# Database Configuration
DB_HOST=postgres
DB_PORT=5432
DB_USER=stunnel
DB_PASSWORD=$DB_PASSWORD
DB_NAME=stunnel_pro

# Redis Configuration
REDIS_HOST=redis
REDIS_PORT=6379

# Security
JWT_SECRET=$JWT_SECRET
API_KEY=$API_KEY

# Telegram Bot (Optional)
TELEGRAM_BOT_TOKEN=
TELEGRAM_CHAT_ID=

# Application
LOG_LEVEL=info
GIN_MODE=release
ENVIRONMENT=production

# Frontend
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080

# Monitoring
PROMETHEUS_ENABLED=true
GRAFANA_ADMIN_PASSWORD=admin

# SSL (Optional)
SSL_ENABLED=false
SSL_CERT_PATH=
SSL_KEY_PATH=
EOF
    
    # Set permissions
    chmod 600 $CONFIG_DIR/.env
    chown root:root $CONFIG_DIR/.env
    
    log "Environment configured successfully"
}

setup_ssl() {
    read -p "Do you want to enable SSL/HTTPS? (y/N): " enable_ssl
    
    if [[ $enable_ssl =~ ^[Yy]$ ]]; then
        info "Setting up SSL..."
        
        read -p "Enter your domain name: " domain_name
        
        if [[ -z "$domain_name" ]]; then
            warn "No domain provided, skipping SSL setup"
            return
        fi
        
        # Install certbot
        case $OS in
            *"Ubuntu"*|*"Debian"*)
                apt-get install -y certbot python3-certbot-nginx
                ;;
            *"CentOS"*|*"Red Hat"*|*"Rocky"*|*"AlmaLinux"*)
                yum install -y certbot python3-certbot-nginx
                ;;
        esac
        
        # Generate SSL certificate
        certbot certonly --standalone -d $domain_name --non-interactive --agree-tos --email admin@$domain_name
        
        # Update environment
        sed -i "s/SSL_ENABLED=false/SSL_ENABLED=true/" $CONFIG_DIR/.env
        sed -i "s|SSL_CERT_PATH=|SSL_CERT_PATH=/etc/letsencrypt/live/$domain_name/fullchain.pem|" $CONFIG_DIR/.env
        sed -i "s|SSL_KEY_PATH=|SSL_KEY_PATH=/etc/letsencrypt/live/$domain_name/privkey.pem|" $CONFIG_DIR/.env
        
        log "SSL configured successfully"
    fi
}

configure_telegram() {
    read -p "Do you want to configure Telegram notifications? (y/N): " setup_telegram
    
    if [[ $setup_telegram =~ ^[Yy]$ ]]; then
        info "Configuring Telegram..."
        
        echo -e "${CYAN}To set up Telegram notifications:${NC}"
        echo -e "${CYAN}1. Create a bot by messaging @BotFather on Telegram${NC}"
        echo -e "${CYAN}2. Get your chat ID by messaging @userinfobot${NC}"
        echo ""
        
        read -p "Enter your Telegram Bot Token: " bot_token
        read -p "Enter your Telegram Chat ID: " chat_id
        
        if [[ -n "$bot_token" && -n "$chat_id" ]]; then
            sed -i "s/TELEGRAM_BOT_TOKEN=/TELEGRAM_BOT_TOKEN=$bot_token/" $CONFIG_DIR/.env
            sed -i "s/TELEGRAM_CHAT_ID=/TELEGRAM_CHAT_ID=$chat_id/" $CONFIG_DIR/.env
            log "Telegram configured successfully"
        else
            warn "Invalid Telegram credentials, skipping configuration"
        fi
    fi
}

start_services() {
    info "Starting UTunnel Pro services..."
    
    cd $INSTALL_DIR
    
    # Copy environment file
    cp $CONFIG_DIR/.env .env
    
    # Start services
    docker-compose up -d
    
    # Wait for services to start
    sleep 30
    
    # Check service status
    if docker-compose ps | grep -q "Up"; then
        log "Services started successfully"
    else
        error "Failed to start services"
    fi
}

create_systemd_service() {
    info "Creating systemd service..."
    
    cat > /etc/systemd/system/$SERVICE_NAME.service << EOF
[Unit]
Description=UTunnel Pro Service
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=$INSTALL_DIR
ExecStart=/usr/local/bin/docker-compose up -d
ExecStop=/usr/local/bin/docker-compose down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable $SERVICE_NAME
    
    log "Systemd service created successfully"
}

setup_firewall() {
    info "Configuring firewall..."
    
    if command -v ufw &> /dev/null; then
        ufw allow 80/tcp
        ufw allow 443/tcp
        ufw allow 8080/tcp
        ufw --force enable
    elif command -v firewall-cmd &> /dev/null; then
        firewall-cmd --permanent --add-port=80/tcp
        firewall-cmd --permanent --add-port=443/tcp
        firewall-cmd --permanent --add-port=8080/tcp
        firewall-cmd --reload
    else
        warn "No firewall detected, please configure manually"
    fi
    
    log "Firewall configured successfully"
}

show_completion_message() {
    echo -e "${GREEN}"
    cat << "EOF"
    ╔══════════════════════════════════════════════════════════════╗
    ║                                                              ║
    ║                  🎉 Installation Complete! 🎉                ║
    ║                                                              ║
    ╚══════════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
    
    echo -e "${CYAN}UTunnel Pro has been installed successfully!${NC}"
    echo ""
    echo -e "${YELLOW}Access URLs:${NC}"
    echo -e "  🌐 Web Dashboard: http://localhost:3000"
    echo -e "  🔧 API Endpoint:  http://localhost:8080"
    echo -e "  📊 Grafana:       http://localhost:3001 (admin/admin)"
    echo -e "  📈 Prometheus:    http://localhost:9091"
    echo ""
    echo -e "${YELLOW}Useful Commands:${NC}"
    echo -e "  📋 View logs:     docker-compose logs -f"
    echo -e "  🔄 Restart:       systemctl restart $SERVICE_NAME"
    echo -e "  ⏹️  Stop:          systemctl stop $SERVICE_NAME"
    echo -e "  📊 Status:        systemctl status $SERVICE_NAME"
    echo ""
    echo -e "${YELLOW}Configuration:${NC}"
    echo -e "  📁 Install Dir:   $INSTALL_DIR"
    echo -e "  ⚙️  Config Dir:    $CONFIG_DIR"
    echo -e "  📝 Log Dir:       $LOG_DIR"
    echo ""
    echo -e "${GREEN}Happy tunneling! 🚀${NC}"
}

# Main installation flow
main() {
    print_banner
    
    log "Starting STunnel Pro v1.0 installation..."

    check_root
    detect_os
    check_architecture
    check_system_requirements
    install_dependencies
    install_docker
    install_docker_compose
    create_directories
    download_stunnel_pro
    configure_environment
    setup_ssl
    configure_telegram
    start_services
    create_systemd_service
    setup_firewall
    
    show_completion_message
}

# Run main function
main "$@"
