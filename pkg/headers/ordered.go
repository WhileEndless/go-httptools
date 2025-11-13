package headers

import (
	"strings"
	"sync"
)

// OrderedHeaders preserves the order of HTTP headers and handles case-insensitive lookups
type OrderedHeaders struct {
	mu     sync.RWMutex
	order  []string          // Preserves insertion order
	values map[string]string // Case-insensitive storage (lowercase keys)
	raw    map[string]string // Preserves original case of keys
}

// HeaderEntry represents a single header name-value pair
type HeaderEntry struct {
	Name  string
	Value string
}

// NewOrderedHeaders creates a new OrderedHeaders instance
func NewOrderedHeaders() *OrderedHeaders {
	return &OrderedHeaders{
		order:  make([]string, 0),
		values: make(map[string]string),
		raw:    make(map[string]string),
	}
}

// Set adds or updates a header, preserving order and case
func (h *OrderedHeaders) Set(name, value string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	lowerName := strings.ToLower(name)

	// If header doesn't exist, add to order
	if _, exists := h.values[lowerName]; !exists {
		h.order = append(h.order, lowerName)
	}

	h.values[lowerName] = value
	h.raw[lowerName] = name // Preserve original case
}

// SetAfter adds or updates a header, placing it after the specified header
func (h *OrderedHeaders) SetAfter(name, value, afterHeader string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	lowerName := strings.ToLower(name)
	afterLower := strings.ToLower(afterHeader)

	// If header exists, just update value
	if _, exists := h.values[lowerName]; exists {
		h.values[lowerName] = value
		h.raw[lowerName] = name
		return
	}

	// Find position after the specified header
	insertPos := len(h.order) // Default to end if not found
	for i, headerName := range h.order {
		if headerName == afterLower {
			insertPos = i + 1
			break
		}
	}

	// Insert at specific position
	h.order = append(h.order[:insertPos], append([]string{lowerName}, h.order[insertPos:]...)...)
	h.values[lowerName] = value
	h.raw[lowerName] = name
}

// SetBefore adds or updates a header, placing it before the specified header
func (h *OrderedHeaders) SetBefore(name, value, beforeHeader string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	lowerName := strings.ToLower(name)
	beforeLower := strings.ToLower(beforeHeader)

	// If header exists, just update value
	if _, exists := h.values[lowerName]; exists {
		h.values[lowerName] = value
		h.raw[lowerName] = name
		return
	}

	// Find position before the specified header
	insertPos := len(h.order) // Default to end if not found
	for i, headerName := range h.order {
		if headerName == beforeLower {
			insertPos = i
			break
		}
	}

	// Insert at specific position
	h.order = append(h.order[:insertPos], append([]string{lowerName}, h.order[insertPos:]...)...)
	h.values[lowerName] = value
	h.raw[lowerName] = name
}

// SetAt adds or updates a header at specific index position
func (h *OrderedHeaders) SetAt(name, value string, index int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	lowerName := strings.ToLower(name)

	// If header exists, just update value
	if _, exists := h.values[lowerName]; exists {
		h.values[lowerName] = value
		h.raw[lowerName] = name
		return
	}

	// Validate index
	if index < 0 || index > len(h.order) {
		index = len(h.order) // Default to end
	}

	// Insert at specific position
	h.order = append(h.order[:index], append([]string{lowerName}, h.order[index:]...)...)
	h.values[lowerName] = value
	h.raw[lowerName] = name
}

// Get retrieves a header value (case-insensitive)
func (h *OrderedHeaders) Get(name string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.values[strings.ToLower(name)]
}

// GetRaw retrieves the original case of the header name
func (h *OrderedHeaders) GetRaw(name string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.raw[strings.ToLower(name)]
}

// Has checks if a header exists (case-insensitive)
func (h *OrderedHeaders) Has(name string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	_, exists := h.values[strings.ToLower(name)]
	return exists
}

// Del removes first occurrence of a header
func (h *OrderedHeaders) Del(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	lowerName := strings.ToLower(name)

	if _, exists := h.values[lowerName]; exists {
		delete(h.values, lowerName)
		delete(h.raw, lowerName)

		// Remove from order
		for i, headerName := range h.order {
			if headerName == lowerName {
				h.order = append(h.order[:i], h.order[i+1:]...)
				break
			}
		}
	}
}

// DelAll removes all occurrences of a header
func (h *OrderedHeaders) DelAll(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	lowerName := strings.ToLower(name)

	// Remove from maps
	delete(h.values, lowerName)
	delete(h.raw, lowerName)

	// Remove all occurrences from order
	newOrder := make([]string, 0, len(h.order))
	for _, headerName := range h.order {
		if headerName != lowerName {
			newOrder = append(newOrder, headerName)
		}
	}
	h.order = newOrder
}

// Add adds a new header without replacing existing ones (for multi-value headers like Set-Cookie)
// Note: Since the internal storage uses a map, this works by appending to order but may override value
// For true multi-value support, consider using multiple Set() calls with unique keys
func (h *OrderedHeaders) Add(name, value string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	lowerName := strings.ToLower(name)

	// Always add to order (allows duplicates for headers like Set-Cookie)
	h.order = append(h.order, lowerName)
	h.values[lowerName] = value
	h.raw[lowerName] = name
}

// All returns all headers in their original order
func (h *OrderedHeaders) All() []Header {
	h.mu.RLock()
	defer h.mu.RUnlock()

	headers := make([]Header, 0, len(h.order))
	for _, lowerName := range h.order {
		headers = append(headers, Header{
			Name:  h.raw[lowerName],
			Value: h.values[lowerName],
		})
	}
	return headers
}

// Len returns the number of headers
func (h *OrderedHeaders) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.order)
}

// Header represents a single HTTP header
type Header struct {
	Name  string
	Value string
}
