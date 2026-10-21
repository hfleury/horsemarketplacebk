package common_test

import (
	"testing"

	"github.com/hfleury/horsemarketplacebk/internal/common"
	"github.com/stretchr/testify/assert"
)

func TestFormatPriceSEK(t *testing.T) {
	price := func(value float64) *float64 { return &value }

	tests := []struct {
		name  string
		price *float64
		want  string
	}{
		{name: "no price", price: nil, want: "Price on request"},
		{name: "whole kronor", price: price(99), want: "99 SEK"},
		{name: "with öre", price: price(49.5), want: "49.5 SEK"},
		{name: "zero", price: price(0), want: "0 SEK"},
		{name: "large", price: price(1250000), want: "1250000 SEK"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, common.FormatPriceSEK(tt.price))
		})
	}
}
