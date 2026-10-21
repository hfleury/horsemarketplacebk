package services

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCode(t *testing.T) {
	tests := map[string]struct {
		code  string
		valid bool
	}{
		"lowercase word":      {code: "boost", valid: true},
		"digits and symbols":  {code: "pro_2-x", valid: true},
		"two characters":      {code: "ab", valid: true},
		"twenty characters":   {code: "a" + strings.Repeat("b", 19), valid: true},
		"one character":       {code: "a"},
		"twenty-one":          {code: "a" + strings.Repeat("b", 20)},
		"uppercase":           {code: "Boost"},
		"leading digit":       {code: "1boost"},
		"leading underscore":  {code: "_boost"},
		"contains space":      {code: "big boost"},
		"empty":               {code: ""},
		"non-ascii character": {code: "bööst"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assertValidity(t, tt.valid, validateCode(tt.code))
		})
	}
}

func TestValidateName(t *testing.T) {
	tests := map[string]struct {
		name    string
		want    string
		invalid bool
	}{
		"plain":                 {name: "Boost", want: "Boost"},
		"trimmed":               {name: "  Pro plus  ", want: "Pro plus"},
		"fifty characters":      {name: strings.Repeat("a", 50), want: strings.Repeat("a", 50)},
		"fifty multibyte runes": {name: strings.Repeat("å", 50), want: strings.Repeat("å", 50)},
		"fifty-one characters":  {name: strings.Repeat("a", 51), invalid: true},
		"blank":                 {name: "   ", invalid: true},
		"empty":                 {name: "", invalid: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := validateName(tt.name)
			assertValidity(t, !tt.invalid, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidatePrice(t *testing.T) {
	tests := map[string]struct {
		price float64
		valid bool
	}{
		"whole":             {price: 49, valid: true},
		"one decimal":       {price: 49.5, valid: true},
		"two decimals":      {price: 49.99, valid: true},
		"smallest":          {price: 0.01, valid: true},
		"largest":           {price: 9_999_999_999.99, valid: true},
		"three decimals":    {price: 49.999},
		"zero":              {price: 0},
		"negative":          {price: -1},
		"too large":         {price: 1e10},
		"not a number":      {price: math.NaN()},
		"positive infinity": {price: math.Inf(1)},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assertValidity(t, tt.valid, validatePrice(tt.price))
		})
	}
}

func TestValidateDuration(t *testing.T) {
	tests := map[string]struct {
		days  int
		valid bool
	}{
		"one day":   {days: 1, valid: true},
		"one year":  {days: 365, valid: true},
		"zero":      {days: 0},
		"negative":  {days: -7},
		"over year": {days: 366},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assertValidity(t, tt.valid, validateDuration(tt.days))
		})
	}
}

func TestValidateTierFields_StopsAtFirstInvalidField(t *testing.T) {
	_, err := validateTierFields("", 0, 0)

	assert.ErrorContains(t, err, "name")
}

func assertValidity(t *testing.T, valid bool, err error) {
	t.Helper()
	if valid {
		assert.NoError(t, err)
		return
	}
	assert.True(t, errors.Is(err, ErrInvalidBoostTier), "expected ErrInvalidBoostTier, got %v", err)
}
