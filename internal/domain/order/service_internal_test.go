package order

import "testing"

func TestComputeTotals(t *testing.T) {
	ptr := func(v int64) *int64 { return &v }
	tests := []struct {
		name                              string
		subtotal, discount                int64
		paid                              *int64
		wantDiscount, wantTotal, wantPaid int64
	}{
		{"paid in full", 100, 10, nil, 10, 90, 90},
		{"partial payment", 100, 0, ptr(40), 0, 100, 40},
		{"discount clamped to subtotal", 100, 500, nil, 100, 0, 0},
		{"negative discount ignored", 100, -5, nil, 0, 100, 100},
		{"over-payment clamped", 100, 0, ptr(900), 0, 100, 100},
		{"negative payment clamped", 100, 0, ptr(-1), 0, 100, 0},
	}
	for _, tt := range tests {
		d, total, paid := computeTotals(tt.subtotal, tt.discount, tt.paid)
		if d != tt.wantDiscount || total != tt.wantTotal || paid != tt.wantPaid {
			t.Errorf("%s: got (%d,%d,%d), want (%d,%d,%d)", tt.name, d, total, paid, tt.wantDiscount, tt.wantTotal, tt.wantPaid)
		}
	}
}
