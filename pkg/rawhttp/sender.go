package rawhttp

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/http2"
)

// Sender handles sending raw HTTP requests over TCP/TLS
type Sender struct {
	pool *ConnectionPool
}

// NewSender creates a new Sender instance
func NewSender() *Sender {
	return &Sender{
		pool: NewConnectionPool(),
	}
}

// Do sends a raw HTTP request and returns the raw response
func (s *Sender) Do(ctx context.Context, rawRequest []byte, opts Options) (*Response, error) {
	// Set default options
	opts.SetDefaults()

	resp := NewResponse()
	startTime := time.Now()

	// DNS resolution (if ConnIP not specified)
	var targetIP string
	var dnsStart time.Time

	if opts.ConnIP != "" {
		targetIP = opts.ConnIP
	} else {
		dnsStart = time.Now()
		ips, err := net.LookupIP(opts.Host)
		if err != nil {
			return nil, NewDNSError(err)
		}
		if len(ips) == 0 {
			return nil, NewDNSError(fmt.Errorf("no IP addresses found for host: %s", opts.Host))
		}
		targetIP = ips[0].String()
		resp.Timing.DNSLookup = time.Since(dnsStart)
	}

	resp.ConnectedIP = targetIP
	resp.ConnectedPort = opts.Port

	// Build connection key for pooling
	connKey := fmt.Sprintf("%s:%d", targetIP, opts.Port)

	// Try to get connection from pool
	var conn net.Conn
	var pooledConn *PooledConnection
	var protocol string

	if opts.ReuseConnection {
		pooledConn = s.pool.Get(connKey)
		if pooledConn != nil {
			conn = pooledConn.Conn
			protocol = pooledConn.Protocol
		}
	}

	// If no pooled connection, create new one
	if conn == nil {
		var err error
		conn, protocol, err = s.connect(ctx, targetIP, opts, resp)
		if err != nil {
			return nil, err
		}

		// Ensure connection is closed if we return early
		shouldClose := true
		defer func() {
			if shouldClose {
				conn.Close()
			}
		}()

		// Store protocol in response
		resp.Protocol = protocol

		// For pooled connection, we'll add it back to pool later
		if opts.ReuseConnection {
			shouldClose = false
		}
	} else {
		resp.Protocol = protocol
	}

	// Send request
	if err := s.writeRequest(conn, rawRequest, opts.WriteTimeout); err != nil {
		return nil, fmt.Errorf("write request failed: %w", err)
	}

	// Read response
	readStart := time.Now()
	rawResponse, err := s.readResponse(conn, opts.ReadTimeout, opts.BodyMemLimit)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	resp.Timing.TTFB = time.Since(readStart)
	resp.Raw = rawResponse

	// Parse response (optional - for convenience)
	if err := s.parseResponse(resp); err != nil {
		// Parsing error is not fatal - we still have raw data
		// Just log or ignore
	}

	resp.Timing.Total = time.Since(startTime)

	// Return connection to pool if reuse is enabled
	if opts.ReuseConnection && conn != nil {
		pooledConn := &PooledConnection{
			Conn:         conn,
			LastUsed:     time.Now(),
			Protocol:     protocol,
			IsHTTP2:      protocol == "HTTP/2",
			RemoteAddr:   fmt.Sprintf("%s:%d", targetIP, opts.Port),
			TLSNegotiated: opts.Scheme == "https",
		}
		s.pool.Put(connKey, pooledConn)
	}

	return resp, nil
}

// connect establishes a connection (with proxy support if configured)
func (s *Sender) connect(ctx context.Context, targetIP string, opts Options, resp *Response) (net.Conn, string, error) {
	// If proxy is configured, connect through proxy
	if opts.ProxyURL != "" {
		return s.connectViaProxy(ctx, targetIP, opts, resp)
	}

	// Direct connection
	return s.connectDirect(ctx, targetIP, opts, resp)
}

// connectDirect establishes a direct connection to the target
func (s *Sender) connectDirect(ctx context.Context, targetIP string, opts Options, resp *Response) (net.Conn, string, error) {
	addr := fmt.Sprintf("%s:%d", targetIP, opts.Port)

	// TCP connection
	tcpStart := time.Now()
	dialer := &net.Dialer{
		Timeout: opts.ConnTimeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, "", NewConnectionError(err)
	}

	resp.Timing.TCPConnect = time.Since(tcpStart)

	// If HTTPS, perform TLS handshake
	if opts.Scheme == "https" {
		tlsStart := time.Now()
		tlsConfig := opts.BuildTLSConfig()

		tlsConn := tls.Client(conn, tlsConfig)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			conn.Close()
			return nil, "", NewTLSError(err)
		}

		resp.Timing.TLSHandshake = time.Since(tlsStart)

		// Determine protocol from ALPN
		state := tlsConn.ConnectionState()
		protocol := "HTTP/1.1"
		if state.NegotiatedProtocol == "h2" {
			protocol = "HTTP/2"
		}

		return tlsConn, protocol, nil
	}

	// For HTTP, check if H2C is enabled
	if opts.EnableH2C {
		return conn, "HTTP/2", nil
	}

	return conn, "HTTP/1.1", nil
}

// connectViaProxy establishes a connection through an upstream proxy
func (s *Sender) connectViaProxy(ctx context.Context, targetIP string, opts Options, resp *Response) (net.Conn, string, error) {
	proxyStart := time.Now()

	// Parse proxy URL
	proxyURL, err := url.Parse(opts.ProxyURL)
	if err != nil {
		return nil, "", NewProxyError(fmt.Errorf("invalid proxy URL: %w", err))
	}

	// Connect to proxy
	dialer := &net.Dialer{
		Timeout: opts.ConnTimeout,
	}

	proxyAddr := proxyURL.Host
	conn, err := dialer.DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		return nil, "", NewProxyError(err)
	}

	resp.Timing.ProxyConnect = time.Since(proxyStart)

	// For HTTPS through HTTP proxy, send CONNECT request
	if opts.Scheme == "https" && proxyURL.Scheme == "http" {
		targetAddr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)
		connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", targetAddr, targetAddr)

		if _, err := conn.Write([]byte(connectReq)); err != nil {
			conn.Close()
			return nil, "", NewProxyError(fmt.Errorf("failed to send CONNECT: %w", err))
		}

		// Read CONNECT response
		reader := bufio.NewReader(conn)
		statusLine, err := reader.ReadString('\n')
		if err != nil {
			conn.Close()
			return nil, "", NewProxyError(fmt.Errorf("failed to read CONNECT response: %w", err))
		}

		if !strings.Contains(statusLine, "200") {
			conn.Close()
			return nil, "", NewProxyError(fmt.Errorf("CONNECT failed: %s", statusLine))
		}

		// Read and discard headers
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				conn.Close()
				return nil, "", NewProxyError(err)
			}
			if line == "\r\n" || line == "\n" {
				break
			}
		}

		// Now perform TLS handshake
		tlsStart := time.Now()
		tlsConfig := opts.BuildTLSConfig()
		tlsConn := tls.Client(conn, tlsConfig)

		if err := tlsConn.HandshakeContext(ctx); err != nil {
			conn.Close()
			return nil, "", NewTLSError(err)
		}

		resp.Timing.TLSHandshake = time.Since(tlsStart)

		// Determine protocol from ALPN
		state := tlsConn.ConnectionState()
		protocol := "HTTP/1.1"
		if state.NegotiatedProtocol == "h2" {
			protocol = "HTTP/2"
		}

		return tlsConn, protocol, nil
	}

	// For HTTP through HTTP proxy, just return the connection
	return conn, "HTTP/1.1", nil
}

// writeRequest writes the raw request to the connection
func (s *Sender) writeRequest(conn net.Conn, rawRequest []byte, timeout time.Duration) error {
	if timeout > 0 {
		conn.SetWriteDeadline(time.Now().Add(timeout))
		defer conn.SetWriteDeadline(time.Time{})
	}

	_, err := conn.Write(rawRequest)
	return err
}

// readResponse reads the raw response from the connection
func (s *Sender) readResponse(conn net.Conn, timeout time.Duration, maxSize int64) ([]byte, error) {
	if timeout > 0 {
		conn.SetReadDeadline(time.Now().Add(timeout))
		defer conn.SetReadDeadline(time.Time{})
	}

	var buf bytes.Buffer
	reader := bufio.NewReader(conn)

	// Read status line
	statusLine, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	buf.Write(statusLine)

	// Read headers
	headerBuf, err := s.readHeaders(reader)
	if err != nil {
		return nil, err
	}
	buf.Write(headerBuf)

	// Parse headers to determine body length
	headers := parseHeadersQuick(headerBuf)

	// Read body
	bodyBuf, err := s.readBody(reader, headers, maxSize)
	if err != nil {
		return nil, err
	}
	buf.Write(bodyBuf)

	return buf.Bytes(), nil
}

// readHeaders reads HTTP headers until blank line
func (s *Sender) readHeaders(reader *bufio.Reader) ([]byte, error) {
	var buf bytes.Buffer

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}

		buf.Write(line)

		// Check for end of headers (blank line)
		if len(line) == 2 && line[0] == '\r' && line[1] == '\n' {
			break
		}
		if len(line) == 1 && line[0] == '\n' {
			break
		}
	}

	return buf.Bytes(), nil
}

// readBody reads the HTTP body based on headers
func (s *Sender) readBody(reader *bufio.Reader, headers map[string]string, maxSize int64) ([]byte, error) {
	// Check for Content-Length
	if contentLength, ok := headers["content-length"]; ok {
		var length int64
		fmt.Sscanf(contentLength, "%d", &length)

		if length > maxSize {
			return nil, ErrBodyTooLarge
		}

		body := make([]byte, length)
		_, err := io.ReadFull(reader, body)
		return body, err
	}

	// Check for chunked encoding
	if transferEncoding, ok := headers["transfer-encoding"]; ok {
		if strings.Contains(strings.ToLower(transferEncoding), "chunked") {
			return s.readChunkedBody(reader, maxSize)
		}
	}

	// No content-length or chunked encoding - read until EOF or max size
	var buf bytes.Buffer
	limitedReader := io.LimitReader(reader, maxSize)
	_, err := io.Copy(&buf, limitedReader)

	return buf.Bytes(), err
}

// readChunkedBody reads a chunked transfer-encoded body
func (s *Sender) readChunkedBody(reader *bufio.Reader, maxSize int64) ([]byte, error) {
	var buf bytes.Buffer
	var totalSize int64

	for {
		// Read chunk size line
		sizeLine, err := reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		buf.Write(sizeLine)

		// Parse chunk size (hex)
		sizeStr := strings.TrimSpace(string(sizeLine))
		// Remove any chunk extensions
		if idx := strings.Index(sizeStr, ";"); idx != -1 {
			sizeStr = sizeStr[:idx]
		}

		var chunkSize int64
		_, err = fmt.Sscanf(sizeStr, "%x", &chunkSize)
		if err != nil {
			return nil, err
		}

		// If chunk size is 0, we've reached the end
		if chunkSize == 0 {
			// Read trailing headers (if any) until blank line
			for {
				line, err := reader.ReadBytes('\n')
				if err != nil {
					return nil, err
				}
				buf.Write(line)
				if len(line) == 2 && line[0] == '\r' && line[1] == '\n' {
					break
				}
				if len(line) == 1 && line[0] == '\n' {
					break
				}
			}
			break
		}

		totalSize += chunkSize
		if totalSize > maxSize {
			return nil, ErrBodyTooLarge
		}

		// Read chunk data
		chunkData := make([]byte, chunkSize)
		_, err = io.ReadFull(reader, chunkData)
		if err != nil {
			return nil, err
		}
		buf.Write(chunkData)

		// Read trailing CRLF
		trailingCRLF, err := reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		buf.Write(trailingCRLF)
	}

	return buf.Bytes(), nil
}

// parseHeadersQuick does a quick parse of headers for body reading
func parseHeadersQuick(headerBuf []byte) map[string]string {
	headers := make(map[string]string)
	lines := bytes.Split(headerBuf, []byte("\n"))

	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		parts := bytes.SplitN(line, []byte(":"), 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.ToLower(string(bytes.TrimSpace(parts[0])))
		value := string(bytes.TrimSpace(parts[1]))
		headers[name] = value
	}

	return headers
}

// parseResponse parses the raw response into structured fields
func (s *Sender) parseResponse(resp *Response) error {
	if len(resp.Raw) == 0 {
		return fmt.Errorf("empty response")
	}

	lines := bytes.Split(resp.Raw, []byte("\n"))
	if len(lines) == 0 {
		return fmt.Errorf("invalid response")
	}

	// Parse status line
	statusLine := string(bytes.TrimSpace(lines[0]))
	parts := strings.SplitN(statusLine, " ", 3)
	if len(parts) < 2 {
		return fmt.Errorf("invalid status line")
	}

	fmt.Sscanf(parts[1], "%d", &resp.StatusCode)

	// Parse headers
	resp.Headers = make(map[string][]string)
	i := 1
	for i < len(lines) {
		line := bytes.TrimSpace(lines[i])
		if len(line) == 0 {
			break
		}

		headerParts := bytes.SplitN(line, []byte(":"), 2)
		if len(headerParts) == 2 {
			name := string(bytes.TrimSpace(headerParts[0]))
			value := string(bytes.TrimSpace(headerParts[1]))
			resp.Headers[name] = append(resp.Headers[name], value)
		}

		i++
	}

	// Body is everything after the blank line
	if i+1 < len(lines) {
		bodyStart := bytes.Index(resp.Raw, []byte("\r\n\r\n"))
		if bodyStart == -1 {
			bodyStart = bytes.Index(resp.Raw, []byte("\n\n"))
			if bodyStart != -1 {
				resp.Body = resp.Raw[bodyStart+2:]
			}
		} else {
			resp.Body = resp.Raw[bodyStart+4:]
		}
	}

	return nil
}

// Close closes all pooled connections
func (s *Sender) Close() {
	s.pool.CloseAll()
}

// HTTP/2 support functions

// isHTTP2Connection checks if we should use HTTP/2 framing
func isHTTP2Connection(protocol string) bool {
	return protocol == "HTTP/2"
}

// sendHTTP2Request sends an HTTP/2 request (using h2 framing)
func (s *Sender) sendHTTP2Request(conn net.Conn, rawRequest []byte) error {
	// This is a simplified version - in a real implementation,
	// we would use golang.org/x/net/http2 for proper framing

	// For now, we'll use the http2 package for framing
	// This is just a placeholder to show the structure
	_ = http2.NewFramer(conn, conn)

	// In a full implementation, we would:
	// 1. Parse the HTTP/1.1 request
	// 2. Convert to HTTP/2 frames (HEADERS, DATA)
	// 3. Send frames over connection

	// For simplicity, we'll just send as HTTP/1.1 for now
	// and let the server handle it
	return nil
}
