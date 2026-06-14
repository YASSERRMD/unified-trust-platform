package validate

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type FieldErrors map[string]string

func (fe FieldErrors) Error() string {
	parts := make([]string, 0, len(fe))
	for k, v := range fe {
		parts = append(parts, fmt.Sprintf("%s: %s", k, v))
	}
	return strings.Join(parts, "; ")
}

func DecodeJSON(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1 MB limit
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func Required(field, value string, errs FieldErrors) {
	if strings.TrimSpace(value) == "" {
		errs[field] = "is required"
	}
}

func MinLength(field, value string, min int, errs FieldErrors) {
	if len(strings.TrimSpace(value)) < min {
		errs[field] = fmt.Sprintf("must be at least %d characters", min)
	}
}

func MaxLength(field, value string, max int, errs FieldErrors) {
	if len(value) > max {
		errs[field] = fmt.Sprintf("must be at most %d characters", max)
	}
}

func Email(field, value string, errs FieldErrors) {
	if !strings.Contains(value, "@") || !strings.Contains(value, ".") {
		errs[field] = "must be a valid email address"
	}
}

func OneOf(field, value string, options []string, errs FieldErrors) {
	for _, o := range options {
		if value == o {
			return
		}
	}
	errs[field] = fmt.Sprintf("must be one of: %s", strings.Join(options, ", "))
}
