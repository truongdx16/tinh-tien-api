package migrations_test

import (
	"testing"

	"tinh-tien-api/internal/app/migrations"
)

func TestMigrationIDsAreUnique(t *testing.T) {
	all := migrations.All()
	seen := map[string]bool{}
	for _, m := range all {
		if m.ID == "" {
			t.Fatal("migration ID must not be empty")
		}
		if seen[m.ID] {
			t.Fatalf("duplicate migration ID: %s", m.ID)
		}
		seen[m.ID] = true
		if m.Up == nil {
			t.Fatalf("migration %s has nil Up", m.ID)
		}
	}
}
