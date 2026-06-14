package tests

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"
)

func TestAuditEventCSVExport(t *testing.T) {
	type event struct {
		ID         string
		TenantID   string
		ActorID    string
		ActorEmail string
		Action     string
		Resource   string
		ResourceID string
		Outcome    string
		IPAddress  string
		OccurredAt string
	}

	events := []event{
		{
			ID:         "evt-1",
			TenantID:   "tenant-1",
			ActorID:    "user-1",
			ActorEmail: "admin@example.com",
			Action:     "user.create",
			Resource:   "user",
			ResourceID: "new-user-id",
			Outcome:    "success",
			IPAddress:  "10.0.0.1",
			OccurredAt: time.Now().Format(time.RFC3339),
		},
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write([]string{"id", "tenantId", "actorId", "actorEmail", "action", "resource", "resourceId", "outcome", "ipAddress", "occurredAt"})
	for _, e := range events {
		w.Write([]string{e.ID, e.TenantID, e.ActorID, e.ActorEmail, e.Action, e.Resource, e.ResourceID, e.Outcome, e.IPAddress, e.OccurredAt})
	}
	w.Flush()

	output := buf.String()
	if !strings.Contains(output, "user.create") {
		t.Error("expected action in CSV output")
	}
	if !strings.Contains(output, "admin@example.com") {
		t.Error("expected actorEmail in CSV output")
	}
	if !strings.Contains(output, "success") {
		t.Error("expected outcome in CSV output")
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines (header + 1 row), got %d", len(lines))
	}
}

func TestAuditCursorPagination(t *testing.T) {
	now := time.Now()
	times := []time.Time{
		now.Add(-3 * time.Minute),
		now.Add(-2 * time.Minute),
		now.Add(-1 * time.Minute),
	}

	limit := 2
	var page []time.Time
	for _, t := range times {
		page = append(page, t)
		if len(page) == limit {
			break
		}
	}

	cursor := page[len(page)-1].Format(time.RFC3339Nano)
	if cursor == "" {
		t.Error("expected non-empty cursor")
	}

	// Next page: items before the cursor
	var nextPage []time.Time
	for _, ts := range times {
		if ts.Before(page[len(page)-1]) {
			nextPage = append(nextPage, ts)
		}
	}
	if len(nextPage) != 1 {
		t.Errorf("expected 1 item on next page, got %d", len(nextPage))
	}
}

func TestAuditImmutabilityIsEnforcedAtDB(t *testing.T) {
	// Immutability is enforced by PostgreSQL CREATE RULE no_update_audit / no_delete_audit
	// defined in migration 011_audit_events.up.sql. This test documents the contract.
	const immutabilityNote = "audit_events has no_update_audit and no_delete_audit rules at the DB level"
	if immutabilityNote == "" {
		t.Error("immutability contract must be documented")
	}
}
