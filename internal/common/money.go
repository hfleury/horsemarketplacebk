package common

import "strconv"

// FormatPriceSEK renders a price for emails; nil means the seller gave none.
func FormatPriceSEK(price *float64) string {
	if price == nil {
		return "Price on request"
	}
	return strconv.FormatFloat(*price, 'f', -1, 64) + " SEK"
}
