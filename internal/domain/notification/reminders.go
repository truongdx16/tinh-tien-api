package notification

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"tinh-tien-api/internal/domain/planting"
)

// HarvestSoonDays: how many days before the expected harvest the first
// reminder goes out.
const HarvestSoonDays = 3

// overdueReminderDays: days after the expected harvest when a not-yet
// harvested crop is reminded again.
var overdueReminderDays = map[int]bool{1: true, 3: true, 7: true}

const dateLayout = "2006-01-02"

// parseDate reads a stored date ("2026-11-19", or a timestamp starting with
// it). ok is false when empty or unreadable.
func parseDate(s string, loc *time.Location) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if len(s) < len(dateLayout) {
		return time.Time{}, false
	}
	d, err := time.ParseInLocation(dateLayout, s[:len(dateLayout)], loc)
	return d, err == nil
}

func dayMonth(t time.Time) string { return t.Format("02/01") }

func scheduleEndpoint(s planting.PlantingSchedule) string {
	return "planting-schedules/" + s.ID.String()
}

// DueReminders lists the notifications due on today (a date in the shop's
// timezone) for schedules: harvest soon / today / overdue, tasks of the day
// and spray quarantines ending. Harvested crops get nothing. Each has a
// RefKey unique to the event, so running this many times a day is safe.
func DueReminders(schedules []planting.PlantingSchedule, today time.Time) []Notification {
	loc := today.Location()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)
	todayKey := today.Format(dateLayout)
	var out []Notification

	for _, s := range schedules {
		if s.IsHarvested || strings.TrimSpace(s.ActualHarvestDate) != "" {
			continue
		}
		id := s.ID.String()
		name := strings.TrimSpace(s.VegetableName)
		endpoint := scheduleEndpoint(s)
		where := ""
		if loc := strings.TrimSpace(s.Location); loc != "" {
			where = " (" + loc + ")"
		}

		if expected, ok := parseDate(s.ExpectedHarvestDate, loc); ok {
			days := int(expected.Sub(today).Hours() / 24)
			expectedKey := expected.Format(dateLayout)
			switch {
			case days == HarvestSoonDays:
				out = append(out, Notification{
					Kind:     KindHarvestSoon,
					Title:    "Sắp thu hoạch: " + name,
					Body:     fmt.Sprintf("Còn %d ngày nữa (%s) đến ngày thu hoạch dự kiến%s.", days, dayMonth(expected), where),
					Endpoint: endpoint,
					RefKey:   fmt.Sprintf("%s:%s:%s", KindHarvestSoon, id, expectedKey),
				})
			case days == 0:
				out = append(out, Notification{
					Kind:     KindHarvestToday,
					Title:    "Đến ngày thu hoạch: " + name,
					Body:     "Hôm nay là ngày thu hoạch dự kiến" + where + ". Nhớ cập nhật sản lượng sau khi thu hoạch.",
					Endpoint: endpoint,
					RefKey:   fmt.Sprintf("%s:%s:%s", KindHarvestToday, id, expectedKey),
				})
			case days < 0 && overdueReminderDays[-days]:
				out = append(out, Notification{
					Kind:     KindHarvestOverdue,
					Title:    "Quá hạn thu hoạch: " + name,
					Body:     fmt.Sprintf("Đã quá %d ngày so với ngày dự kiến (%s)%s. Thu hoạch hoặc cập nhật lại lịch.", -days, dayMonth(expected), where),
					Endpoint: endpoint,
					RefKey:   fmt.Sprintf("%s:%s:%s:%d", KindHarvestOverdue, id, expectedKey, -days),
				})
			}
		}

		var tasks []string
		var quarantines []string
		for _, t := range s.SprayTasks {
			if d, ok := parseDate(t.SprayDate, loc); ok && d.Equal(today) {
				tasks = append(tasks, labelled("Phun thuốc", t.PesticideName))
			}
			if d, ok := parseDate(t.QuarantineDate, loc); ok && d.Equal(today) {
				quarantines = append(quarantines, strings.TrimSpace(t.PesticideName))
			}
		}
		for _, t := range s.FertilizeTasks {
			if d, ok := parseDate(t.FertilizeDate, loc); ok && d.Equal(today) {
				tasks = append(tasks, labelled("Bón phân", t.FertilizerName))
			}
		}
		for _, t := range s.PlantingTasks {
			if t.Status != 0 {
				continue // done
			}
			if d, ok := parseDate(t.TaskDate, loc); ok && d.Equal(today) {
				tasks = append(tasks, labelled("Việc", t.TaskName))
			}
		}
		if len(tasks) > 0 {
			out = append(out, Notification{
				Kind:     KindTasksToday,
				Title:    "Việc hôm nay: " + name,
				Body:     strings.Join(tasks, " · ") + where,
				Endpoint: endpoint,
				RefKey:   fmt.Sprintf("%s:%s:%s", KindTasksToday, id, todayKey),
			})
		}
		if len(quarantines) > 0 {
			what := "thuốc"
			if names := strings.Join(nonEmpty(quarantines), ", "); names != "" {
				what = "thuốc " + names
			}
			out = append(out, Notification{
				Kind:     KindQuarantineEnded,
				Title:    "Hết thời gian cách ly: " + name,
				Body:     "Đã hết thời gian cách ly " + what + where + ", có thể thu hoạch an toàn.",
				Endpoint: endpoint,
				RefKey:   fmt.Sprintf("%s:%s:%s", KindQuarantineEnded, id, todayKey),
			})
		}
	}
	return out
}

func labelled(kind, name string) string {
	if name = strings.TrimSpace(name); name == "" {
		return kind
	}
	return kind + " " + name
}

func nonEmpty(items []string) []string {
	out := items[:0:0]
	for _, s := range items {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// ScheduleLister is what the scheduler needs from planting schedules.
type ScheduleLister interface {
	List() ([]planting.PlantingSchedule, error)
}

// Scheduler sends the day's reminders once the shop's clock reaches Hour.
// It checks every Interval; RefKeys keep each reminder to one send, so a
// restart, or a schedule added later in the day, is handled by the next
// check.
type Scheduler struct {
	Service   *Service
	Schedules ScheduleLister
	Location  *time.Location
	Hour      int
	Interval  time.Duration
	Now       func() time.Time
}

// RunOnce sends what is due now. Returns how many notifications were new.
func (sc *Scheduler) RunOnce(ctx context.Context) (int, error) {
	now := sc.now().In(sc.Location)
	if now.Hour() < sc.Hour {
		return 0, nil
	}
	schedules, err := sc.Schedules.List()
	if err != nil {
		return 0, fmt.Errorf("list schedules: %w", err)
	}
	created := 0
	for _, n := range DueReminders(schedules, now) {
		isNew, err := sc.Service.Notify(ctx, n)
		if isNew {
			created++
		}
		if err != nil {
			log.Printf("notification %s: %v", n.RefKey, err)
		}
	}
	return created, nil
}

// Run checks until ctx is cancelled.
func (sc *Scheduler) Run(ctx context.Context) {
	interval := sc.Interval
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	check := func() {
		if n, err := sc.RunOnce(ctx); err != nil {
			log.Printf("reminders: %v", err)
		} else if n > 0 {
			log.Printf("reminders: %d notification(s) sent", n)
		}
	}
	check()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check()
		}
	}
}

func (sc *Scheduler) now() time.Time {
	if sc.Now != nil {
		return sc.Now()
	}
	return time.Now()
}
