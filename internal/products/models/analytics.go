package models

// AnalyticsProperties returns the non-sensitive listing fields attached to
// listing-related analytics events.
func (p *Product) AnalyticsProperties() map[string]any {
	properties := map[string]any{
		"listing_id":   p.ID.String(),
		"listing_type": string(p.Type),
		"status":       string(p.Status),
	}
	if p.CategoryID != nil {
		properties["category_id"] = p.CategoryID.String()
	}
	return properties
}
