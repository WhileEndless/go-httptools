package rawhttp

import (
	"net"
	"sync"
	"time"
)

// PooledConnection represents a pooled connection
type PooledConnection struct {
	Conn         net.Conn
	LastUsed     time.Time
	Protocol     string
	IsHTTP2      bool
	RemoteAddr   string
	TLSNegotiated bool
}

// ConnectionPool manages a pool of persistent connections
type ConnectionPool struct {
	mu          sync.Mutex
	connections map[string][]*PooledConnection // key: "host:port"
	maxIdle     int
	maxIdleTime time.Duration
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool() *ConnectionPool {
	pool := &ConnectionPool{
		connections: make(map[string][]*PooledConnection),
		maxIdle:     10, // Maximum idle connections per host
		maxIdleTime: 90 * time.Second,
	}

	// Start cleanup goroutine
	go pool.cleanupLoop()

	return pool
}

// Get retrieves a connection from the pool or returns nil if none available
func (p *ConnectionPool) Get(key string) *PooledConnection {
	p.mu.Lock()
	defer p.mu.Unlock()

	conns, ok := p.connections[key]
	if !ok || len(conns) == 0 {
		return nil
	}

	// Get the most recently used connection
	conn := conns[len(conns)-1]
	p.connections[key] = conns[:len(conns)-1]

	// Check if connection is still alive
	if !isConnAlive(conn.Conn) {
		conn.Conn.Close()
		return nil
	}

	return conn
}

// Put adds a connection to the pool
func (p *ConnectionPool) Put(key string, conn *PooledConnection) {
	p.mu.Lock()
	defer p.mu.Unlock()

	conns := p.connections[key]

	// Don't exceed max idle connections
	if len(conns) >= p.maxIdle {
		conn.Conn.Close()
		return
	}

	conn.LastUsed = time.Now()
	p.connections[key] = append(conns, conn)
}

// Remove removes all connections for a given key
func (p *ConnectionPool) Remove(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conns, ok := p.connections[key]; ok {
		for _, conn := range conns {
			conn.Conn.Close()
		}
		delete(p.connections, key)
	}
}

// CloseAll closes all connections in the pool
func (p *ConnectionPool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for key, conns := range p.connections {
		for _, conn := range conns {
			conn.Conn.Close()
		}
		delete(p.connections, key)
	}
}

// cleanupLoop periodically removes stale connections
func (p *ConnectionPool) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		p.cleanup()
	}
}

// cleanup removes connections that have been idle too long
func (p *ConnectionPool) cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for key, conns := range p.connections {
		var activeConns []*PooledConnection

		for _, conn := range conns {
			if now.Sub(conn.LastUsed) > p.maxIdleTime || !isConnAlive(conn.Conn) {
				conn.Conn.Close()
			} else {
				activeConns = append(activeConns, conn)
			}
		}

		if len(activeConns) > 0 {
			p.connections[key] = activeConns
		} else {
			delete(p.connections, key)
		}
	}
}

// isConnAlive checks if a connection is still alive by reading with a timeout
func isConnAlive(conn net.Conn) bool {
	// Set a very short read deadline
	conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
	defer conn.SetReadDeadline(time.Time{})

	one := make([]byte, 1)
	_, err := conn.Read(one)

	// If we get a timeout, connection is alive (no data to read)
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	// Any other error or EOF means connection is dead
	return false
}
