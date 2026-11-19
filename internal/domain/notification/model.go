package notification

import (
	"time"

	"tinh-tien-api/internal/pkg/model"
)

// Kinds of notifications (also sent to the app as data.kind).
const (
	KindHarvestSoon     = "harvest_soon"     // expected harvest in a few days
	KindHarvestToday    = "harvest_today"    // expected harvest today
	KindHarvestOverdue  = "harvest_overdue"  // expected harvest passed, not harvested
	KindTasksToday      = "tasks_today"      // spray / fertilize / other tasks due today
	KindQuarantineEnded = "quarantine_ended" // spray quarantine ends today: safe to harvest
	KindTest            = "test"
)

// DeviceToken is an FCM registration token of a signed-in device. One
// device = one row; signing in with another account moves the row.
type DeviceToken struct {
	model.Base
	UserID     string    `gorm:"type:uuid;index;not null" json:"user_id"`
	Token      string    `gorm:"size:255;uniqueIndex;not null" json:"token"`
	Platform   string    `gorm:"size:16" json:"platform"` // android | ios
	LastSeenAt time.Time `json:"last_seen_at"`
}

// Notification is one message shown to the shop (the store is shared by
// every account, so every account receives it). RefKey makes sending
// idempotent: the same reminder for the same day is stored and pushed once,
// whatever the number of scheduler runs or restarts.
type Notification struct {
	model.Base
	Kind  string `gorm:"size:32;index;not null" json:"kind"`
	Title string `gorm:"size:255;not null" json:"title"`
	Body  string `gorm:"size:1024" json:"body"`
	// Endpoint the app opens on tap, e.g. "planting-schedules/<id>".
	Endpoint string `gorm:"size:255" json:"endpoint"`
	RefKey   string `gorm:"size:191;uniqueIndex;not null" json:"-"`
	// Devices it was pushed to (0 when push is not configured).
	SentCount int `gorm:"not null;default:0" json:"-"`
}
