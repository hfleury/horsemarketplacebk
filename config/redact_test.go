package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactRequestBody(t *testing.T) {
	const unparseable = "<unparseable placeholder>"

	tests := []struct {
		name string
		body string
		want string
	}{
		{"empty body", ``, ``},
		{"top-level password", `{"username":"bob","password":"hunter2!"}`, `{"password":"[REDACTED]","username":"bob"}`},
		{"nested key is case-insensitive", `{"user":{"Password":"x","name":"n"}}`, `{"user":{"Password":"[REDACTED]","name":"n"}}`},
		{"objects inside arrays", `[{"refresh_token":"r"},{"id":1}]`, `[{"refresh_token":"[REDACTED]"},{"id":1}]`},
		{"token and secret variants", `{"access_token":"a","client_secret":"s","Authorization":"Bearer x","session_cookie":"c"}`, `{"Authorization":"[REDACTED]","access_token":"[REDACTED]","client_secret":"[REDACTED]","session_cookie":"[REDACTED]"}`},
		{"non-sensitive values kept exactly", `{"price":12345678901234567890,"note":"<b>&</b>"}`, `{"note":"<b>&</b>","price":12345678901234567890}`},
		{"malformed JSON", `{"username":"bob","password":"hun`, unparseable},
		{"trailing data after JSON", `{"username":"bob"} {"password":"x"}`, unparseable},
		{"multipart body", "--boundary\r\nContent-Disposition: form-data; name=\"file\"\r\n\r\nbinary\r\n--boundary--", unparseable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := tt.want
			if want == unparseable {
				want = fmt.Sprintf("[unparseable body omitted, %d bytes]", len(tt.body))
			}
			assert.Equal(t, want, RedactRequestBody([]byte(tt.body)))
		})
	}
}

func TestRedactFields(t *testing.T) {
	fields := map[string]any{"verification_token": "t", "email": "e@example.com", "count": 3}

	redacted := redactFields(fields)

	assert.Equal(t, map[string]any{"verification_token": redactedValue, "email": "e@example.com", "count": 3}, redacted)
	assert.Equal(t, "t", fields["verification_token"], "caller's map must not be mutated")
	assert.Nil(t, redactFields(nil))
	assert.Empty(t, redactFields(map[string]any{}))
}
