package client

import (
	"net/url"
	"testing"
)

func TestEncodeQueryParams(t *testing.T) {
	tests := []struct {
		name     string
		params   url.Values
		expected string
	}{
		{
			name: "Simple filter with spaces",
			params: url.Values{
				"$filter": []string{"Name eq 'Test Value'"},
			},
			expected: "%24filter=Name%20eq%20%27Test%20Value%27",
		},
		{
			name: "Multiple parameters with spaces",
			params: url.Values{
				"$filter": []string{"Category eq 'Test Category'"},
				"$select": []string{"ID, Name, Description"},
			},
			expected: "%24filter=Category%20eq%20%27Test%20Category%27&%24select=ID%2C%20Name%2C%20Description",
		},
		{
			name: "Special characters",
			params: url.Values{
				"$filter": []string{"Code eq '$TEST_CODE'"},
			},
			expected: "%24filter=Code%20eq%20%27%24TEST_CODE%27",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeQueryParams(tt.params)
			if result != tt.expected {
				t.Errorf("encodeQueryParams() = %v, want %v", result, tt.expected)
			}
		})
	}
}
