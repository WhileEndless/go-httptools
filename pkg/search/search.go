package search

import (
	"bytes"
	"regexp"
	"strings"
)

// SearchLocation specifies where to search
type SearchLocation int

const (
	SearchInHeaders SearchLocation = 1 << iota
	SearchInBody
	SearchInAll = SearchInHeaders | SearchInBody
)

// SearchOptions configures search behavior
type SearchOptions struct {
	// Pattern to search for
	Pattern string

	// UseRegex treats Pattern as a regular expression
	UseRegex bool

	// CaseInsensitive ignores case when matching
	CaseInsensitive bool

	// Location specifies where to search (headers, body, or both)
	Location SearchLocation

	// AutoDecompress decompresses body before searching (for compressed responses)
	AutoDecompress bool

	// SearchHeaderNames also search in header names, not just values
	SearchHeaderNames bool

	// SearchHeaderRaw search in raw/original header format (preserves whitespace)
	SearchHeaderRaw bool

	// MaxResults limits number of results (0 = unlimited)
	MaxResults int
}

// DefaultOptions returns default search options
func DefaultOptions() SearchOptions {
	return SearchOptions{
		Location:          SearchInAll,
		UseRegex:          false,
		CaseInsensitive:   false,
		AutoDecompress:    true,
		SearchHeaderNames: true,
		SearchHeaderRaw:   false,
		MaxResults:        0,
	}
}

// SearchResult represents a single search match
type SearchResult struct {
	// Location where match was found
	Location SearchLocation

	// HeaderName if found in headers (empty for body matches)
	HeaderName string

	// HeaderValue if found in headers
	HeaderValue string

	// MatchedText the actual text that matched
	MatchedText string

	// StartIndex position in the searched content
	StartIndex int

	// EndIndex end position in the searched content
	EndIndex int

	// LineNumber if applicable (1-indexed, 0 if not applicable)
	LineNumber int

	// Context surrounding text for context
	Context string
}

// SearchResults holds all search results
type SearchResults struct {
	// Query the original search pattern
	Query string

	// Options used for this search
	Options SearchOptions

	// Results all matches found
	Results []SearchResult

	// TotalMatches count (may be higher than len(Results) if MaxResults was set)
	TotalMatches int

	// HeaderMatches count of matches in headers
	HeaderMatches int

	// BodyMatches count of matches in body
	BodyMatches int
}

// HasMatches returns true if any matches were found
func (sr *SearchResults) HasMatches() bool {
	return sr.TotalMatches > 0
}

// Searcher provides search functionality
type Searcher struct {
	opts    SearchOptions
	regex   *regexp.Regexp
	pattern string
}

// NewSearcher creates a new searcher with the given options
func NewSearcher(opts SearchOptions) (*Searcher, error) {
	s := &Searcher{
		opts:    opts,
		pattern: opts.Pattern,
	}

	if opts.UseRegex {
		flags := ""
		if opts.CaseInsensitive {
			flags = "(?i)"
		}
		re, err := regexp.Compile(flags + opts.Pattern)
		if err != nil {
			return nil, err
		}
		s.regex = re
	} else if opts.CaseInsensitive {
		s.pattern = strings.ToLower(opts.Pattern)
	}

	return s, nil
}

// SearchBytes searches in raw bytes
func (s *Searcher) SearchBytes(data []byte) []SearchResult {
	var results []SearchResult

	if s.opts.UseRegex {
		matches := s.regex.FindAllIndex(data, -1)
		for _, match := range matches {
			if s.opts.MaxResults > 0 && len(results) >= s.opts.MaxResults {
				break
			}
			results = append(results, SearchResult{
				MatchedText: string(data[match[0]:match[1]]),
				StartIndex:  match[0],
				EndIndex:    match[1],
				LineNumber:  countLines(data[:match[0]]) + 1,
				Context:     extractContext(data, match[0], match[1], 50),
			})
		}
	} else {
		searchData := data
		searchPattern := []byte(s.pattern)
		if s.opts.CaseInsensitive {
			searchData = bytes.ToLower(data)
			searchPattern = []byte(strings.ToLower(s.pattern))
		}

		offset := 0
		for {
			if s.opts.MaxResults > 0 && len(results) >= s.opts.MaxResults {
				break
			}

			idx := bytes.Index(searchData[offset:], searchPattern)
			if idx == -1 {
				break
			}

			absIdx := offset + idx
			endIdx := absIdx + len(s.opts.Pattern)

			results = append(results, SearchResult{
				MatchedText: string(data[absIdx:endIdx]),
				StartIndex:  absIdx,
				EndIndex:    endIdx,
				LineNumber:  countLines(data[:absIdx]) + 1,
				Context:     extractContext(data, absIdx, endIdx, 50),
			})

			offset = absIdx + 1
		}
	}

	return results
}

// SearchString searches in a string
func (s *Searcher) SearchString(text string) []SearchResult {
	return s.SearchBytes([]byte(text))
}

// HeaderField represents a header for search purposes
type HeaderField struct {
	Name         string
	Value        string
	OriginalLine string
}

// SearchHeaders searches in headers
func (s *Searcher) SearchHeaders(headers []HeaderField) []SearchResult {
	var results []SearchResult

	for _, h := range headers {
		if s.opts.MaxResults > 0 && len(results) >= s.opts.MaxResults {
			break
		}

		// Search in header name if enabled
		if s.opts.SearchHeaderNames {
			nameMatches := s.SearchString(h.Name)
			for _, m := range nameMatches {
				if s.opts.MaxResults > 0 && len(results) >= s.opts.MaxResults {
					break
				}
				m.Location = SearchInHeaders
				m.HeaderName = h.Name
				m.HeaderValue = h.Value
				results = append(results, m)
			}
		}

		// Search in header value
		var valueToSearch string
		if s.opts.SearchHeaderRaw && h.OriginalLine != "" {
			// Search in original line format
			valueToSearch = h.OriginalLine
		} else {
			valueToSearch = h.Value
		}

		valueMatches := s.SearchString(valueToSearch)
		for _, m := range valueMatches {
			if s.opts.MaxResults > 0 && len(results) >= s.opts.MaxResults {
				break
			}
			m.Location = SearchInHeaders
			m.HeaderName = h.Name
			m.HeaderValue = h.Value
			results = append(results, m)
		}
	}

	return results
}

// SearchBody searches in body content
func (s *Searcher) SearchBody(body []byte) []SearchResult {
	results := s.SearchBytes(body)
	for i := range results {
		results[i].Location = SearchInBody
	}
	return results
}

// countLines counts newlines in data
func countLines(data []byte) int {
	count := 0
	for _, b := range data {
		if b == '\n' {
			count++
		}
	}
	return count
}

// extractContext extracts surrounding context
func extractContext(data []byte, start, end, contextSize int) string {
	ctxStart := start - contextSize
	if ctxStart < 0 {
		ctxStart = 0
	}

	ctxEnd := end + contextSize
	if ctxEnd > len(data) {
		ctxEnd = len(data)
	}

	return string(data[ctxStart:ctxEnd])
}

// QuickSearch is a convenience function for simple searches
func QuickSearch(data []byte, pattern string, caseInsensitive bool) bool {
	if caseInsensitive {
		return bytes.Contains(bytes.ToLower(data), []byte(strings.ToLower(pattern)))
	}
	return bytes.Contains(data, []byte(pattern))
}

// QuickSearchRegex is a convenience function for regex searches
func QuickSearchRegex(data []byte, pattern string) (bool, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.Match(data), nil
}

// FindAll finds all matches and returns them
func FindAll(data []byte, pattern string, opts SearchOptions) (*SearchResults, error) {
	opts.Pattern = pattern
	searcher, err := NewSearcher(opts)
	if err != nil {
		return nil, err
	}

	matches := searcher.SearchBytes(data)

	return &SearchResults{
		Query:        pattern,
		Options:      opts,
		Results:      matches,
		TotalMatches: len(matches),
		BodyMatches:  len(matches),
	}, nil
}

// ReplaceAll replaces all matches with replacement
func ReplaceAll(data []byte, pattern string, replacement string, opts SearchOptions) ([]byte, int, error) {
	opts.Pattern = pattern

	if opts.UseRegex {
		flags := ""
		if opts.CaseInsensitive {
			flags = "(?i)"
		}
		re, err := regexp.Compile(flags + pattern)
		if err != nil {
			return nil, 0, err
		}

		// Count matches first
		matches := re.FindAllIndex(data, -1)
		count := len(matches)

		result := re.ReplaceAll(data, []byte(replacement))
		return result, count, nil
	}

	// Non-regex replacement
	searchData := data
	searchPattern := []byte(pattern)
	if opts.CaseInsensitive {
		searchData = bytes.ToLower(data)
		searchPattern = []byte(strings.ToLower(pattern))
	}

	// Find all match positions
	var positions []int
	offset := 0
	for {
		idx := bytes.Index(searchData[offset:], searchPattern)
		if idx == -1 {
			break
		}
		positions = append(positions, offset+idx)
		offset = offset + idx + 1
	}

	if len(positions) == 0 {
		return data, 0, nil
	}

	// Build result with replacements
	var result bytes.Buffer
	lastEnd := 0
	for _, pos := range positions {
		result.Write(data[lastEnd:pos])
		result.WriteString(replacement)
		lastEnd = pos + len(pattern)
	}
	result.Write(data[lastEnd:])

	return result.Bytes(), len(positions), nil
}
