package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const redactedValue = "[REDACTED]"

var sensitiveKeyFragments = []string{"password", "token", "secret", "authorization", "cookie"}

func isSensitiveKey(key string) bool {
	lowerKey := strings.ToLower(key)
	for _, fragment := range sensitiveKeyFragments {
		if strings.Contains(lowerKey, fragment) {
			return true
		}
	}
	return false
}

// RedactRequestBody returns a JSON body with every sensitive value replaced, so it
// can be logged. A body that isn't a single JSON value can't be redacted reliably,
// so only its size is kept.
func RedactRequestBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}

	parsed, err := decodeSingleJSONValue(body)
	if err != nil {
		return fmt.Sprintf("[unparseable body omitted, %d bytes]", len(body))
	}

	var redacted bytes.Buffer
	encoder := json.NewEncoder(&redacted)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(redactValue(parsed)); err != nil {
		return fmt.Sprintf("[unparseable body omitted, %d bytes]", len(body))
	}
	return strings.TrimSuffix(redacted.String(), "\n")
}

func decodeSingleJSONValue(body []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	var parsed any
	if err := decoder.Decode(&parsed); err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing data after JSON value")
	}
	return parsed, nil
}

func redactValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if isSensitiveKey(key) {
				typed[key] = redactedValue
			} else {
				typed[key] = redactValue(nested)
			}
		}
		return typed
	case []any:
		for index, item := range typed {
			typed[index] = redactValue(item)
		}
		return typed
	default:
		return value
	}
}

// redactFields returns a copy of fields with sensitive top-level values replaced,
// leaving the caller's map untouched.
func redactFields(fields map[string]any) map[string]any {
	if fields == nil {
		return nil
	}
	redacted := make(map[string]any, len(fields))
	for key, value := range fields {
		if isSensitiveKey(key) {
			redacted[key] = redactedValue
		} else {
			redacted[key] = value
		}
	}
	return redacted
}
