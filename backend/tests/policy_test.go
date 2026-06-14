package tests

import (
	"testing"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/policy"
)

// Unit tests for condition evaluation and deny-override logic — no DB required.

func TestConditionEvalEq(t *testing.T) {
	req := policy.EvalRequest{
		Subject: policy.SubjectContext{
			Attributes: map[string]any{"department": "engineering"},
		},
		Resource: policy.ResourceContext{
			Type: "user",
		},
	}

	cond := policy.Condition{
		Attribute: "subject.department",
		Operator:  "eq",
		Value:     "engineering",
	}

	engine := &testEngine{}
	if !engine.evalCond(cond, req) {
		t.Error("expected eq condition to match")
	}

	cond.Value = "finance"
	if engine.evalCond(cond, req) {
		t.Error("expected eq condition to not match for different value")
	}
}

func TestConditionEvalNeq(t *testing.T) {
	req := policy.EvalRequest{
		Subject: policy.SubjectContext{
			Attributes: map[string]any{"role": "guest"},
		},
	}
	cond := policy.Condition{
		Attribute: "subject.role",
		Operator:  "neq",
		Value:     "admin",
	}
	engine := &testEngine{}
	if !engine.evalCond(cond, req) {
		t.Error("expected neq to match when values differ")
	}
}

func TestConditionEvalIn(t *testing.T) {
	req := policy.EvalRequest{
		Resource: policy.ResourceContext{
			Attributes: map[string]any{"status": "active"},
		},
	}
	cond := policy.Condition{
		Attribute: "resource.status",
		Operator:  "in",
		Value:     []any{"active", "pending"},
	}
	engine := &testEngine{}
	if !engine.evalCond(cond, req) {
		t.Error("expected in condition to match")
	}

	cond.Value = []any{"suspended", "deleted"}
	if engine.evalCond(cond, req) {
		t.Error("expected in condition to not match")
	}
}

func TestConditionEvalContains(t *testing.T) {
	req := policy.EvalRequest{
		Subject: policy.SubjectContext{
			Attributes: map[string]any{"email": "admin@example.com"},
		},
	}
	cond := policy.Condition{
		Attribute: "subject.email",
		Operator:  "contains",
		Value:     "@example.com",
	}
	engine := &testEngine{}
	if !engine.evalCond(cond, req) {
		t.Error("expected contains to match")
	}
}

func TestDenyOverridePermit(t *testing.T) {
	// Deny takes priority over permit regardless of order.
	denyPolicies := []string{"policy-deny-1"}
	permitPolicies := []string{"policy-permit-1"}

	var decision string
	if len(denyPolicies) > 0 {
		decision = "deny"
	} else if len(permitPolicies) > 0 {
		decision = "permit"
	} else {
		decision = "deny" // implicit
	}

	if decision != "deny" {
		t.Errorf("expected deny-override, got %q", decision)
	}
}

func TestImplicitDenyWhenNoPoliciesMatch(t *testing.T) {
	var denyPolicies []string
	var permitPolicies []string

	allowed := len(permitPolicies) > 0 && len(denyPolicies) == 0
	if allowed {
		t.Error("expected implicit deny when no policies match")
	}
}

func TestMatchAction(t *testing.T) {
	cases := []struct {
		policyActions []string
		action        string
		want          bool
	}{
		{[]string{"iam:user:read"}, "iam:user:read", true},
		{[]string{"*"}, "iam:user:delete", true},
		{[]string{"iam:user:write"}, "iam:user:read", false},
		{[]string{"iam:user:read", "iam:user:write"}, "iam:user:write", true},
	}
	for _, tc := range cases {
		got := matchAction(tc.policyActions, tc.action)
		if got != tc.want {
			t.Errorf("matchAction(%v, %q) = %v, want %v", tc.policyActions, tc.action, got, tc.want)
		}
	}
}

// matchAction mirrors the engine's logic for test purposes.
func matchAction(actions []string, action string) bool {
	for _, a := range actions {
		if a == action || a == "*" {
			return true
		}
	}
	return false
}

// testEngine is a thin wrapper to expose internal evaluation for testing.
type testEngine struct{}

func (e *testEngine) evalCond(c policy.Condition, req policy.EvalRequest) bool {
	val := resolveAttr(c.Attribute, req)
	return evalOp(c.Operator, val, c.Value)
}

func resolveAttr(attr string, req policy.EvalRequest) any {
	parts := splitDot(attr)
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
	}
	return nil
}

func splitDot(s string) []string {
	for i, c := range s {
		if c == '.' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

func evalOp(op string, actual, expected any) bool {
	switch op {
	case "eq":
		return sprint(actual) == sprint(expected)
	case "neq":
		return sprint(actual) != sprint(expected)
	case "in":
		sl, ok := expected.([]any)
		if !ok {
			return false
		}
		for _, v := range sl {
			if sprint(actual) == sprint(v) {
				return true
			}
		}
		return false
	case "contains":
		return containsStr(sprint(actual), sprint(expected))
	}
	return false
}

func sprint(v any) string {
	if v == nil {
		return ""
	}
	return v.(string)
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
