// Package statistics computes dashboard statistics from a snapshot of a
// user's apiaries, hives, and inspections. Every function here is pure -
// no context, no I/O - so the numbers this service reports can be
// unit-tested without mocking any of the upstream services they're
// ultimately sourced from.
package statistics

import (
	"time"

	"github.com/google/uuid"
)

// Apiary is the subset of apiary-service's Apiary this service needs.
type Apiary struct {
	ID   uuid.UUID
	Name string
}

// Hive is the subset of hive-service's Hive this service needs.
type Hive struct {
	ID       uuid.UUID
	ApiaryID uuid.UUID
	Name     string
}

// Inspection is the subset of inspection-service's Inspection this
// service needs.
type Inspection struct {
	ID          uuid.UUID
	HiveID      uuid.UUID
	InspectedAt time.Time
	Notes       string
}
