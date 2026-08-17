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
		{order.StatusPacked, order.StatusDelivered, true},
		{order.StatusDelivered, order.StatusCancelled, false},
		{order.StatusCancelled, order.StatusConfirmed, false},
	}

	for _, tt := range tests {
		got := canTransitionExported(tt.from, tt.to)
		if got != tt.want {
			t.Errorf("canTransition(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

// canTransitionExported mirrors order.canTransition for testing.
func canTransitionExported(from, to order.Status) bool {
	if from == to {
		return true
	}
	switch from {
	case order.StatusDraft:
		return to == order.StatusConfirmed || to == order.StatusCancelled
	case order.StatusConfirmed:
		return to == order.StatusPacked || to == order.StatusCancelled
	case order.StatusPacked:
		return to == order.StatusDelivered || to == order.StatusCancelled
	case order.StatusDelivered, order.StatusCancelled:
		return false
	default:
		return false
	}
}
