package request

import (
	"github.com/WhileEndless/go-httptools/pkg/search"
)

// Search searches in request headers and/or body
func (r *Request) Search(pattern string, opts search.SearchOptions) (*search.SearchResults, error) {
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
		bodyToSearch := r.Body

		// If chunked, decode first
		if r.IsBodyChunked {
			clone := r.Clone()
			clone.DecodeChunkedBody()
			bodyToSearch = clone.Body
		}

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

// SearchHeaders searches only in headers
func (r *Request) SearchHeaders(pattern string, caseInsensitive bool) (*search.SearchResults, error) {
	opts := search.DefaultOptions()
	opts.Location = search.SearchInHeaders
	opts.CaseInsensitive = caseInsensitive
	return r.Search(pattern, opts)
}

// SearchBody searches only in body
func (r *Request) SearchBody(pattern string, caseInsensitive bool) (*search.SearchResults, error) {
	opts := search.DefaultOptions()
	opts.Location = search.SearchInBody
	opts.CaseInsensitive = caseInsensitive
	return r.Search(pattern, opts)
}

// SearchRegex searches using regular expression
func (r *Request) SearchRegex(pattern string) (*search.SearchResults, error) {
	opts := search.DefaultOptions()
	opts.UseRegex = true
	return r.Search(pattern, opts)
}

// Contains checks if pattern exists anywhere in request
func (r *Request) Contains(pattern string, caseInsensitive bool) bool {
	// Quick check in headers first
	for _, h := range r.Headers.All() {
		if search.QuickSearch([]byte(h.Value), pattern, caseInsensitive) {
			return true
		}
		if search.QuickSearch([]byte(h.Name), pattern, caseInsensitive) {
			return true
		}
	}

	// Check body
	body := r.Body
	if r.IsBodyChunked {
		clone := r.Clone()
		clone.DecodeChunkedBody()
		body = clone.Body
	}

	return search.QuickSearch(body, pattern, caseInsensitive)
}

// ContainsRegex checks if regex pattern matches anywhere in request
func (r *Request) ContainsRegex(pattern string) (bool, error) {
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
	body := r.Body
	if r.IsBodyChunked {
		clone := r.Clone()
		clone.DecodeChunkedBody()
		body = clone.Body
	}

	return search.QuickSearchRegex(body, pattern)
}

// ReplaceInBody replaces all occurrences of pattern in body
func (r *Request) ReplaceInBody(pattern, replacement string, opts search.SearchOptions) (int, error) {
	newBody, count, err := search.ReplaceAll(r.Body, pattern, replacement, opts)
	if err != nil {
		return 0, err
	}
	r.Body = newBody
	return count, nil
}

// getHeaderFields converts headers to search.HeaderField slice
func (r *Request) getHeaderFields() []search.HeaderField {
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
