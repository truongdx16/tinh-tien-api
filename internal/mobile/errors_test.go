package mobile

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"tinh-tien-api/internal/domain/inventory"
	"tinh-tien-api/internal/domain/order"
	"tinh-tien-api/internal/mobile/adapter"
)

func TestUserMessageMapsDomainErrors(t *testing.T) {
	wrapped := fmt.Errorf("deduct stock: %w", inventory.ErrInsufficient)
	_, _, dateErr := adapter.DayRange("07/11/2026", "", time.UTC)

	cases := []struct {
		err  error
		want string
	}{
		{order.ErrEmptyItems, "Đơn hàng phải có ít nhất 1 sản phẩm"},
		{order.ErrInvalidStatus, "Không thể chuyển đơn hàng sang trạng thái này"},
		{wrapped, "Không đủ hàng trong kho"},
		{dateErr, "Ngày không hợp lệ (định dạng YYYY-MM-DD)"},
		{errors.New("sql: connection refused"), "fallback"},
	}
	for _, c := range cases {
		if got := userMessage(c.err, "fallback"); got != c.want {
			t.Errorf("userMessage(%v) = %q, want %q", c.err, got, c.want)
		}
	}
}
