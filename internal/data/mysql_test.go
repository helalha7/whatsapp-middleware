package data

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestIsDuplicateError(t *testing.T) {
	duplicateError := &mysql.MySQLError{Number: 1062, Message: "duplicate entry"}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "duplicate error", err: duplicateError, want: true},
		{name: "wrapped duplicate error", err: fmt.Errorf("insert tenant: %w", duplicateError), want: true},
		{name: "other MySQL error", err: &mysql.MySQLError{Number: 1205}},
		{name: "other error", err: errors.New("database unavailable")},
		{name: "nil error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDuplicateError(tt.err); got != tt.want {
				t.Errorf("expected %t, got %t", tt.want, got)
			}
		})
	}
}

func TestIsForeignKeyViolation(t *testing.T) {
	foreignKeyError := &mysql.MySQLError{Number: 1452, Message: "foreign key violation"}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "foreign key error", err: foreignKeyError, want: true},
		{name: "wrapped foreign key error", err: fmt.Errorf("insert message: %w", foreignKeyError), want: true},
		{name: "other MySQL error", err: &mysql.MySQLError{Number: 1062}},
		{name: "other error", err: errors.New("database unavailable")},
		{name: "nil error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsForeignKeyViolation(tt.err); got != tt.want {
				t.Errorf("expected %t, got %t", tt.want, got)
			}
		})
	}
}
