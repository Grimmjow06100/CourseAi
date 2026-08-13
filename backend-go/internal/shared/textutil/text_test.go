package textutil

import (
	"reflect"
	"testing"
)

func TestTextNormalization(t *testing.T) {
	if got := CollapseWhitespace("  formation\n Linux\t avancee "); got != "formation Linux avancee" {
		t.Fatalf("unexpected collapsed text: %q", got)
	}
	want := []string{"one", "two words"}
	if got := NormalizeNonBlank([]string{" one ", " ", "two\twords"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected normalized values: %#v", got)
	}
}

func TestSplitAndTrimNonBlank(t *testing.T) {
	want := []string{"https://one.example", "https://two.example"}
	got := SplitNonBlank(" https://one.example, ,https://two.example ", ",")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected split values: %#v", got)
	}
}

func TestOptionalTextHelpers(t *testing.T) {
	if TrimmedPointer("  ") != nil || TrimmedPointerFrom(nil) != nil {
		t.Fatal("blank and nil optional strings must remain nil")
	}
	value := " value "
	if got := TrimmedPointerFrom(&value); got == nil || *got != "value" {
		t.Fatalf("unexpected optional value: %v", got)
	}
	if got := FirstNonBlank(" ", " selected ", "fallback"); got != " selected " {
		t.Fatalf("unexpected first non-blank value: %q", got)
	}
	if got := ValueOr(&value, "fallback"); got != "value" {
		t.Fatalf("unexpected optional fallback value: %q", got)
	}
}
