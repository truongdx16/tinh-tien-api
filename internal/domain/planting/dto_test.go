package planting

import (
	"encoding/json"
	"testing"
)

// Body shape produced by Flutter PlantingSchedule.toJson (numbers for area/cost, nulls, ints).
const flutterBody = `{"vegetable_name":"Cải","planting_date":"2026-11-01","actual_harvest_date":null,
"area":120.5,"seed_cost":45000.0,"status":0,"is_harvested":false,"id":null,"created_at":null,
"spray_tasks":[{"spray_date":"2026-11-07T00:00:00.000","quarantine_days":7,"cost":30000.0,"quarantine_date":"2026-11-14"}],
"fertilize_tasks":[{"fertilize_date":"2026-11-05","amount":5,"application_number":"2"}]}`

func TestWriteRequestAcceptsFlutterPayload(t *testing.T) {
	var req WriteRequest
	if err := json.Unmarshal([]byte(flutterBody), &req); err != nil {
		t.Fatalf("decode: %v", err)
	}
	s := reqToModel(req)
	if s.Area != "120.5" || s.SeedCost != "45000" || s.ActualHarvestDate != "" {
		t.Errorf("schedule fields: %+v", s)
	}
	if len(s.SprayTasks) != 1 || s.SprayTasks[0].SprayDate != "2026-11-07" ||
		s.SprayTasks[0].QuarantineDays == nil || *s.SprayTasks[0].QuarantineDays != 7 ||
		s.SprayTasks[0].Cost != "30000" {
		t.Errorf("spray task: %+v", s.SprayTasks)
	}
	f := s.FertilizeTasks[0]
	if f.Amount != "5" || f.ApplicationNumber == nil || *f.ApplicationNumber != 2 {
		t.Errorf("fertilize task: %+v", f)
	}
}
