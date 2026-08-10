package models

// HorseFilter holds optional structured filter criteria for horse listings.
// Discipline is the public/API name; it maps to the product_horses.orientation
// column in the repository layer.
type HorseFilter struct {
	Breed      *string
	Gender     *string
	Discipline *string
	MinAge     *int
	MaxAge     *int
	MinHeight  *int
	MaxHeight  *int
	MinPrice   *float64
	MaxPrice   *float64
}
