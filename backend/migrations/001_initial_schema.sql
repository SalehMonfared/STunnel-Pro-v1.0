-- STunnel Pro v1.0 Initial Database Schema
-- Version: 1.0.0

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(30) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    
    -- Account Information
    role VARCHAR(20) DEFAULT 'user' CHECK (role IN ('admin', 'moderator', 'user', 'guest')),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'suspended', 'banned')),
    
    -- Profile
    avatar TEXT,
    phone VARCHAR(20),
    company VARCHAR(100),
    department VARCHAR(100),
    
    -- Preferences
    language VARCHAR(2) DEFAULT 'en',
    timezone VARCHAR(50) DEFAULT 'UTC',
    theme VARCHAR(10) DEFAULT 'light' CHECK (theme IN ('light', 'dark', 'auto')),
    
    -- Security
    two_factor_enabled BOOLEAN DEFAULT FALSE,
    two_factor_secret VARCHAR(255),
    last_login_at TIMESTAMP,
    last_login_ip INET,
    password_changed_at TIMESTAMP DEFAULT NOW(),
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP,
    
    -- Limits and Quotas
    max_tunnels INTEGER DEFAULT 10,
    max_bandwidth_mbps INTEGER DEFAULT 100,
    max_connections INTEGER DEFAULT 1000,
    max_storage_gb INTEGER DEFAULT 10,
    daily_transfer_gb INTEGER DEFAULT 100,
    monthly_transfer_gb INTEGER DEFAULT 1000,
    can_create_public_tunnels BOOLEAN DEFAULT FALSE,
    can_use_custom_domains BOOLEAN DEFAULT FALSE,
    can_access_api BOOLEAN DEFAULT TRUE,
    
    -- API Access
    api_key VARCHAR(255) UNIQUE,
    api_key_created_at TIMESTAMP,
    
    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Tunnels table
CREATE TABLE tunnels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL,
    description TEXT,
    protocol VARCHAR(20) NOT NULL CHECK (protocol IN ('tcp', 'udp', 'ws', 'wss', 'tcpmux', 'wsmux', 'wssmux', 'utcpmux', 'uwsmux')),
    status VARCHAR(20) DEFAULT 'inactive' CHECK (status IN ('active', 'inactive', 'error', 'connecting')),
    
    -- Server Configuration
    server_ip INET NOT NULL,
    server_port INTEGER NOT NULL CHECK (server_port > 0 AND server_port <= 65535),
    
    -- Client Configuration
    client_ip INET,
    client_port INTEGER CHECK (client_port > 0 AND client_port <= 65535),
    
    -- Target Configuration
    target_ip INET NOT NULL,
    target_port INTEGER NOT NULL CHECK (target_port > 0 AND target_port <= 65535),
    
    -- Authentication
    token VARCHAR(255) NOT NULL,
    
    -- MUX Configuration
    mux_enabled BOOLEAN DEFAULT TRUE,
    mux_connections INTEGER DEFAULT 8,
    mux_frame_size INTEGER DEFAULT 32768,
    mux_receive_buffer INTEGER DEFAULT 4194304,
    mux_stream_buffer INTEGER DEFAULT 65536,
    mux_version INTEGER DEFAULT 2,
    mux_channel_size INTEGER DEFAULT 2048,
    mux_connection_pool INTEGER DEFAULT 8,
    mux_heartbeat INTEGER DEFAULT 30,
    
    -- TLS Configuration
    tls_enabled BOOLEAN DEFAULT FALSE,
    tls_cert_file TEXT,
    tls_key_file TEXT,
    tls_ca_file TEXT,
    tls_insecure_skip_verify BOOLEAN DEFAULT FALSE,
    tls_min_version VARCHAR(10) DEFAULT '1.2',
    tls_max_version VARCHAR(10) DEFAULT '1.3',
    
    -- Monitoring
    last_seen TIMESTAMP,
    bytes_in BIGINT DEFAULT 0,
    bytes_out BIGINT DEFAULT 0,
    connection_count INTEGER DEFAULT 0,
    
    -- Metadata
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- User Sessions table
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(512) UNIQUE NOT NULL,
    refresh_token VARCHAR(512) UNIQUE,
    ip_address INET,
    user_agent TEXT,
    device_info TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    last_used_at TIMESTAMP DEFAULT NOW()
);

-- Tunnel Logs table
CREATE TABLE tunnel_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tunnel_id UUID NOT NULL REFERENCES tunnels(id) ON DELETE CASCADE,
    level VARCHAR(10) NOT NULL CHECK (level IN ('DEBUG', 'INFO', 'WARN', 'ERROR')),
    message TEXT NOT NULL,
    metadata JSONB,
    timestamp TIMESTAMP DEFAULT NOW()
);

-- Tunnel Metrics table
CREATE TABLE tunnel_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tunnel_id UUID NOT NULL REFERENCES tunnels(id) ON DELETE CASCADE,
    timestamp TIMESTAMP DEFAULT NOW(),
    bytes_in BIGINT DEFAULT 0,
    bytes_out BIGINT DEFAULT 0,
    connection_count INTEGER DEFAULT 0,
    latency DOUBLE PRECISION DEFAULT 0,
    cpu_usage DOUBLE PRECISION DEFAULT 0,
    memory_usage BIGINT DEFAULT 0,
    error_count INTEGER DEFAULT 0
);

-- Audit Logs table
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL,
    resource VARCHAR(50),
    resource_id VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    success BOOLEAN DEFAULT TRUE,
    error_message TEXT,
    metadata JSONB,
    timestamp TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_api_key ON users(api_key);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_role ON users(role);

CREATE INDEX idx_tunnels_user_id ON tunnels(user_id);
CREATE INDEX idx_tunnels_name ON tunnels(name);
CREATE INDEX idx_tunnels_status ON tunnels(status);
CREATE INDEX idx_tunnels_protocol ON tunnels(protocol);
CREATE INDEX idx_tunnels_server_port ON tunnels(server_port);

CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_token ON user_sessions(token);
CREATE INDEX idx_user_sessions_is_active ON user_sessions(is_active);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);

CREATE INDEX idx_tunnel_logs_tunnel_id ON tunnel_logs(tunnel_id);
CREATE INDEX idx_tunnel_logs_level ON tunnel_logs(level);
CREATE INDEX idx_tunnel_logs_timestamp ON tunnel_logs(timestamp);

CREATE INDEX idx_tunnel_metrics_tunnel_id ON tunnel_metrics(tunnel_id);
CREATE INDEX idx_tunnel_metrics_timestamp ON tunnel_metrics(timestamp);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp);

-- Unique constraints
ALTER TABLE tunnels ADD CONSTRAINT unique_tunnel_name_per_user UNIQUE (user_id, name);

-- Functions for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_tunnels_updated_at BEFORE UPDATE ON tunnels FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert default admin user (password: admin123!)
INSERT INTO users (
    username, 
    email, 
    password, 
    first_name, 
    last_name, 
    role,
    max_tunnels,
    max_bandwidth_mbps,
    max_connections,
    max_storage_gb,
    daily_transfer_gb,
    monthly_transfer_gb,
    can_create_public_tunnels,
    can_use_custom_domains,
    can_access_api
) VALUES (
    'admin',
    'admin@utunnel-pro.local',
    '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', -- admin123!
    'System',
    'Administrator',
    'admin',
    1000,
    10000,
    100000,
    1000,
    10000,
    100000,
    TRUE,
    TRUE,
    TRUE
);

-- Comments
COMMENT ON TABLE users IS 'System users with authentication and authorization';
COMMENT ON TABLE tunnels IS 'Tunnel configurations and status';
COMMENT ON TABLE user_sessions IS 'Active user sessions for authentication';
COMMENT ON TABLE tunnel_logs IS 'Tunnel activity and error logs';
COMMENT ON TABLE tunnel_metrics IS 'Tunnel performance metrics';
COMMENT ON TABLE audit_logs IS 'System audit trail for security';

COMMENT ON COLUMN users.role IS 'User role: admin, moderator, user, guest';
COMMENT ON COLUMN users.status IS 'Account status: active, inactive, suspended, banned';
COMMENT ON COLUMN tunnels.protocol IS 'Tunnel protocol: tcp, udp, ws, wss, tcpmux, wsmux, wssmux, utcpmux, uwsmux';
COMMENT ON COLUMN tunnels.status IS 'Tunnel status: active, inactive, error, connecting';
