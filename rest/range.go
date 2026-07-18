package rest

import (
	"fmt"
	"regexp"
	"strconv"
)

var (
	rangeHeaderPattern        = regexp.MustCompile(`^bytes=([0-9]+)-([0-9]+)$`)
	contentRangeHeaderPattern = regexp.MustCompile(`^([0-9]+)-([0-9]+)$`)
)

// parseRangeHeader parses the `bytes=#-#` bytes header.
func parseRangeHeader(header string) (int64, int64, error) {
	m := rangeHeaderPattern.FindStringSubmatch(header)
	if m == nil {
		return 0, 0, fmt.Errorf("%w: %q", errRangeHeaderInvalid, header)
	}

	first, _ := strconv.ParseInt(m[1], 10, 64)
	last, _ := strconv.ParseInt(m[2], 10, 64)

	if first > last {
		return 0, 0, fmt.Errorf("%w: first(%d) > last(%d)", errRangeHeaderInvalid, first, last)
	}

	return first, last, nil
}

// parseContentRangeHeader parses the `#-#` Content-Range header based on OCI specifications.
func parseContentRangeHeader(header string) (int64, int64, error) {
	m := contentRangeHeaderPattern.FindStringSubmatch(header)
	if m == nil {
		return 0, 0, fmt.Errorf("%w: %q", errContentRangeHeaderInvalid, header)
	}

	first, _ := strconv.ParseInt(m[1], 10, 64)
	last, _ := strconv.ParseInt(m[2], 10, 64)

	if first > last {
		return 0, 0, fmt.Errorf("%w: first(%d) > last(%d)", errContentRangeHeaderInvalid, first, last)
	}

	return first, last, nil
}
