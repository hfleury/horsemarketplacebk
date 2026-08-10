package geocoding_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hfleury/horsemarketplacebk/internal/geocoding"
	"github.com/stretchr/testify/assert"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestMapboxClient(body string, statusCode int) *geocoding.MapboxClient {
	client := geocoding.NewMapboxClient("test-api-key")
	client.HTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	return client
}

func TestMapboxClient_Geocode_ParsesSuccessfulResponse(t *testing.T) {
	body := `{
		"features": [
			{"geometry": {"type": "Point", "coordinates": [18.0686, 59.3293]}}
		]
	}`
	client := newTestMapboxClient(body, http.StatusOK)

	coordinates, err := client.Geocode(context.Background(), "Stockholm, Sweden")

	assert.NoError(t, err)
	assert.Equal(t, 59.3293, coordinates.Lat)
	assert.Equal(t, 18.0686, coordinates.Lng)
}

func TestMapboxClient_Geocode_EmptyFeaturesReturnsNotFound(t *testing.T) {
	body := `{"features": []}`
	client := newTestMapboxClient(body, http.StatusOK)

	coordinates, err := client.Geocode(context.Background(), "Nowhereville")

	assert.Nil(t, coordinates)
	assert.True(t, errors.Is(err, geocoding.ErrLocationNotFound))
}
