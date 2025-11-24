package response

import (
	"github.com/WhileEndless/go-httptools/pkg/chunked"
	"github.com/WhileEndless/go-httptools/pkg/compression"
	"github.com/WhileEndless/go-httptools/pkg/search"
)

// Search searches in response headers and/or body
func (r *Response) Search(pattern string, opts search.SearchOptions) (*search.SearchResults, error) {
	opts.Pattern = pattern
	searcher, err := search.NewSearcher(opts)
	if err != nil {
		return nil, err
	}

	results := &search.SearchResults{
		Query:   pattern,
		Options: opts,
		Results: []search.SearchResult{},
	}

	// Search in headers
	if opts.Location&search.SearchInHeaders != 0 {
		headerFields := r.getHeaderFields()
		headerMatches := searcher.SearchHeaders(headerFields)
		results.Results = append(results.Results, headerMatches...)
		results.HeaderMatches = len(headerMatches)
	}

	// Search in body
	if opts.Location&search.SearchInBody != 0 {
		bodyToSearch := r.getSearchableBody(opts.AutoDecompress)

		bodyMatches := searcher.SearchBody(bodyToSearch)

		// Apply MaxResults limit
		if opts.MaxResults > 0 && len(results.Results)+len(bodyMatches) > opts.MaxResults {
			remaining := opts.MaxResults - len(results.Results)
			if remaining > 0 {
				bodyMatches = bodyMatches[:remaining]
			} else {
				bodyMatches = nil
			}
		}

		results.Results = append(results.Results, bodyMatches...)
		results.BodyMatches = len(bodyMatches)
	}

	results.TotalMatches = len(results.Results)
	return results, nil
}

// getSearchableBody returns the body ready for searching
// Handles decompression and chunked decoding automatically
func (r *Response) getSearchableBody(autoDecompress bool) []byte {
	body := r.Body

	// If body is chunked, decode it first
	if r.IsBodyChunked {
		decodedBody, _ := chunked.Decode(body)
		body = decodedBody
	}

	// If body is still compressed and AutoDecompress is enabled
	if autoDecompress && r.Compressed && len(r.RawBody) > 0 {
		// Body should already be decompressed in r.Body
		// But if using RawBody, decompress it
		contentEncoding := r.GetContentEncoding()
		if contentEncoding != "" {
			compressionType := compression.DetectCompression(contentEncoding)
			if compressionType != compression.CompressionNone {
				decompressed, err := compression.Decompress(body, compressionType)
				if err == nil {
					body = decompressed
				}
			}
		}
	}

	return body
}

// SearchHeaders searches only in headers
func (r *Response) SearchHeaders(pattern string, caseInsensitive bool) (*search.SearchResults, error) {
	opts := search.DefaultOptions()
	opts.Location = search.SearchInHeaders
	opts.CaseInsensitive = caseInsensitive
	return r.Search(pattern, opts)
}

// SearchBody searches only in body
func (r *Response) SearchBody(pattern string, caseInsensitive bool) (*search.SearchResults, error) {
	opts := search.DefaultOptions()
	opts.Location = search.SearchInBody
	opts.CaseInsensitive = caseInsensitive
	return r.Search(pattern, opts)
}

// SearchRegex searches using regular expression
func (r *Response) SearchRegex(pattern string) (*search.SearchResults, error) {
	opts := search.DefaultOptions()
	opts.UseRegex = true
	return r.Search(pattern, opts)
}

// Contains checks if pattern exists anywhere in response
func (r *Response) Contains(pattern string, caseInsensitive bool) bool {
	// Quick check in headers first
	for _, h := range r.Headers.All() {
		if search.QuickSearch([]byte(h.Value), pattern, caseInsensitive) {
			return true
		}
		if search.QuickSearch([]byte(h.Name), pattern, caseInsensitive) {
			return true
		}
	}

	// Check body (decompressed/decoded)
	body := r.getSearchableBody(true)
	return search.QuickSearch(body, pattern, caseInsensitive)
}

// ContainsRegex checks if regex pattern matches anywhere in response
func (r *Response) ContainsRegex(pattern string) (bool, error) {
	// Check headers
	for _, h := range r.Headers.All() {
		match, err := search.QuickSearchRegex([]byte(h.Value), pattern)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
		match, err = search.QuickSearchRegex([]byte(h.Name), pattern)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
	}

	// Check body
	body := r.getSearchableBody(true)
	return search.QuickSearchRegex(body, pattern)
}

// ReplaceInBody replaces all occurrences of pattern in body
func (r *Response) ReplaceInBody(pattern, replacement string, opts search.SearchOptions) (int, error) {
	newBody, count, err := search.ReplaceAll(r.Body, pattern, replacement, opts)
	if err != nil {
		return 0, err
	}
	r.Body = newBody
	return count, nil
}

// getHeaderFields converts headers to search.HeaderField slice
func (r *Response) getHeaderFields() []search.HeaderField {
	all := r.Headers.All()
	fields := make([]search.HeaderField, len(all))
	for i, h := range all {
		fields[i] = search.HeaderField{
			Name:         h.Name,
			Value:        h.Value,
			OriginalLine: h.OriginalLine,
		}
	}
	return fields
}
