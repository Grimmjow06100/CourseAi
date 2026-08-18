package pointer

import "testing"

func TestToAndCloneReturnIndependentPointers(t *testing.T) {
	original := To("course-ai")
	cloned := Clone(original)
	if cloned == original || *cloned != *original {
		t.Fatal("expected an independent pointer with the same value")
	}

	*cloned = "changed"
	if *original != "course-ai" {
		t.Fatal("changing the clone must not mutate the original")
	}
}

func TestEqualHandlesOptionalValues(t *testing.T) {
	if !Equal[int](nil, nil) {
		t.Fatal("two nil pointers must be equal")
	}
	if Equal(To(1), nil) || !Equal(To(1), To(1)) || Equal(To(1), To(2)) {
		t.Fatal("unexpected optional value comparison")
	}
}

func TestMapPreservesNilAndTransformsValue(t *testing.T) {
	if Map[int, string](nil, func(int) string { return "unused" }) != nil {
		t.Fatal("mapping nil must return nil")
	}

	mapped := Map(To(42), func(value int) string { return "value" })
	if mapped == nil || *mapped != "value" {
		t.Fatalf("unexpected mapped value: %v", mapped)
	}
}
