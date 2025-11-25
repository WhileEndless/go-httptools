package chunked

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Decode decodes chunked transfer encoding to plain body
// Always succeeds - if malformed, returns what it can parse
// Returns decoded body and any trailers found after final chunk
func Decode(chunkedBody []byte) (body []byte, trailers map[string]string) {
	trailers = make(map[string]string)

	if len(chunkedBody) == 0 {
		return []byte{}, trailers
	}

	var result bytes.Buffer
	data := chunkedBody
	pos := 0

	for pos < len(data) {
		// Find chunk size line (terminated by \r\n or \n)
		lineEnd := bytes.Index(data[pos:], []byte("\r\n"))
		useCRLF := true
		if lineEnd == -1 {
			// Try Unix line ending
			lineEnd = bytes.Index(data[pos:], []byte("\n"))
			useCRLF = false
			if lineEnd == -1 {
				// No more valid chunks, return what we have
				break
			}
		}

		// Parse chunk size (hex)
		sizeLine := string(data[pos : pos+lineEnd])

		// Handle chunk extensions (e.g., "5;name=value")
		if idx := strings.Index(sizeLine, ";"); idx != -1 {
			sizeLine = sizeLine[:idx]
		}

		sizeLine = strings.TrimSpace(sizeLine)

		// Parse hex size
		chunkSize, err := strconv.ParseInt(sizeLine, 16, 64)
		if err != nil || chunkSize < 0 {
			// Invalid chunk size, best effort: stop here
			break
		}

		// Move past size line
		if useCRLF {
			pos += lineEnd + 2 // Skip \r\n
		} else {
			pos += lineEnd + 1 // Skip \n
		}

		// If chunk size is 0, this is the last chunk
		if chunkSize == 0 {
			// Parse trailers (if any)
			trailers = parseTrailers(data[pos:])
			break
		}

		// Read chunk data
		if pos+int(chunkSize) > len(data) {
			// Not enough data, take what we can
			result.Write(data[pos:])
			break
		}

		result.Write(data[pos : pos+int(chunkSize)])
		pos += int(chunkSize)

		// Skip trailing \r\n or \n after chunk data
		if pos < len(data) {
			if pos+1 < len(data) && data[pos] == '\r' && data[pos+1] == '\n' {
				pos += 2
			} else if data[pos] == '\n' {
				pos += 1
			}
			// If no line ending found, continue anyway (best effort)
		}
	}

	return result.Bytes(), trailers
}

// parseTrailers parses HTTP trailers after final chunk
// Format: "Header-Name: value\r\n" repeated, ending with \r\n\r\n
func parseTrailers(data []byte) map[string]string {
	trailers := make(map[string]string)

	if len(data) == 0 {
		return trailers
	}

	// Trailers end with double line ending
	lines := bytes.Split(data, []byte("\r\n"))
	if len(lines) == 0 {
		// Try Unix line endings
		lines = bytes.Split(data, []byte("\n"))
	}

	for _, line := range lines {
		if len(line) == 0 {
			// Empty line signals end of trailers
			break
		}

		// Parse "Name: Value"
		parts := bytes.SplitN(line, []byte(":"), 2)
		if len(parts) == 2 {
			name := strings.TrimSpace(string(parts[0]))
			value := strings.TrimSpace(string(parts[1]))
			trailers[name] = value
		}
	}

	return trailers
}

// Encode encodes data with chunked transfer encoding
// chunkSize specifies the size of each chunk (must be > 0)
// If chunkSize <= 0, uses default of 8192 bytes
func Encode(data []byte, chunkSize int) []byte {
	if chunkSize <= 0 {
		chunkSize = 8192 // Default chunk size
	}

	var result bytes.Buffer
	pos := 0

	for pos < len(data) {
		// Determine chunk size for this iteration
		remaining := len(data) - pos
		currentChunkSize := chunkSize
		if remaining < chunkSize {
			currentChunkSize = remaining
		}

		// Write chunk size in hex
		result.WriteString(fmt.Sprintf("%x\r\n", currentChunkSize))

		// Write chunk data
		result.Write(data[pos : pos+currentChunkSize])
		result.WriteString("\r\n")

		pos += currentChunkSize
	}

	// Write final chunk (size 0)
	result.WriteString("0\r\n\r\n")

	return result.Bytes()
}

// EncodeWithTrailers encodes data with chunked transfer encoding and trailers
func EncodeWithTrailers(data []byte, chunkSize int, trailers map[string]string) []byte {
	if chunkSize <= 0 {
		chunkSize = 8192
	}

	var result bytes.Buffer
	pos := 0

	// Encode chunks
	for pos < len(data) {
		remaining := len(data) - pos
		currentChunkSize := chunkSize
		if remaining < chunkSize {
			currentChunkSize = remaining
		}

		result.WriteString(fmt.Sprintf("%x\r\n", currentChunkSize))
		result.Write(data[pos : pos+currentChunkSize])
		result.WriteString("\r\n")

		pos += currentChunkSize
	}

	// Write final chunk (size 0)
	result.WriteString("0\r\n")

	// Write trailers
	for name, value := range trailers {
		result.WriteString(fmt.Sprintf("%s: %s\r\n", name, value))
	}

	// Final CRLF
	result.WriteString("\r\n")

	return result.Bytes()
}

// IsChunked heuristically checks if data appears to be chunked encoded
// This is a best-effort check and may have false positives/negatives
func IsChunked(data []byte) bool {
	if len(data) < 3 {
		return false
	}

	// Look for pattern: hex-digits followed by \r\n or \n
	// Try to find first line
	lineEnd := bytes.Index(data, []byte("\r\n"))
	if lineEnd == -1 {
		lineEnd = bytes.Index(data, []byte("\n"))
		if lineEnd == -1 {
			return false
		}
	}

	if lineEnd == 0 || lineEnd > 16 {
		// Chunk size lines are typically short (hex number)
		return false
	}

	firstLine := string(data[:lineEnd])
	firstLine = strings.TrimSpace(firstLine)

	// Handle chunk extensions
	if idx := strings.Index(firstLine, ";"); idx != -1 {
		firstLine = firstLine[:idx]
	}

	// Try to parse as hex
	_, err := strconv.ParseInt(firstLine, 16, 64)
	return err == nil
}

// ============================================================================
// Streaming Chunked Decoder
// ============================================================================

// DecodeReader provides streaming chunked transfer decoding
// Implements io.Reader interface to decode chunked data on-the-fly
type DecodeReader struct {
	reader       *bufio.Reader
	remaining    int64  // remaining bytes in current chunk
	eof          bool   // reached final 0-length chunk
	trailers     map[string]string
	trailersRead bool
}

// NewDecodeReader creates a new streaming chunked decoder reader
// The returned reader decodes chunked transfer encoding on-the-fly as it's read
func NewDecodeReader(r io.Reader) *DecodeReader {
	var br *bufio.Reader
	if b, ok := r.(*bufio.Reader); ok {
		br = b
	} else {
		br = bufio.NewReader(r)
	}

	return &DecodeReader{
		reader:   br,
		trailers: make(map[string]string),
	}
}

// Read implements io.Reader interface
// Decodes chunked transfer encoding and returns raw body data
func (d *DecodeReader) Read(p []byte) (int, error) {
	if d.eof {
		return 0, io.EOF
	}

	// If we have remaining data in the current chunk, read it
	if d.remaining > 0 {
		toRead := int64(len(p))
		if toRead > d.remaining {
			toRead = d.remaining
		}
		n, err := d.reader.Read(p[:toRead])
		d.remaining -= int64(n)

		// If we finished the chunk, read the trailing CRLF
		if d.remaining == 0 {
			d.readChunkTerminator()
		}

		if err != nil && err != io.EOF {
			return n, err
		}
		return n, nil
	}

	// Read next chunk size
	sizeLine, err := d.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			d.eof = true
		}
		return 0, err
	}

	// Parse chunk size (strip CRLF and any extensions)
	sizeLine = strings.TrimRight(sizeLine, "\r\n")
	if idx := strings.Index(sizeLine, ";"); idx != -1 {
		sizeLine = sizeLine[:idx]
	}
	sizeLine = strings.TrimSpace(sizeLine)

	chunkSize, err := strconv.ParseInt(sizeLine, 16, 64)
	if err != nil {
		// Invalid chunk size - might be end or corrupted
		d.eof = true
		return 0, io.EOF
	}

	// Zero-length chunk signals end of body
	if chunkSize == 0 {
		d.eof = true
		d.readTrailers()
		return 0, io.EOF
	}

	d.remaining = chunkSize

	// Now read from the chunk
	return d.Read(p)
}

// readChunkTerminator reads and discards the CRLF after chunk data
func (d *DecodeReader) readChunkTerminator() {
	// Read until \n (which handles both \r\n and \n)
	d.reader.ReadString('\n')
}

// readTrailers reads optional trailers after the final chunk
func (d *DecodeReader) readTrailers() {
	if d.trailersRead {
		return
	}
	d.trailersRead = true

	for {
		line, err := d.reader.ReadString('\n')
		if err != nil {
			return
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			// Empty line signals end of trailers
			return
		}

		// Parse "Name: Value"
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			d.trailers[name] = value
		}
	}
}

// Trailers returns any trailers found after the final chunk
// Only valid after Read returns io.EOF
func (d *DecodeReader) Trailers() map[string]string {
	return d.trailers
}

// ============================================================================
// Streaming Chunked Encoder
// ============================================================================

// EncodeWriter provides streaming chunked transfer encoding
// Implements io.WriteCloser interface to encode data as chunked on-the-fly
type EncodeWriter struct {
	writer    io.Writer
	chunkSize int
	closed    bool
	trailers  map[string]string
}

// NewEncodeWriter creates a new streaming chunked encoder writer
// chunkSize specifies the maximum size of each chunk (0 = default 8192)
// The returned writer encodes data with chunked transfer encoding as it's written
// IMPORTANT: Always call Close() when done to write the final zero-length chunk
func NewEncodeWriter(w io.Writer, chunkSize int) *EncodeWriter {
	if chunkSize <= 0 {
		chunkSize = 8192
	}
	return &EncodeWriter{
		writer:    w,
		chunkSize: chunkSize,
		trailers:  make(map[string]string),
	}
}

// SetTrailer sets a trailer to be written after the final chunk
// Must be called before Close()
func (e *EncodeWriter) SetTrailer(name, value string) {
	e.trailers[name] = value
}

// Write implements io.Writer interface
// Writes data as one or more chunks
func (e *EncodeWriter) Write(p []byte) (int, error) {
	if e.closed {
		return 0, io.ErrClosedPipe
	}

	if len(p) == 0 {
		return 0, nil
	}

	totalWritten := 0
	for len(p) > 0 {
		// Determine chunk size for this iteration
		chunkLen := len(p)
		if chunkLen > e.chunkSize {
			chunkLen = e.chunkSize
		}

		// Write chunk size in hex
		sizeStr := fmt.Sprintf("%x\r\n", chunkLen)
		if _, err := e.writer.Write([]byte(sizeStr)); err != nil {
			return totalWritten, err
		}

		// Write chunk data
		n, err := e.writer.Write(p[:chunkLen])
		totalWritten += n
		if err != nil {
			return totalWritten, err
		}

		// Write chunk terminator
		if _, err := e.writer.Write([]byte("\r\n")); err != nil {
			return totalWritten, err
		}

		p = p[chunkLen:]
	}

	return totalWritten, nil
}

// Close implements io.Closer interface
// Writes the final zero-length chunk and any trailers
// MUST be called to complete the chunked encoding
func (e *EncodeWriter) Close() error {
	if e.closed {
		return nil
	}
	e.closed = true

	// Write final chunk (size 0)
	if _, err := e.writer.Write([]byte("0\r\n")); err != nil {
		return err
	}

	// Write trailers if any
	for name, value := range e.trailers {
		trailer := fmt.Sprintf("%s: %s\r\n", name, value)
		if _, err := e.writer.Write([]byte(trailer)); err != nil {
			return err
		}
	}

	// Write final CRLF
	_, err := e.writer.Write([]byte("\r\n"))
	return err
}
