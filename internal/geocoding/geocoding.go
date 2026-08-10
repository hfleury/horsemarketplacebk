package geocoding

import (
	"context"
	"errors"
)

var ErrLocationNotFound = errors.New("location not found")

type Coordinates struct {
	Lat float64
	Lng float64
}

type Client interface {
	Geocode(ctx context.Context, query string) (*Coordinates, error)
}
