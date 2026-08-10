package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/db"
	"github.com/hfleury/horsemarketplacebk/internal/geocoding"
)

// backfillgeocode is a one-off, idempotent script that geocodes every
// catalog.products row missing latitude/longitude. Safe to re-run: it only
// ever selects rows where coordinates are still NULL.
func main() {
	ctx := context.Background()

	configService := config.NewVipperService()
	configService.LoadConfiguration()

	logger := config.NewZerologService()
	psqlDB, err := db.NewPsqlDB(configService.GetConfig(), *logger.Logger)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer psqlDB.Close()

	mapboxClient := geocoding.NewMapboxClient(configService.GetConfig().Mapbox.APIKey)

	rows, err := psqlDB.Query(ctx, `SELECT id, city, area FROM catalog.products WHERE latitude IS NULL OR longitude IS NULL`)
	if err != nil {
		log.Fatalf("failed to query products missing coordinates: %v", err)
	}

	type productLocation struct {
		id   string
		city *string
		area *string
	}

	var products []productLocation
	for rows.Next() {
		var p productLocation
		if err := rows.Scan(&p.id, &p.city, &p.area); err != nil {
			log.Printf("failed to scan product row: %v", err)
			continue
		}
		products = append(products, p)
	}
	rows.Close()

	var geocoded, skipped, failed int
	for _, p := range products {
		if p.city == nil || *p.city == "" {
			skipped++
			continue
		}

		query := *p.city + ", Sweden"
		if p.area != nil && *p.area != "" {
			query = *p.area + ", " + query
		}

		coordinates, err := mapboxClient.Geocode(ctx, query)
		if err != nil {
			log.Printf("failed to geocode product %s (query=%q): %v", p.id, query, err)
			failed++
			continue
		}

		if _, err := psqlDB.Execute(ctx, `UPDATE catalog.products SET latitude = $1, longitude = $2 WHERE id = $3`,
			coordinates.Lat, coordinates.Lng, p.id); err != nil {
			log.Printf("failed to update coordinates for product %s: %v", p.id, err)
			failed++
			continue
		}

		geocoded++
	}

	fmt.Printf("Backfill complete: %d geocoded, %d skipped (no city), %d failed\n", geocoded, skipped, failed)
}
