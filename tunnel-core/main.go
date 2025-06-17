package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

// Configuration
type Config struct {
	Mode       string
	Protocol   string
	Listen     string
	Target     string
	Token      string
	CertFile   string
	KeyFile    string
	MuxEnabled bool
	MuxStreams int
	Debug      bool
}

// TunnelManager manages tunnel connections
type TunnelManager struct {
	config    *Config
	listener  net.Listener
	sessions  map[string]*yamux.Session
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// ConnectionStats tracks connection statistics
type ConnectionStats struct {
	BytesIn     int64
	BytesOut    int64
	Connections int64
	Errors      int64
	StartTime   time.Time
}

var stats = &ConnectionStats{StartTime: time.Now()}

func main() {
	config := parseFlags()
	
	if config.Debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Println("Debug mode enabled")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := &TunnelManager{
		config:   config,
		sessions: make(map[string]*yamux.Session),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down gracefully...")
		cancel()
	}()

	// Start tunnel based on mode
	switch config.Mode {
	case "server":
		err := manager.startServer()
		if err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	case "client":
		err := manager.startClient()
		if err != nil {
			log.Fatalf("Failed to start client: %v", err)
		}
	default:
		log.Fatalf("Invalid mode: %s. Use 'server' or 'client'", config.Mode)
	}

	manager.wg.Wait()
	log.Println("Tunnel stopped")
}

func parseFlags() *Config {
	config := &Config{}
	
	flag.StringVar(&config.Mode, "mode", "server", "Mode: server or client")
	flag.StringVar(&config.Protocol, "protocol", "tcp", "Protocol: tcp, udp, ws, wss")
	flag.StringVar(&config.Listen, "listen", "0.0.0.0:8080", "Listen address")
	flag.StringVar(&config.Target, "target", "127.0.0.1:22", "Target address")
	flag.StringVar(&config.Token, "token", "", "Authentication token")
	flag.StringVar(&config.CertFile, "cert", "", "TLS certificate file")
	flag.StringVar(&config.KeyFile, "key", "", "TLS private key file")
	flag.BoolVar(&config.MuxEnabled, "mux", true, "Enable multiplexing")
	flag.IntVar(&config.MuxStreams, "mux-streams", 8, "Number of multiplexed streams")
	flag.BoolVar(&config.Debug, "debug", false, "Enable debug logging")
	
	flag.Parse()
	
	if config.Token == "" {
		log.Fatal("Token is required")
	}
	
	return config
}

func (tm *TunnelManager) startServer() error {
	log.Printf("Starting %s server on %s -> %s", tm.config.Protocol, tm.config.Listen, tm.config.Target)

	switch tm.config.Protocol {
	case "tcp":
		return tm.startTCPServer()
	case "udp":
		return tm.startUDPServer()
	case "ws":
		return tm.startWebSocketServer(false)
	case "wss":
		return tm.startWebSocketServer(true)
	default:
		return fmt.Errorf("unsupported protocol: %s", tm.config.Protocol)
	}
}

func (tm *TunnelManager) startTCPServer() error {
	listener, err := net.Listen("tcp", tm.config.Listen)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer listener.Close()

	tm.listener = listener
	log.Printf("TCP server listening on %s", tm.config.Listen)

	for {
		select {
		case <-tm.ctx.Done():
			return nil
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			if tm.ctx.Err() != nil {
				return nil
			}
			log.Printf("Accept error: %v", err)
			continue
		}

		tm.wg.Add(1)
		go tm.handleTCPConnection(conn)
	}
}

func (tm *TunnelManager) handleTCPConnection(clientConn net.Conn) {
	defer tm.wg.Done()
	defer clientConn.Close()

	stats.Connections++
	
	if tm.config.Debug {
		log.Printf("New TCP connection from %s", clientConn.RemoteAddr())
	}

	// Connect to target
	targetConn, err := net.DialTimeout("tcp", tm.config.Target, 10*time.Second)
	if err != nil {
		log.Printf("Failed to connect to target %s: %v", tm.config.Target, err)
		stats.Errors++
		return
	}
	defer targetConn.Close()

	// Handle multiplexing if enabled
	if tm.config.MuxEnabled {
		tm.handleMuxConnection(clientConn, targetConn)
	} else {
		tm.handleDirectConnection(clientConn, targetConn)
	}
}

func (tm *TunnelManager) handleMuxConnection(clientConn, targetConn net.Conn) {
	// Create yamux session
	session, err := yamux.Server(clientConn, yamux.DefaultConfig())
	if err != nil {
		log.Printf("Failed to create yamux session: %v", err)
		stats.Errors++
		return
	}
	defer session.Close()

	// Store session
	sessionID := fmt.Sprintf("%s-%d", clientConn.RemoteAddr(), time.Now().UnixNano())
	tm.mu.Lock()
	tm.sessions[sessionID] = session
	tm.mu.Unlock()

	defer func() {
		tm.mu.Lock()
		delete(tm.sessions, sessionID)
		tm.mu.Unlock()
	}()

	// Handle streams
	for {
		stream, err := session.AcceptStream()
		if err != nil {
			if tm.config.Debug {
				log.Printf("Stream accept error: %v", err)
			}
			break
		}

		go tm.handleStream(stream, targetConn)
	}
}

func (tm *TunnelManager) handleStream(stream net.Conn, targetConn net.Conn) {
	defer stream.Close()

	// Create new connection to target for each stream
	target, err := net.DialTimeout("tcp", tm.config.Target, 10*time.Second)
	if err != nil {
		log.Printf("Failed to connect to target: %v", err)
		stats.Errors++
		return
	}
	defer target.Close()

	tm.handleDirectConnection(stream, target)
}

func (tm *TunnelManager) handleDirectConnection(client, target net.Conn) {
	// Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	// Client to target
	go func() {
		defer wg.Done()
		n, err := io.Copy(target, client)
		stats.BytesIn += n
		if err != nil && tm.config.Debug {
			log.Printf("Client to target copy error: %v", err)
		}
	}()

	// Target to client
	go func() {
		defer wg.Done()
		n, err := io.Copy(client, target)
		stats.BytesOut += n
		if err != nil && tm.config.Debug {
			log.Printf("Target to client copy error: %v", err)
		}
	}()

	wg.Wait()
}

func (tm *TunnelManager) startUDPServer() error {
	addr, err := net.ResolveUDPAddr("udp", tm.config.Listen)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen UDP: %w", err)
	}
	defer conn.Close()

	log.Printf("UDP server listening on %s", tm.config.Listen)

	buffer := make([]byte, 65536)
	clientMap := make(map[string]*net.UDPConn)
	var mu sync.RWMutex

	for {
		select {
		case <-tm.ctx.Done():
			return nil
		default:
		}

		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if tm.ctx.Err() != nil {
				return nil
			}
			log.Printf("UDP read error: %v", err)
			continue
		}

		stats.BytesIn += int64(n)
		clientKey := clientAddr.String()

		mu.RLock()
		targetConn, exists := clientMap[clientKey]
		mu.RUnlock()

		if !exists {
			// Create new connection to target
			targetAddr, err := net.ResolveUDPAddr("udp", tm.config.Target)
			if err != nil {
				log.Printf("Failed to resolve target address: %v", err)
				continue
			}

			targetConn, err = net.DialUDP("udp", nil, targetAddr)
			if err != nil {
				log.Printf("Failed to connect to target: %v", err)
				continue
			}

			mu.Lock()
			clientMap[clientKey] = targetConn
			mu.Unlock()

			// Start response handler
			go tm.handleUDPResponse(conn, targetConn, clientAddr, clientKey, clientMap, &mu)
		}

		// Forward to target
		_, err = targetConn.Write(buffer[:n])
		if err != nil {
			log.Printf("Failed to write to target: %v", err)
			mu.Lock()
			delete(clientMap, clientKey)
			mu.Unlock()
			targetConn.Close()
		}
	}
}

func (tm *TunnelManager) handleUDPResponse(serverConn *net.UDPConn, targetConn *net.UDPConn, clientAddr *net.UDPAddr, clientKey string, clientMap map[string]*net.UDPConn, mu *sync.RWMutex) {
	defer func() {
		mu.Lock()
		delete(clientMap, clientKey)
		mu.Unlock()
		targetConn.Close()
	}()

	buffer := make([]byte, 65536)
	targetConn.SetReadDeadline(time.Now().Add(5 * time.Minute))

	for {
		n, err := targetConn.Read(buffer)
		if err != nil {
			if tm.config.Debug {
				log.Printf("UDP target read error: %v", err)
			}
			break
		}

		stats.BytesOut += int64(n)

		_, err = serverConn.WriteToUDP(buffer[:n], clientAddr)
		if err != nil {
			log.Printf("Failed to write to client: %v", err)
			break
		}

		targetConn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	}
}

func (tm *TunnelManager) startWebSocketServer(useSSL bool) error {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Validate token
			token := r.Header.Get("Authorization")
			return token == "Bearer "+tm.config.Token
		},
	}

	http.HandleFunc("/tunnel", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}
		defer conn.Close()

		stats.Connections++
		tm.handleWebSocketConnection(conn)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"bytes_in": %d,
			"bytes_out": %d,
			"connections": %d,
			"errors": %d,
			"uptime": "%s"
		}`, stats.BytesIn, stats.BytesOut, stats.Connections, stats.Errors, time.Since(stats.StartTime))
	})

	server := &http.Server{
		Addr:    tm.config.Listen,
		Handler: nil,
	}

	if useSSL {
		if tm.config.CertFile == "" || tm.config.KeyFile == "" {
			return fmt.Errorf("SSL certificate and key files are required for WSS")
		}
		
		cert, err := tls.LoadX509KeyPair(tm.config.CertFile, tm.config.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to load SSL certificate: %w", err)
		}
		
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		
		log.Printf("WSS server listening on %s", tm.config.Listen)
		return server.ListenAndServeTLS("", "")
	} else {
		log.Printf("WS server listening on %s", tm.config.Listen)
		return server.ListenAndServe()
	}
}

func (tm *TunnelManager) handleWebSocketConnection(wsConn *websocket.Conn) {
	// Connect to target
	targetConn, err := net.DialTimeout("tcp", tm.config.Target, 10*time.Second)
	if err != nil {
		log.Printf("Failed to connect to target: %v", err)
		stats.Errors++
		return
	}
	defer targetConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	// WebSocket to target
	go func() {
		defer wg.Done()
		for {
			_, data, err := wsConn.ReadMessage()
			if err != nil {
				break
			}
			stats.BytesIn += int64(len(data))
			_, err = targetConn.Write(data)
			if err != nil {
				break
			}
		}
	}()

	// Target to WebSocket
	go func() {
		defer wg.Done()
		buffer := make([]byte, 32768)
		for {
			n, err := targetConn.Read(buffer)
			if err != nil {
				break
			}
			stats.BytesOut += int64(n)
			err = wsConn.WriteMessage(websocket.BinaryMessage, buffer[:n])
			if err != nil {
				break
			}
		}
	}()

	wg.Wait()
}

func (tm *TunnelManager) startClient() error {
	log.Printf("Starting %s client connecting to %s", tm.config.Protocol, tm.config.Listen)
	
	// Client mode implementation would go here
	// This would connect to a server and establish the tunnel from the client side
	
	return fmt.Errorf("client mode not implemented yet")
}
