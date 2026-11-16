package mobile

import (
	"errors"

	"tinh-tien-api/internal/domain/customer"
	"tinh-tien-api/internal/domain/inventory"
	"tinh-tien-api/internal/domain/order"
	"tinh-tien-api/internal/domain/planting"
	"tinh-tien-api/internal/domain/product"
	"tinh-tien-api/internal/mobile/adapter"
)

// The Flutter app shows `message` from the response as-is, so every error
// returned to it must be a user-facing Vietnamese sentence. userMessage maps
// known domain errors; anything else gets the fallback (never err.Error(),
// which would leak internal details in English).
var domainMessages = []struct {
	err error
	msg string
}{
	{order.ErrEmptyItems, "Đơn hàng phải có ít nhất 1 sản phẩm"},
	{order.ErrInvalidStatus, "Không thể chuyển đơn hàng sang trạng thái này"},
	{order.ErrOrderClosed, "Đơn hàng đã hoàn thành hoặc đã huỷ, không thể sửa"},
	{order.ErrPaymentExceedsTotal, "Số tiền thanh toán vượt quá số tiền còn nợ"},
	{order.ErrNotFound, "Không tìm thấy đơn hàng"},
	{product.ErrNotFound, "Không tìm thấy sản phẩm"},
	{customer.ErrNotFound, "Không tìm thấy khách hàng"},
	{inventory.ErrInsufficient, "Không đủ hàng trong kho"},
	{planting.ErrNotFound, "Không tìm thấy lịch gieo trồng"},
	{planting.ErrVegetableNameRequired, "Vui lòng nhập tên giống cây"},
	{adapter.ErrInvalidDate, "Ngày không hợp lệ (định dạng YYYY-MM-DD)"},
}

func userMessage(err error, fallback string) string {
	for _, m := range domainMessages {
		if errors.Is(err, m.err) {
			return m.msg
		}
	}
	return fallback
}
