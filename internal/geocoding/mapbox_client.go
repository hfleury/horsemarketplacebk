package geocoding

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const mapboxForwardGeocodeURL = "https://api.mapbox.com/search/geocode/v6/forward"

type MapboxClient struct {
	APIKey     string
	HTTPClient *http.Client
}

func NewMapboxClient(apiKey string) *MapboxClient {
	return &MapboxClient{
		APIKey:     apiKey,
		HTTPClient: &http.Client{},
	}
}

type mapboxForwardGeocodeResponse struct {
	Features []struct {
		Geometry struct {
			Coordinates [2]float64 `json:"coordinates"`
		} `json:"geometry"`
	} `json:"features"`
}

func (m *MapboxClient) Geocode(ctx context.Context, query string) (*Coordinates, error) {
	reqURL := fmt.Sprintf("%s?q=%s&country=SE&access_token=%s",
		mapboxForwardGeocodeURL, url.QueryEscape(query), url.QueryEscape(m.APIKey))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mapbox geocoding request failed with status %d", resp.StatusCode)
	}

	var result mapboxForwardGeocodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Features) == 0 {
		return nil, ErrLocationNotFound
	}

	coordinates := result.Features[0].Geometry.Coordinates
	return &Coordinates{Lat: coordinates[1], Lng: coordinates[0]}, nil
}
