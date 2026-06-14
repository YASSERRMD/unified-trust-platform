package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PolicyDocument is the in-memory representation of a stored policy row.
type PolicyDocument struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenantId"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Effect      string          `json:"effect"`
	Priority    int             `json:"priority"`
	Subjects    PolicySubjects  `json:"subjects"`
	Actions     []string        `json:"actions"`
	Resources   PolicyResources `json:"resources"`
	Conditions  []Condition     `json:"conditions"`
	IsActive    bool            `json:"isActive"`
}

type PolicySubjects struct {
	Roles  []string `json:"roles"`
	Users  []string `json:"users"`
	Groups []string `json:"groups"`
}

type PolicyResources struct {
	Types      []string    `json:"types"`
	Conditions []Condition `json:"conditions"`
}

type Condition struct {
	Attribute string `json:"attribute"`
	Operator  string `json:"operator"`
	Value     any    `json:"value"`
}

// EvalRequest represents a single authorization evaluation request.
type EvalRequest struct {
	Subject  SubjectContext  `json:"subject"`
	Action   string          `json:"action"`
	Resource ResourceContext `json:"resource"`
	Env      EnvContext      `json:"context"`
}

type SubjectContext struct {
	UserID     string         `json:"userId"`
	TenantID   string         `json:"tenantId"`
	Roles      []string       `json:"roles"`
	Attributes map[string]any `json:"attributes"`
}

type ResourceContext struct {
	Type       string         `json:"type"`
	ID         string         `json:"id"`
	Attributes map[string]any `json:"attributes"`
}

type EnvContext struct {
	IPAddress   string         `json:"ipAddress"`
	NetworkZone string         `json:"networkZone"`
	Time        time.Time      `json:"time"`
	Attributes  map[string]any `json:"attributes"`
}

// EvalDecision is the authorization decision returned to the caller.
type EvalDecision struct {
	Allowed         bool     `json:"allowed"`
	Decision        string   `json:"decision"`
	Reason          string   `json:"reason"`
	MatchedPolicies []string `json:"matchedPolicies"`
	EvaluationMs    int64    `json:"evaluationMs"`
}

// PolicyEngine evaluates authorization requests against stored policies.
type PolicyEngine struct {
	db   *pgxpool.Pool
	rbac *RBACService
}

func NewPolicyEngine(db *pgxpool.Pool, rbac *RBACService) *PolicyEngine {
	return &PolicyEngine{db: db, rbac: rbac}
}

// Evaluate runs the full RBAC + ABAC + PBAC evaluation.
func (e *PolicyEngine) Evaluate(ctx context.Context, tenantID uuid.UUID, req EvalRequest) (*EvalDecision, error) {
	start := time.Now()

	decision := &EvalDecision{
		Decision:        "deny",
		Reason:          "implicit deny: no matching policy",
		MatchedPolicies: []string{},
	}

	// 1. RBAC: check if user has the permission directly via roles
	parts := strings.SplitN(req.Action, ":", 2)
	if len(parts) == 2 {
		userID, err := uuid.Parse(req.Subject.UserID)
		if err == nil {
			rbacPermitted, err := e.rbac.HasPermission(ctx, userID, tenantID, parts[0], parts[1])
			if err == nil && rbacPermitted {
				// RBAC permits — still check for explicit PBAC deny
				rbacOK := true
				_ = rbacOK
			}
		}
	}

	// 2. Load active PBAC policies for tenant
	policies, err := e.loadPolicies(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var permitPolicies []string
	var denyPolicies []string

	for _, p := range policies {
		if !e.matchesSubject(p, req.Subject) {
			continue
		}
		if !e.matchesAction(p, req.Action) {
			continue
		}
		if !e.matchesResource(p, req.Resource) {
			continue
		}
		if !e.evaluateConditions(p.Conditions, req) {
			continue
		}

		if p.Effect == "permit" {
			permitPolicies = append(permitPolicies, p.ID.String())
		} else {
			denyPolicies = append(denyPolicies, p.ID.String())
		}
	}

	decision.EvaluationMs = time.Since(start).Milliseconds()

	// Deny-override: explicit deny beats any permit
	if len(denyPolicies) > 0 {
		decision.Allowed = false
		decision.Decision = "deny"
		decision.Reason = "explicit deny policy matched"
		decision.MatchedPolicies = denyPolicies
		return decision, nil
	}

	if len(permitPolicies) > 0 {
		decision.Allowed = true
		decision.Decision = "permit"
		decision.Reason = "permit policy matched"
		decision.MatchedPolicies = permitPolicies
		return decision, nil
	}

	// Fallback: RBAC only
	parts = strings.SplitN(req.Action, ":", 2)
	if len(parts) == 2 {
		userID, err := uuid.Parse(req.Subject.UserID)
		if err == nil {
			rbacPermitted, err := e.rbac.HasPermission(ctx, userID, tenantID, parts[0], parts[1])
			if err == nil && rbacPermitted {
				decision.Allowed = true
				decision.Decision = "permit"
				decision.Reason = "RBAC: role-based permission"
				return decision, nil
			}
		}
	}

	return decision, nil
}

func (e *PolicyEngine) loadPolicies(ctx context.Context, tenantID uuid.UUID) ([]*PolicyDocument, error) {
	rows, err := e.db.Query(ctx,
		`SELECT id, name, description, effect, priority, subjects, actions, resources, conditions
		 FROM policies WHERE tenant_id = $1 AND is_active = TRUE ORDER BY priority DESC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []*PolicyDocument
	for rows.Next() {
		p := &PolicyDocument{TenantID: tenantID, IsActive: true}
		var subjectsJSON, resourcesJSON, conditionsJSON []byte
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Effect, &p.Priority,
			&subjectsJSON, &p.Actions, &resourcesJSON, &conditionsJSON); err != nil {
			return nil, err
		}
		json.Unmarshal(subjectsJSON, &p.Subjects)
		json.Unmarshal(resourcesJSON, &p.Resources)
		json.Unmarshal(conditionsJSON, &p.Conditions)
		policies = append(policies, p)
	}
	return policies, nil
}

func (e *PolicyEngine) matchesSubject(p *PolicyDocument, sub SubjectContext) bool {
	if len(p.Subjects.Users) == 0 && len(p.Subjects.Roles) == 0 && len(p.Subjects.Groups) == 0 {
		return true
	}
	for _, u := range p.Subjects.Users {
		if u == sub.UserID || u == "*" {
			return true
		}
	}
	for _, pr := range p.Subjects.Roles {
		for _, sr := range sub.Roles {
			if pr == sr || pr == "*" {
				return true
			}
		}
	}
	return false
}

func (e *PolicyEngine) matchesAction(p *PolicyDocument, action string) bool {
	for _, a := range p.Actions {
		if a == action || a == "*" {
			return true
		}
	}
	return false
}

func (e *PolicyEngine) matchesResource(p *PolicyDocument, res ResourceContext) bool {
	if len(p.Resources.Types) == 0 {
		return true
	}
	for _, t := range p.Resources.Types {
		if t == res.Type || t == "*" {
			return true
		}
	}
	return false
}

func (e *PolicyEngine) evaluateConditions(conditions []Condition, req EvalRequest) bool {
	for _, c := range conditions {
		val := resolveAttribute(c.Attribute, req)
		if !evalCondition(c.Operator, val, c.Value) {
			return false
		}
	}
	return true
}

func resolveAttribute(attr string, req EvalRequest) any {
	parts := strings.SplitN(attr, ".", 2)
	if len(parts) != 2 {
		return nil
	}
	switch parts[0] {
	case "subject":
		if req.Subject.Attributes != nil {
			return req.Subject.Attributes[parts[1]]
		}
	case "resource":
		if req.Resource.Attributes != nil {
			return req.Resource.Attributes[parts[1]]
		}
	case "environment", "env":
		switch parts[1] {
		case "networkZone":
			return req.Env.NetworkZone
		case "ipAddress":
			return req.Env.IPAddress
		}
		if req.Env.Attributes != nil {
			return req.Env.Attributes[parts[1]]
		}
	}
	return nil
}

func evalCondition(op string, actual, expected any) bool {
	switch op {
	case "eq":
		return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
	case "neq":
		return fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected)
	case "in":
		exp := reflect.ValueOf(expected)
		if exp.Kind() != reflect.Slice {
			return false
		}
		for i := 0; i < exp.Len(); i++ {
			if fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", exp.Index(i).Interface()) {
				return true
			}
		}
		return false
	case "contains":
		return strings.Contains(fmt.Sprintf("%v", actual), fmt.Sprintf("%v", expected))
	}
	return false
}

// CreatePolicy stores a new policy document.
func (e *PolicyEngine) CreatePolicy(ctx context.Context, tenantID uuid.UUID, p *PolicyDocument) error {
	p.ID = uuid.New()
	p.TenantID = tenantID
	p.IsActive = true

	subJSON, _ := json.Marshal(p.Subjects)
	resJSON, _ := json.Marshal(p.Resources)
	conJSON, _ := json.Marshal(p.Conditions)

	_, err := e.db.Exec(ctx,
		`INSERT INTO policies (id, tenant_id, name, description, effect, priority, subjects, actions, resources, conditions, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		p.ID, p.TenantID, p.Name, p.Description, p.Effect, p.Priority,
		subJSON, p.Actions, resJSON, conJSON, p.IsActive,
	)
	return err
}
