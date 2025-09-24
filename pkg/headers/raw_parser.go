package headers

import (
	"bufio"
	"bytes"
	"strings"
	"sync"
)

// RawHeader represents a header with its exact original formatting preserved
type RawHeader struct {
	Name         string // Parsed name (for lookups)
	Value        string // Parsed value (for lookups)
	OriginalLine string // Exact original line including spacing
}

// OrderedHeadersRaw preserves exact formatting while allowing lookups
type OrderedHeadersRaw struct {
	mu        sync.RWMutex
	headers   []RawHeader       // Preserves exact order and formatting
	lookup    map[string]int    // Case-insensitive name -> last index
	rawLookup map[string]string // Case-insensitive name -> original case
}

// NewOrderedHeadersRaw creates a new OrderedHeadersRaw instance
func NewOrderedHeadersRaw() *OrderedHeadersRaw {
	return &OrderedHeadersRaw{
		headers:   make([]RawHeader, 0),
		lookup:    make(map[string]int),
		rawLookup: make(map[string]string),
	}
}

// ParseHeadersRaw parses raw HTTP headers preserving exact formatting
func ParseHeadersRaw(data []byte) (*OrderedHeadersRaw, error) {
	headers := NewOrderedHeadersRaw()
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines (end of headers)
		if len(strings.TrimSpace(line)) == 0 {
			break
		}

		originalLine := line

		// Find colon separator
		colonPos := strings.Index(line, ":")
		if colonPos == -1 {
			// Invalid header format, store as-is for fault tolerance
			headers.addRawHeader("X-Malformed-Header", line, originalLine)
			continue
		}

		name := strings.TrimSpace(line[:colonPos])
		value := strings.TrimSpace(line[colonPos+1:])

		// Handle empty header name (fault tolerance)
		if name == "" {
			name = "X-Empty-Header-Name"
		}

		headers.addRawHeader(name, value, originalLine)
	}

	return headers, scanner.Err()
}

// addRawHeader adds a header while preserving original formatting
func (h *OrderedHeadersRaw) addRawHeader(name, value, originalLine string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	lowerName := strings.ToLower(name)

	rawHeader := RawHeader{
		Name:         name,
		Value:        value,
		OriginalLine: originalLine,
	}

	// Add to headers list
	h.headers = append(h.headers, rawHeader)

	// Update lookup (points to last occurrence)
	h.lookup[lowerName] = len(h.headers) - 1
	h.rawLookup[lowerName] = name
}

// Set adds or updates a header, preserving case but not spacing (for new headers)
func (h *OrderedHeadersRaw) Set(name, value string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	lowerName := strings.ToLower(name)

	// If header exists, update the last occurrence
	if idx, exists := h.lookup[lowerName]; exists {
		h.headers[idx].Name = name
		h.headers[idx].Value = value
		h.headers[idx].OriginalLine = name + ": " + value // Standard format for new values
		h.rawLookup[lowerName] = name
	} else {
		// Add new header - call internal method without additional locking
		h.addRawHeaderUnsafe(name, value, name+": "+value)
	}
}

// SetAfter adds or updates a header, placing it after the specified header
func (h *OrderedHeadersRaw) SetAfter(name, value, afterHeader string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	lowerName := strings.ToLower(name)
	afterLower := strings.ToLower(afterHeader)

	// If header exists, update it in place
	if idx, exists := h.lookup[lowerName]; exists {
		h.headers[idx].Name = name
		h.headers[idx].Value = value
		h.headers[idx].OriginalLine = name + ": " + value
		h.rawLookup[lowerName] = name
		return
	}

	// Find position after the specified header
	insertPos := len(h.headers) // Default to end if not found
	for i, header := range h.headers {
		if strings.ToLower(header.Name) == afterLower {
			insertPos = i + 1
			break
		}
	}

	// Create new header
	rawHeader := RawHeader{
		Name:         name,
		Value:        value,
		OriginalLine: name + ": " + value,
	}

	// Insert at specific position
	h.headers = append(h.headers[:insertPos], append([]RawHeader{rawHeader}, h.headers[insertPos:]...)...)

	// Rebuild lookup map (indices changed)
	h.rebuildLookup()
}

// SetBefore adds or updates a header, placing it before the specified header
func (h *OrderedHeadersRaw) SetBefore(name, value, beforeHeader string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	lowerName := strings.ToLower(name)
	beforeLower := strings.ToLower(beforeHeader)

	// If header exists, update it in place
	if idx, exists := h.lookup[lowerName]; exists {
		h.headers[idx].Name = name
		h.headers[idx].Value = value
		h.headers[idx].OriginalLine = name + ": " + value
		h.rawLookup[lowerName] = name
		return
	}

	// Find position before the specified header
	insertPos := len(h.headers) // Default to end if not found
	for i, header := range h.headers {
		if strings.ToLower(header.Name) == beforeLower {
			insertPos = i
			break
		}
	}

	// Create new header
	rawHeader := RawHeader{
		Name:         name,
		Value:        value,
		OriginalLine: name + ": " + value,
	}

	// Insert at specific position
	h.headers = append(h.headers[:insertPos], append([]RawHeader{rawHeader}, h.headers[insertPos:]...)...)

	// Rebuild lookup map (indices changed)
	h.rebuildLookup()
}

// SetAt adds or updates a header at specific index position
func (h *OrderedHeadersRaw) SetAt(name, value string, index int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	lowerName := strings.ToLower(name)

	// If header exists, update it in place
	if idx, exists := h.lookup[lowerName]; exists {
		h.headers[idx].Name = name
		h.headers[idx].Value = value
		h.headers[idx].OriginalLine = name + ": " + value
		h.rawLookup[lowerName] = name
		return
	}

	// Validate index
	if index < 0 || index > len(h.headers) {
		index = len(h.headers) // Default to end
	}

	// Create new header
	rawHeader := RawHeader{
		Name:         name,
		Value:        value,
		OriginalLine: name + ": " + value,
	}

	// Insert at specific position
	h.headers = append(h.headers[:index], append([]RawHeader{rawHeader}, h.headers[index:]...)...)

	// Rebuild lookup map (indices changed)
	h.rebuildLookup()
}

// addRawHeaderUnsafe adds a header without locking (for internal use)
func (h *OrderedHeadersRaw) addRawHeaderUnsafe(name, value, originalLine string) {
	lowerName := strings.ToLower(name)

	rawHeader := RawHeader{
		Name:         name,
		Value:        value,
		OriginalLine: originalLine,
	}

	// Add to headers list
	h.headers = append(h.headers, rawHeader)

	// Update lookup (points to last occurrence)
	h.lookup[lowerName] = len(h.headers) - 1
	h.rawLookup[lowerName] = name
}

// Get retrieves a header value (case-insensitive)
func (h *OrderedHeadersRaw) Get(name string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	lowerName := strings.ToLower(name)
	if idx, exists := h.lookup[lowerName]; exists {
		return h.headers[idx].Value
	}
	return ""
}

// GetRaw retrieves the original case of the header name
func (h *OrderedHeadersRaw) GetRaw(name string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.rawLookup[strings.ToLower(name)]
}

// Has checks if a header exists (case-insensitive)
func (h *OrderedHeadersRaw) Has(name string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	_, exists := h.lookup[strings.ToLower(name)]
	return exists
}

// Del removes a header
func (h *OrderedHeadersRaw) Del(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	lowerName := strings.ToLower(name)
	if idx, exists := h.lookup[lowerName]; exists {
		// Remove from headers slice
		h.headers = append(h.headers[:idx], h.headers[idx+1:]...)

		// Rebuild lookup map (indices changed)
		h.rebuildLookup()
	}
}

// rebuildLookup rebuilds the lookup map after deletions
func (h *OrderedHeadersRaw) rebuildLookup() {
	h.lookup = make(map[string]int)
	h.rawLookup = make(map[string]string)

	for i, header := range h.headers {
		lowerName := strings.ToLower(header.Name)
		h.lookup[lowerName] = i // Last occurrence wins
		h.rawLookup[lowerName] = header.Name
	}
}

// All returns all headers preserving exact formatting
func (h *OrderedHeadersRaw) All() []RawHeader {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Return copy to prevent external modification
	result := make([]RawHeader, len(h.headers))
	copy(result, h.headers)
	return result
}

// AllStandard returns headers in standard format for compatibility
func (h *OrderedHeadersRaw) AllStandard() []Header {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]Header, len(h.headers))
	for i, rawHeader := range h.headers {
		result[i] = Header{
			Name:  rawHeader.Name,
			Value: rawHeader.Value,
		}
	}
	return result
}

// Len returns the number of headers
func (h *OrderedHeadersRaw) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.headers)
}

// BuildRaw reconstructs headers with exact original formatting
func (h *OrderedHeadersRaw) BuildRaw() []byte {
	var buf bytes.Buffer

	for _, header := range h.All() {
		buf.WriteString(header.OriginalLine)
		// Don't add line ending - preserve exactly what was in original
		buf.WriteString("\n")
	}

	return buf.Bytes()
}
