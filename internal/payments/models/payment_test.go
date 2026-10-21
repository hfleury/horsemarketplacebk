package models_test

import (
	"testing"

	"github.com/hfleury/horsemarketplacebk/internal/payments/models"
	"github.com/stretchr/testify/assert"
)

func TestSEKToOre(t *testing.T) {
	tests := []struct {
		name      string
		amountSEK float64
		want      int64
	}{
		{name: "zero", amountSEK: 0, want: 0},
		{name: "whole kronor", amountSEK: 3, want: 300},
		{name: "float just below its value", amountSEK: 99.99, want: 9999},
		{name: "sum of inexact floats", amountSEK: 0.1 + 0.2, want: 30},
		{name: "single öre", amountSEK: 0.01, want: 1},
		{name: "half öre rounds away from zero", amountSEK: 49.995, want: 5000},
		{name: "largest DECIMAL(12,2) value", amountSEK: 9999999999.99, want: 999999999999},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, models.SEKToOre(tt.amountSEK))
		})
	}
}
