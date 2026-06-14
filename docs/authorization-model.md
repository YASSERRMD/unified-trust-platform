# Unified Trust Platform — Authorization Model

## Overview

The platform implements a **unified authorization engine** combining three complementary models evaluated in order:

```
Request → RBAC check → ABAC check → PBAC evaluation → Decision
               ↓            ↓              ↓
           Role-based   Attribute-     Policy document
           permission   based rules    evaluation
```

Final decision: **deny unless explicitly permitted** (closed-world assumption).

---

## 1. RBAC — Role-Based Access Control

### Model
```
User ──belongs to──▶ Role ──has──▶ Permission
                     ↑
               Tenant-scoped
```

### Permission Format
```
<service>:<resource>:<action>
```
Examples:
- `iam:user:read`
- `iam:user:write`
- `iam:role:assign`
- `oauth:client:register`
- `audit:event:export`

### Evaluation
1. Resolve all roles for the requesting user in the tenant
2. Collect all permissions from those roles
3. Check if the required permission is in the set
4. Result: `permit` or `deny`

### Hierarchy
Roles can inherit from parent roles (role inheritance tree, max depth 5).

---

## 2. ABAC — Attribute-Based Access Control

### Attributes
**Subject attributes** (from user profile and context):
- `department`, `jobTitle`, `location`, `clearanceLevel`
- `authMethod` (password, mfa, sso)
- `sessionAge` (seconds since login)

**Resource attributes** (from resource metadata):
- `owner`, `ownerDepartment`, `classification`, `tenantId`
- `sensitivity` (public, internal, confidential, restricted)

**Environment attributes** (from request context):
- `ipAddress`, `userAgent`, `time`, `dayOfWeek`
- `networkZone` (internal, vpn, external)

### Condition Syntax
```json
{
  "conditions": [
    { "attribute": "subject.department", "operator": "eq", "value": "resource.ownerDepartment" },
    { "attribute": "environment.networkZone", "operator": "in", "value": ["internal", "vpn"] },
    { "attribute": "subject.sessionAge", "operator": "lt", "value": 3600 }
  ],
  "operator": "AND"
}
```

### Operators
`eq`, `neq`, `in`, `not_in`, `gt`, `gte`, `lt`, `lte`, `contains`, `starts_with`, `regex`

---

## 3. PBAC — Policy-Based Access Control

### Policy Document Format
```json
{
  "id": "policy-uuid",
  "name": "Finance Read Access",
  "tenantId": "tenant-uuid",
  "effect": "permit",
  "priority": 100,
  "subjects": {
    "roles": ["finance-analyst"],
    "users": [],
    "groups": []
  },
  "actions": ["iam:report:read", "iam:report:export"],
  "resources": {
    "types": ["report"],
    "conditions": [
      { "attribute": "resource.classification", "operator": "in", "value": ["internal", "public"] }
    ]
  },
  "conditions": [
    { "attribute": "environment.networkZone", "operator": "in", "value": ["internal", "vpn"] }
  ],
  "timeConstraints": {
    "daysOfWeek": ["Mon", "Tue", "Wed", "Thu", "Fri"],
    "hoursUTC": { "start": 8, "end": 18 }
  }
}
```

### Evaluation Algorithm
1. Load all active policies for the tenant
2. Filter policies where subject matches (user, role, or group)
3. Filter policies where action matches
4. Filter policies where resource matches
5. Evaluate all conditions (subject + resource + environment)
6. Apply time constraints if present
7. Collect matching policy effects (`permit` / `deny`)
8. **Explicit deny overrides any permit** (deny-override combining algorithm)
9. If no policies match: **implicit deny**

---

## 4. Combined Evaluation

```
┌──────────────────────────────────────────────────────┐
│                 Evaluation Request                    │
│  { subject, action, resource, context }              │
└─────────────────────────┬────────────────────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │   1. RBAC Evaluation  │
              │   Does subject have   │
              │   the permission?     │
              └───────────┬───────────┘
                          │ permit/deny
                          ▼
              ┌───────────────────────┐
              │   2. ABAC Conditions  │
              │   Do attribute rules  │
              │   pass?               │
              └───────────┬───────────┘
                          │ pass/fail
                          ▼
              ┌───────────────────────┐
              │   3. PBAC Policies    │
              │   Match active policy │
              │   documents           │
              └───────────┬───────────┘
                          │ permit/deny/no-match
                          ▼
              ┌───────────────────────┐
              │   4. Final Decision   │
              │   deny-override       │
              │   closed-world        │
              └───────────────────────┘
```

### Decision Response
```json
{
  "allowed": true,
  "decision": "permit",
  "reason": "User has role finance-analyst; department attribute matches; policy 'Finance Read Access' permits",
  "matchedPolicies": ["policy-uuid-1"],
  "evaluationMs": 3
}
```

---

## 5. Delegation and JIT Overlay

When a delegation grant exists for the requesting user:
- The delegated user's effective permissions are added to the subject's permission set
- Scope constraints are enforced (only specific permissions from the delegation)
- Time constraints are checked before adding delegated permissions

When a JIT grant is active:
- The granted role or permission is added for the session duration
- Evaluated identically to a permanent role assignment
- Automatically expires at the grant's `expiresAt` timestamp

---

## 6. Policy Conflict Resolution

| Scenario | Result |
|----------|--------|
| No matching policy | implicit deny |
| Only permit matches | permit |
| Only deny matches | deny |
| Both permit and deny match | deny (deny-override) |
| No PBAC policies exist, RBAC permits | permit |
