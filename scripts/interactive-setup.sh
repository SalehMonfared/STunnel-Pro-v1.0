#!/bin/bash

# STunnel Pro v1.0 - Interactive Setup Script
# This script provides an interactive configuration experience

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
LIGHT_BLUE='\033[1;34m'
LIGHT_GRAY='\033[0;37m'
BOLD='\033[1m'
NC='\033[0m'

# Configuration variables
DB_NAME=""
DB_USER=""
DB_PASSWORD=""
JWT_SECRET=""
TELEGRAM_BOT_TOKEN=""
TELEGRAM_CHAT_ID=""
DOMAIN_NAME=""
ADMIN_EMAIL=""
SSL_ENABLED="false"

print_banner() {
    clear
    echo -e "${PURPLE}"
    cat << "EOF"
    ╔══════════════════════════════════════════════════════════════╗
    ║                  🚀 STunnel Pro v1.0 🚀                      ║
    ║                Interactive Configuration                     ║
    ╚══════════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
}

# Function to prompt for input with styled label
prompt_input() {
    local label="$1"
    local var_name="$2"
    local is_secret="$3"
    local default_value="$4"
    local validation="$5"
    
    while true; do
        echo ""
        echo -e "${LIGHT_BLUE}STunnel Pro${NC} ${LIGHT_GRAY}|${NC} ${CYAN}${label}${NC}"
        if [ -n "$default_value" ]; then
            echo -e "${LIGHT_GRAY}Press Enter for default: ${default_value}${NC}"
        fi
        echo -n "> "
        
        if [ "$is_secret" = "true" ]; then
            read -s input_value
            echo
        else
            read input_value
        fi
        
        if [ -z "$input_value" ] && [ -n "$default_value" ]; then
            input_value="$default_value"
        fi
        
        # Validation
        if [ -n "$validation" ]; then
            case $validation in
                "email")
                    if [[ $input_value =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; then
                        break
                    else
                        echo -e "${RED}❌ Invalid email format${NC}"
                        continue
                    fi
                    ;;
                "port")
                    if [[ $input_value =~ ^[0-9]+$ ]] && [ $input_value -ge 1 ] && [ $input_value -le 65535 ]; then
                        break
                    else
                        echo -e "${RED}❌ Invalid port number (1-65535)${NC}"
                        continue
                    fi
                    ;;
                "required")
                    if [ -n "$input_value" ]; then
                        break
                    else
                        echo -e "${RED}❌ This field is required${NC}"
                        continue
                    fi
                    ;;
                *)
                    break
                    ;;
            esac
        else
            break
        fi
    done
    
    eval "$var_name='$input_value'"
}

# Function to confirm settings
confirm_settings() {
    echo ""
    echo -e "${PURPLE}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}║                    📋 Configuration Summary                  ║${NC}"
    echo -e "${PURPLE}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    
    echo -e "${CYAN}🗄️  Database Configuration:${NC}"
    echo -e "${LIGHT_GRAY}   Database Name: ${DB_NAME}${NC}"
    echo -e "${LIGHT_GRAY}   Database User: ${DB_USER}${NC}"
    echo -e "${LIGHT_GRAY}   Password: ${DB_PASSWORD:+[SET]}${DB_PASSWORD:-[NOT SET]}${NC}"
    echo ""
    
    echo -e "${CYAN}🔐 Security:${NC}"
    echo -e "${LIGHT_GRAY}   JWT Secret: ${JWT_SECRET:+[GENERATED]}${JWT_SECRET:-[NOT SET]}${NC}"
    echo ""
    
    if [ -n "$TELEGRAM_BOT_TOKEN" ]; then
        echo -e "${CYAN}📱 Telegram:${NC}"
        echo -e "${LIGHT_GRAY}   Bot Token: ${TELEGRAM_BOT_TOKEN:0:10}...${NC}"
        echo -e "${LIGHT_GRAY}   Chat ID: ${TELEGRAM_CHAT_ID}${NC}"
        echo ""
    fi
    
    if [ -n "$DOMAIN_NAME" ]; then
        echo -e "${CYAN}🌐 Domain:${NC}"
        echo -e "${LIGHT_GRAY}   Domain: ${DOMAIN_NAME}${NC}"
        echo -e "${LIGHT_GRAY}   SSL: ${SSL_ENABLED}${NC}"
        echo ""
    fi
    
    echo -e "${CYAN}👤 Admin:${NC}"
    echo -e "${LIGHT_GRAY}   Email: ${ADMIN_EMAIL}${NC}"
    echo ""
    
    read -p "Do you want to proceed with this configuration? (y/N): " confirm
    if [[ ! $confirm =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}Configuration cancelled. Exiting...${NC}"
        exit 0
    fi
}

# Database configuration
configure_database() {
    echo ""
    echo -e "${PURPLE}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}║                   🗄️  Database Configuration                 ║${NC}"
    echo -e "${PURPLE}╚══════════════════════════════════════════════════════════════╝${NC}"
    
    prompt_input "Database Name" "DB_NAME" "false" "stunnel_pro"
    prompt_input "Database User" "DB_USER" "false" "stunnel"
    prompt_input "Database Password (leave empty to auto-generate)" "DB_PASSWORD" "true" ""
    
    # Generate secure password if not provided
    if [[ -z "$DB_PASSWORD" ]]; then
        DB_PASSWORD=$(openssl rand -base64 32)
        echo -e "${GREEN}✅ Generated secure database password${NC}"
    fi
}

# Security configuration
configure_security() {
    echo ""
    echo -e "${PURPLE}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}║                     🔐 Security Configuration                ║${NC}"
    echo -e "${PURPLE}╚══════════════════════════════════════════════════════════════╝${NC}"
    
    prompt_input "JWT Secret (leave empty to auto-generate)" "JWT_SECRET" "true" ""
    
    if [[ -z "$JWT_SECRET" ]]; then
        JWT_SECRET=$(openssl rand -base64 64)
        echo -e "${GREEN}✅ Generated secure JWT secret${NC}"
    fi
}

# Telegram configuration
configure_telegram() {
    echo ""
    echo -e "${PURPLE}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}║                  📱 Telegram Configuration                   ║${NC}"
    echo -e "${PURPLE}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    
    echo -e "${CYAN}📋 To set up Telegram notifications:${NC}"
    echo -e "${CYAN}   1. Message @BotFather on Telegram${NC}"
    echo -e "${CYAN}   2. Send: /newbot${NC}"
    echo -e "${CYAN}   3. Follow instructions to get your Bot Token${NC}"
    echo -e "${CYAN}   4. Message @userinfobot to get your Chat ID${NC}"
    echo ""
    echo -e "${LIGHT_GRAY}Leave empty to skip Telegram notifications${NC}"
    
    prompt_input "Telegram Bot Token" "TELEGRAM_BOT_TOKEN" "false" ""
    
    if [[ -n "$TELEGRAM_BOT_TOKEN" ]]; then
        prompt_input "Telegram Chat ID" "TELEGRAM_CHAT_ID" "false" ""
        
        # Test Telegram configuration
        if [[ -n "$TELEGRAM_CHAT_ID" ]]; then
            echo ""
            echo -e "${CYAN}Testing Telegram configuration...${NC}"
            
            test_response=$(curl -s "https://api.telegram.org/bot$TELEGRAM_BOT_TOKEN/sendMessage" \
                -d "chat_id=$TELEGRAM_CHAT_ID" \
                -d "text=🚀 STunnel Pro v1.0 setup started! Bot is working correctly.")
            
            if echo "$test_response" | grep -q '"ok":true'; then
                echo -e "${GREEN}✅ Telegram test message sent successfully!${NC}"
            else
                echo -e "${YELLOW}⚠️  Failed to send test message. Please check your credentials.${NC}"
                read -p "Continue anyway? (y/N): " continue_anyway
                if [[ ! $continue_anyway =~ ^[Yy]$ ]]; then
                    TELEGRAM_BOT_TOKEN=""
                    TELEGRAM_CHAT_ID=""
                fi
            fi
        fi
    fi
}

# Domain and SSL configuration
configure_domain() {
    echo ""
    echo -e "${PURPLE}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}║                   🌐 Domain & SSL Configuration              ║${NC}"
    echo -e "${PURPLE}╚══════════════════════════════════════════════════════════════╝${NC}"
    
    prompt_input "Domain Name (optional, for SSL)" "DOMAIN_NAME" "false" ""
    
    if [[ -n "$DOMAIN_NAME" ]]; then
        read -p "Enable SSL/HTTPS? (y/N): " enable_ssl
        if [[ $enable_ssl =~ ^[Yy]$ ]]; then
            SSL_ENABLED="true"
            echo -e "${GREEN}✅ SSL will be enabled${NC}"
        fi
    fi
}

# Admin configuration
configure_admin() {
    echo ""
    echo -e "${PURPLE}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}║                    👤 Admin Configuration                    ║${NC}"
    echo -e "${PURPLE}╚══════════════════════════════════════════════════════════════╝${NC}"
    
    prompt_input "Admin Email" "ADMIN_EMAIL" "false" "admin@localhost" "email"
}

# Create .env file
create_env_file() {
    echo ""
    echo -e "${CYAN}Creating configuration file...${NC}"
    
    cat > .env << EOF
# Database Configuration
DB_HOST=postgres
DB_PORT=5432
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=${DB_NAME}

# Redis Configuration
REDIS_HOST=redis
REDIS_PORT=6379

# Security
JWT_SECRET=${JWT_SECRET}
API_KEY=$(openssl rand -hex 32)

# Telegram Bot (Optional)
TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
TELEGRAM_CHAT_ID=${TELEGRAM_CHAT_ID}

# Application
LOG_LEVEL=info
GIN_MODE=release
ENVIRONMENT=production

# Frontend
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080

# Admin
ADMIN_EMAIL=${ADMIN_EMAIL}

# Domain (Optional)
DOMAIN_NAME=${DOMAIN_NAME}

# Monitoring
PROMETHEUS_ENABLED=true
GRAFANA_ADMIN_PASSWORD=admin

# SSL
SSL_ENABLED=${SSL_ENABLED}
SSL_CERT_PATH=
SSL_KEY_PATH=
EOF
    
    chmod 600 .env
    echo -e "${GREEN}✅ Configuration file created successfully!${NC}"
}

# Main function
main() {
    print_banner
    
    echo -e "${CYAN}Welcome to STunnel Pro v1.0 Interactive Setup!${NC}"
    echo -e "${LIGHT_GRAY}This wizard will guide you through the configuration process.${NC}"
    echo ""
    
    configure_database
    configure_security
    configure_telegram
    configure_domain
    configure_admin
    
    confirm_settings
    create_env_file
    
    echo ""
    echo -e "${GREEN}🎉 Configuration completed successfully!${NC}"
    echo -e "${CYAN}You can now run: docker-compose up -d${NC}"
    echo ""
}

# Run main function
main "$@"
