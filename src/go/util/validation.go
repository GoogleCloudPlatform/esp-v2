package util

import (
	"fmt"
	"strings"
)

const (
	httpHeaderName  = "^:?[0-9a-zA-Z!#$%&'*+-.^_|~\x60]+$"
	httpHeaderValue = "^[^\u0000-\u0008\u000A-\u001F\u007F]*$"
	// For non-strict validation.
	headerString = "^[^\u0000\u000A\u000D]*$"
)

func HasOnlyUrlChars(url string) error {
	return hasOnlyUrlCharsHelper(url)
}

func HasOnlyUrlCharsWithNoQuery(url string) error {
	if err := hasOnlyUrlCharsHelper(url); err != nil {
		return err
	}

	if strings.Contains(url, "?") ||
		strings.Contains(url, "&") ||
		strings.Contains(url, "#") {
		return fmt.Errorf("URL (%v) failed validation: contains query parameters or fragements", url)
	}

	return nil
}

func hasOnlyUrlCharsHelper(url string) error {
	_, _, _, _, err := ParseURI(url)
	if err != nil {
		return fmt.Errorf("URL (%v) failed validation: %v", url, err)
	}
	return nil
}
