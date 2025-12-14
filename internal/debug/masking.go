// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package debug

import (
	"net/url"
	"strings"
)

// SensitiveKeys contains keys that trigger automatic masking when detected
var SensitiveKeys = []string{
	"password", "passwd", "pwd", "secret",
	"token", "api_key", "apikey", "api-key",
	"authorization", "auth", "credential",
	"x-csrf-token", "csrf",
}

// MaskPassword completely masks a password, returning "***"
func MaskPassword(password string) string {
	if len(password) == 0 {
		return ""
	}
	return "***"
}

// MaskToken masks a token, showing only the last 8 characters
// For tokens shorter than 8 characters, returns "****"
func MaskToken(token string) string {
	if len(token) == 0 {
		return ""
	}
	if len(token) <= 8 {
		return "****"
	}
	return "****" + token[len(token)-8:]
}

// MaskValue masks a sensitive value, showing only the last N characters
func MaskValue(value string, showLastChars int) string {
	if len(value) == 0 {
		return ""
	}
	if len(value) <= showLastChars {
		return strings.Repeat("*", len(value))
	}
	return strings.Repeat("*", len(value)-showLastChars) + value[len(value)-showLastChars:]
}

// MaskURL removes sensitive information from a URL
// - Masks password in userinfo (user:password@host)
// - Masks sensitive query parameters
func MaskURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// Mask userinfo password if present
	if parsed.User != nil {
		if _, hasPass := parsed.User.Password(); hasPass {
			parsed.User = url.UserPassword(parsed.User.Username(), "***")
		}
	}

	// Mask sensitive query parameters
	query := parsed.Query()
	modified := false
	for key := range query {
		if IsSensitiveKey(key) {
			query.Set(key, "***")
			modified = true
		}
	}
	if modified {
		parsed.RawQuery = query.Encode()
	}

	return parsed.String()
}

// MaskHeader masks sensitive HTTP header values
// - Authorization headers show type but mask the credential
// - Other sensitive headers are masked using MaskToken
func MaskHeader(name, value string) string {
	if len(value) == 0 {
		return ""
	}

	nameLower := strings.ToLower(name)

	// Special handling for Authorization header
	if nameLower == "authorization" {
		parts := strings.SplitN(value, " ", 2)
		if len(parts) == 2 {
			// Preserve auth type (Basic, Bearer, etc.) but mask the credential
			return parts[0] + " " + MaskToken(parts[1])
		}
		return MaskToken(value)
	}

	// Mask other sensitive headers
	if IsSensitiveKey(nameLower) {
		return MaskToken(value)
	}

	return value
}

// IsSensitiveKey checks if a key name indicates sensitive data
func IsSensitiveKey(key string) bool {
	keyLower := strings.ToLower(key)
	for _, sensitive := range SensitiveKeys {
		if strings.Contains(keyLower, sensitive) {
			return true
		}
	}
	return false
}
