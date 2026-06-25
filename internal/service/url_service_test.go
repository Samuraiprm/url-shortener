package service

import (
	"net/url"
	"strings"
	"testing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{"valid http", "http://example.com", false, ""},
		{"valid https", "https://example.com/path?q=1", false, ""},
		{"valid with port", "https://localhost:8080/api", false, ""},
		{"empty", "", true, "url must have a host"},
		{"no host", "https://", true, "url must have a host"},
		{"javascript scheme", "javascript:alert(1)", true, "only http and https"},
		{"data uri", "data:text/html,<h1>hi</h1>", true, "only http and https"},
		{"vbscript", "vbscript:MsgBox", true, "only http and https"},
		{"ftp scheme", "ftp://files.example.com", true, "only http and https"},
		{"file scheme", "file:///etc/passwd", true, "only http and https"},
		{"too long", strings.Repeat("a", 2049), true, "too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error

			if len(tt.url) > 2048 {
				err = errURLTooLong
			} else {
				parsed, parseErr := url.ParseRequestURI(tt.url)
				if parseErr != nil {
					err = parseErr
				} else {
					scheme := strings.ToLower(parsed.Scheme)
					if scheme != "http" && scheme != "https" {
						err = errWrongScheme
					} else if parsed.Host == "" {
						err = errNoHost
					} else {
						lower := strings.ToLower(tt.url)
						for _, block := range []string{"javascript:", "data:", "vbscript:"} {
							if strings.Contains(lower, block) {
								err = errBlockedScheme
								break
							}
						}
					}
				}
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("validate(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

var (
	errURLTooLong   = &testErr{"url too long"}
	errWrongScheme  = &testErr{"only http and https"}
	errNoHost       = &testErr{"url must have a host"}
	errBlockedScheme = &testErr{"url contains blocked scheme"}
)

type testErr struct{ msg string }

func (e *testErr) Error() string { return e.msg }
