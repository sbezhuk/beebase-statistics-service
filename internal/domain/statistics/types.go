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

// Harvest is the subset of harvest-service's Harvest this service needs.
// Product and Unit are carried as plain strings rather than
// harvest-service's own enum types, matching how every other type in this
// file only depends on the shape of the upstream data, never on the
// service that owns it.
type Harvest struct {
	ID          uuid.UUID
	Product     string
	Amount      float64
	Unit        string
	HarvestedAt time.Time
}
