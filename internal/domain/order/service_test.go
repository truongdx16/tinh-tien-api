package order_test

import (
	"testing"

	"tinh-tien-api/internal/domain/order"
)

func TestCanTransition(t *testing.T) {
	tests := []struct {
		from, to order.Status
		want     bool
	}{
		{order.StatusDraft, order.StatusConfirmed, true},
		{order.StatusDraft, order.StatusCancelled, true},
		{order.StatusConfirmed, order.StatusPacked, true},
		{order.StatusConfirmed, order.StatusDelivered, true},
		{order.StatusPacked, order.StatusDelivered, true},
		// POS sales are created as delivered and must remain voidable.
		{order.StatusDelivered, order.StatusCancelled, true},
		{order.StatusDelivered, order.StatusConfirmed, false},
		{order.StatusCancelled, order.StatusConfirmed, false},
	}

	for _, tt := range tests {
		if got := order.CanTransition(tt.from, tt.to); got != tt.want {
			t.Errorf("canTransition(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}
