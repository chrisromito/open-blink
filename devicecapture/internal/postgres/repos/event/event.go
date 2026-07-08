package event

import (
	"context"
	dEvent "devicecapture/internal/domain/event"
	"devicecapture/internal/postgres/db"
	"slices"
	"strings"
)

type PgDetectionEventRepo struct {
	queries *db.Queries
}

func NewPgDetectionEventRepo(q *db.Queries) *PgDetectionEventRepo {
	return &PgDetectionEventRepo{
		queries: q,
	}
}

// StartEvent implements [dEvent.DetectionEventRepo]
func (de *PgDetectionEventRepo) StartEvent(ctx context.Context, deviceID int64, labels []string) (dEvent.DetectionEvent, error) {
	dbLabels := LabelsToDb(labels)
	p := db.StartEventParams{
		DeviceID: deviceID,
		Labels:   dbLabels,
		State:    int(dEvent.Started),
	}
	record, err := de.queries.StartEvent(ctx, p)
	if err != nil {
		return dEvent.DetectionEvent{}, err
	}
	return de.dbToDomain(record), nil
}

// EndEvent implements [dEvent.DetectionEventRepo]
func (de *PgDetectionEventRepo) EndEvent(ctx context.Context, e dEvent.DetectionEvent, labels []string) (dEvent.DetectionEvent, error) {
	record, err := de.queries.EndEvent(ctx, e.ID)
	if err != nil {
		return dEvent.DetectionEvent{}, err
	}
	// Update the DB if the labels changed
	if !slicesEq(labels, e.Labels) {
		r, err2 := de.queries.UpdateEvent(ctx, db.UpdateEventParams{
			SetLabels: true,
			Labels:    LabelsToDb(labels),
			SetState:  true,
			State:     int32(dEvent.Ended),
			ID:        e.ID,
		})
		if err2 != nil {
			return dEvent.DetectionEvent{}, err2
		}
		return de.dbToDomain(r), nil
	}
	return de.dbToDomain(record), nil
}

var pageSize = int32(100)

// GetDeviceEvents implements [dEvent.DetectionEventRepo]
func (de *PgDetectionEventRepo) GetDeviceEvents(ctx context.Context, p dEvent.QueryParams) ([]dEvent.DetectionEvent, error) {
	page := int32(p.Page)
	if page < 1 {
		page = 1
	}
	params := db.GetEventsParams{
		DeviceID: p.DeviceID,
		Lim:      pageSize,
		Off:      (page - 1) * pageSize,
	}
	if p.State != 0 {
		params.State = int32(p.State)
	}
	rows, err := de.queries.GetEvents(ctx, params)
	if err != nil {
		return []dEvent.DetectionEvent{}, err
	}
	var temp []dEvent.DetectionEvent
	for _, row := range rows {
		temp = append(temp, de.dbToDomain(row))
	}
	return temp, nil
}

// dbToDomain converts a [db.DetectionEvent] to a [dEvent.DetectionEvent]
func (de *PgDetectionEventRepo) dbToDomain(evt db.DetectionEvent) dEvent.DetectionEvent {
	return dEvent.DetectionEvent{
		ID:        evt.ID,
		DeviceID:  evt.DeviceID,
		CreatedAt: evt.CreatedAt,
		EndedAt:   evt.EndedAt,
		State:     evt.State,
		Labels:    DbToLabels(evt.Labels),
	}
}

// LabelsToDb converts a slice of strings to a sorted, comma-separated string of values
func LabelsToDb(labels []string) string {
	slices.Sort(labels)
	var out []string
	for _, label := range labels {
		if !slices.Contains(out, label) {
			out = append(out, label)
		}
	}
	return strings.Join(out, ", ")
}

func DbToLabels(label string) []string {
	return strings.Split(label, ", ")
}

func slicesEq(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for _, l := range left {
		if !slices.Contains(right, l) {
			return false
		}
	}
	for _, r := range right {
		if !slices.Contains(left, r) {
			return false
		}
	}
	return true
}
