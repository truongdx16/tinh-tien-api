package notification

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"tinh-tien-api/internal/domain/planting"
	"tinh-tien-api/internal/pkg/model"
)

var ict = time.FixedZone("ICT", 7*3600)

func schedule(name, expected string) planting.PlantingSchedule {
	return planting.PlantingSchedule{
		Base:                model.Base{ID: uuid.New()},
		VegetableName:       name,
		ExpectedHarvestDate: expected,
		Location:            "Luống 2",
	}
}

func kinds(ns []Notification) []string {
	out := make([]string, 0, len(ns))
	for _, n := range ns {
		out = append(out, n.Kind)
	}
	return out
}

func TestDueRemindersHarvest(t *testing.T) {
	today := time.Date(2026, 11, 19, 8, 30, 0, 0, ict)
	cases := []struct {
		expected string
		want     []string
	}{
		{"2026-11-22", []string{KindHarvestSoon}},
		{"2026-11-21", nil}, // 2 days: only the 3-day reminder
		{"2026-11-19", []string{KindHarvestToday}},
		{"2026-11-18", []string{KindHarvestOverdue}}, // 1 day late
		{"2026-11-17", nil},                          // 2 days late: not a reminder day
		{"2026-11-16", []string{KindHarvestOverdue}}, // 3 days
		{"2026-11-12", []string{KindHarvestOverdue}}, // 7 days
		{"2026-11-19T00:00:00Z", []string{KindHarvestToday}},
		{"", nil},
		{"not a date", nil},
	}
	for _, c := range cases {
		got := kinds(DueReminders([]planting.PlantingSchedule{schedule("Cải ngọt", c.expected)}, today))
		if strings.Join(got, ",") != strings.Join(c.want, ",") {
			t.Errorf("expected %q: got %v, want %v", c.expected, got, c.want)
		}
	}
}

func TestDueRemindersSkipsHarvested(t *testing.T) {
	today := time.Date(2026, 11, 19, 8, 0, 0, 0, ict)
	done := schedule("Xà lách", "2026-11-19")
	done.IsHarvested = true
	recorded := schedule("Xà lách", "2026-11-19")
	recorded.ActualHarvestDate = "2026-11-18"
	if got := DueReminders([]planting.PlantingSchedule{done, recorded}, today); len(got) != 0 {
		t.Fatalf("harvested crops reminded: %v", kinds(got))
	}
}

func TestDueRemindersTasksAndQuarantine(t *testing.T) {
	today := time.Date(2026, 11, 19, 7, 0, 0, 0, ict)
	s := schedule("Cà chua", "2026-12-30")
	s.SprayTasks = []planting.SprayTask{
		{SprayDate: "2026-11-19", PesticideName: "Abamectin"},
		{SprayDate: "2026-11-10", PesticideName: "Regent", QuarantineDate: "2026-11-19"},
	}
	s.FertilizeTasks = []planting.FertilizeTask{{FertilizeDate: "2026-11-19", FertilizerName: "NPK"}}
	s.PlantingTasks = []planting.PlantingTask{
		{TaskDate: "2026-11-19", TaskName: "Làm cỏ"},
		{TaskDate: "2026-11-19", TaskName: "Tưới", Status: 1}, // done
		{TaskDate: "2026-11-20", TaskName: "Tỉa cành"},        // tomorrow
	}

	got := DueReminders([]planting.PlantingSchedule{s}, today)
	if strings.Join(kinds(got), ",") != KindTasksToday+","+KindQuarantineEnded {
		t.Fatalf("kinds: %v", kinds(got))
	}
	tasks := got[0]
	if tasks.Body != "Phun thuốc Abamectin · Bón phân NPK · Việc Làm cỏ (Luống 2)" {
		t.Errorf("tasks body: %q", tasks.Body)
	}
	if tasks.Endpoint != "planting-schedules/"+s.ID.String() {
		t.Errorf("endpoint: %q", tasks.Endpoint)
	}
	if !strings.Contains(got[1].Body, "Regent") {
		t.Errorf("quarantine body: %q", got[1].Body)
	}
}

func TestDueRemindersRefKeysAreStable(t *testing.T) {
	s := schedule("Cải ngọt", "2026-11-22")
	morning := DueReminders([]planting.PlantingSchedule{s}, time.Date(2026, 11, 19, 7, 0, 0, 0, ict))
	evening := DueReminders([]planting.PlantingSchedule{s}, time.Date(2026, 11, 19, 21, 0, 0, 0, ict))
	if morning[0].RefKey != evening[0].RefKey {
		t.Fatalf("same day, different keys: %q vs %q", morning[0].RefKey, evening[0].RefKey)
	}
	// Moving the harvest date gives a new reminder.
	s.ExpectedHarvestDate = "2026-11-23"
	later := DueReminders([]planting.PlantingSchedule{s}, time.Date(2026, 11, 20, 7, 0, 0, 0, ict))
	if later[0].RefKey == morning[0].RefKey {
		t.Fatal("new harvest date reused the old key")
	}
}
