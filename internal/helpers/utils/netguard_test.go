package utils

import "testing"

func TestValidateWebhookURL(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"public https host", "https://example.com/hooks", false},
		{"http scheme rejected", "http://example.com/hooks", true},
		{"loopback literal rejected", "https://127.0.0.1/hooks", true},
		{"loopback name rejected", "https://localhost/hooks", true},
		{"private range rejected", "https://10.0.0.5/hooks", true},
		{"private range 192 rejected", "https://192.168.1.10/hooks", true},
		{"cloud metadata rejected", "https://169.254.169.254/latest/meta-data", true},
		{"unspecified rejected", "https://0.0.0.0/hooks", true},
		{"missing host rejected", "https:///hooks", true},
		{"garbage rejected", "not-a-url", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateWebhookURL(tc.url)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for %q, got nil", tc.url)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.url, err)
			}
		})
	}
}
