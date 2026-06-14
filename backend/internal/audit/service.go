package audit

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	ID         uuid.UUID      `json:"id"`
	TenantID   uuid.UUID      `json:"tenantId"`
	ActorID    *uuid.UUID     `json:"actorId,omitempty"`
	ActorEmail string         `json:"actorEmail,omitempty"`
	Action     string         `json:"action"`
	Resource   string         `json:"resource"`
	ResourceID string         `json:"resourceId,omitempty"`
	Outcome    string         `json:"outcome"` // success | failure | denied
	IPAddress  string         `json:"ipAddress,omitempty"`
	UserAgent  string         `json:"userAgent,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
	OccurredAt time.Time      `json:"occurredAt"`
}

type ListFilter struct {
	ActorID  *uuid.UUID
	Action   string
	Resource string
	Outcome  string
	Since    *time.Time
	Until    *time.Time
	Cursor   string
	Limit    int
}

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

// Write appends an immutable audit event. The DB rules prevent UPDATE/DELETE.
func (s *Service) Write(ctx context.Context, e *Event) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	}

	detailsJSON, _ := json.Marshal(e.Details)

	_, err := s.db.Exec(ctx,
		`INSERT INTO audit_events
		 (id, tenant_id, actor_id, actor_email, action, resource, resource_id, outcome, ip_address, user_agent, details, occurred_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		e.ID, e.TenantID, e.ActorID, e.ActorEmail, e.Action, e.Resource, e.ResourceID,
		e.Outcome, e.IPAddress, e.UserAgent, detailsJSON, e.OccurredAt,
	)
	return err
}

// List queries audit events with cursor-based pagination and optional filters.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, f ListFilter) ([]*Event, string, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}

	q := `SELECT id, tenant_id, actor_id, actor_email, action, resource, resource_id, outcome, ip_address, user_agent, details, occurred_at
	      FROM audit_events WHERE tenant_id = $1`
	args := []any{tenantID}
	n := 2

	if f.ActorID != nil {
		q += fmt.Sprintf(" AND actor_id = $%d", n)
		args = append(args, *f.ActorID)
		n++
	}
	if f.Action != "" {
		q += fmt.Sprintf(" AND action = $%d", n)
		args = append(args, f.Action)
		n++
	}
	if f.Resource != "" {
		q += fmt.Sprintf(" AND resource = $%d", n)
		args = append(args, f.Resource)
		n++
	}
	if f.Outcome != "" {
		q += fmt.Sprintf(" AND outcome = $%d", n)
		args = append(args, f.Outcome)
		n++
	}
	if f.Since != nil {
		q += fmt.Sprintf(" AND occurred_at >= $%d", n)
		args = append(args, *f.Since)
		n++
	}
	if f.Until != nil {
		q += fmt.Sprintf(" AND occurred_at <= $%d", n)
		args = append(args, *f.Until)
		n++
	}
	if f.Cursor != "" {
		// Cursor is the occurred_at timestamp of the last item (ISO8601)
		t, err := time.Parse(time.RFC3339Nano, f.Cursor)
		if err == nil {
			q += fmt.Sprintf(" AND occurred_at < $%d", n)
			args = append(args, t)
			n++
		}
	}

	q += fmt.Sprintf(" ORDER BY occurred_at DESC LIMIT $%d", n)
	args = append(args, f.Limit+1) // fetch one extra to determine if there's a next page

	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		e := &Event{}
		var detailsJSON []byte
		if err := rows.Scan(&e.ID, &e.TenantID, &e.ActorID, &e.ActorEmail, &e.Action, &e.Resource,
			&e.ResourceID, &e.Outcome, &e.IPAddress, &e.UserAgent, &detailsJSON, &e.OccurredAt); err != nil {
			return nil, "", err
		}
		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &e.Details)
		}
		events = append(events, e)
	}

	var nextCursor string
	if len(events) > f.Limit {
		events = events[:f.Limit]
		nextCursor = events[len(events)-1].OccurredAt.Format(time.RFC3339Nano)
	}

	return events, nextCursor, nil
}

// ExportCSV writes audit events matching the filter as CSV to a buffer.
func (s *Service) ExportCSV(ctx context.Context, tenantID uuid.UUID, f ListFilter) ([]byte, error) {
	f.Limit = 10000 // export cap
	events, _, err := s.List(ctx, tenantID, f)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write([]string{"id", "tenantId", "actorId", "actorEmail", "action", "resource", "resourceId", "outcome", "ipAddress", "occurredAt"})
	for _, e := range events {
		actorID := ""
		if e.ActorID != nil {
			actorID = e.ActorID.String()
		}
		w.Write([]string{
			e.ID.String(), e.TenantID.String(), actorID, e.ActorEmail,
			e.Action, e.Resource, e.ResourceID, e.Outcome, e.IPAddress,
			e.OccurredAt.Format(time.RFC3339),
		})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}
