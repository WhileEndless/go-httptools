package chunked

import (
	"bytes"
	"fmt"
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
