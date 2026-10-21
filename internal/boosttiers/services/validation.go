package services

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	maxNameLength   = 50
	maxDurationDays = 365
	// DECIMAL(12,2) holds values below 10^10.
	maxPriceSEK = 1e10
)

var codePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,19}$`)

func validateCode(code string) error {
	if !codePattern.MatchString(code) {
		return fmt.Errorf("%w: code must be 2-20 lowercase letters, digits, '-' or '_', starting with a letter", ErrInvalidBoostTier)
	}
	return nil
}

// validateName returns the trimmed name.
func validateName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || utf8.RuneCountInString(trimmed) > maxNameLength {
		return "", fmt.Errorf("%w: name must be between 1 and %d characters", ErrInvalidBoostTier, maxNameLength)
	}
	return trimmed, nil
}

func validatePrice(priceSEK float64) error {
	if math.IsNaN(priceSEK) || priceSEK <= 0 || priceSEK >= maxPriceSEK {
		return fmt.Errorf("%w: price_sek must be greater than 0", ErrInvalidBoostTier)
	}
	if hundredths := priceSEK * 100; math.Abs(hundredths-math.Round(hundredths)) > 1e-6 {
		return fmt.Errorf("%w: price_sek must have at most 2 decimals", ErrInvalidBoostTier)
	}
	return nil
}

func validateDuration(durationDays int) error {
	if durationDays < 1 || durationDays > maxDurationDays {
		return fmt.Errorf("%w: duration_days must be between 1 and %d", ErrInvalidBoostTier, maxDurationDays)
	}
	return nil
}

// validateTierFields checks the fields shared by create and update, and
// returns the trimmed name.
func validateTierFields(name string, priceSEK float64, durationDays int) (string, error) {
	trimmedName, err := validateName(name)
	if err != nil {
		return "", err
	}
	if err := validatePrice(priceSEK); err != nil {
		return "", err
	}
	if err := validateDuration(durationDays); err != nil {
		return "", err
	}
	return trimmedName, nil
}
