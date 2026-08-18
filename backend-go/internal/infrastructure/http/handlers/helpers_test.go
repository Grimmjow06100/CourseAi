package handlers

import (
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
)

func TestParseCourseOrderField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value string
		want  contract.CourseOrderField
		ok    bool
	}{
		{value: "createdAt", want: contract.CourseOrderByCreatedAt, ok: true},
		{value: "updated_at", want: contract.CourseOrderByUpdatedAt, ok: true},
		{value: " TITLE ", want: contract.CourseOrderByTitle, ok: true},
		{value: "status", want: contract.CourseOrderByStatus, ok: true},
		{value: "duration", ok: false},
	}
	for _, test := range tests {
		got, ok := parseCourseOrderField(test.value)
		if got != test.want || ok != test.ok {
			t.Fatalf("parseCourseOrderField(%q) = %q, %t; want %q, %t", test.value, got, ok, test.want, test.ok)
		}
	}
}
