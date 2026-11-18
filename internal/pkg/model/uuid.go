package model

import (
	"strings"

	"github.com/google/uuid"
)

// OptionalUUID turns a possibly empty id into a value for a nullable uuid
// column: nil when s is empty or not a uuid, otherwise the canonical form.
func OptionalUUID(s string) *string {
	id, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil || id == uuid.Nil {
		return nil
	}
	v := id.String()
	return &v
}
