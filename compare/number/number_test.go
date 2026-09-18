package number

import (
	"testing"
)

func TestEqual(t *testing.T) {
	tests := []struct {
		a, b   any
		result bool
	}{
		{1, 1, true},
		{1, 2, false},
		{1.0, 1.0, true},
		{"1", "1", true},
		{"1", "2", false},
	}

	for _, test := range tests {
		if res := Equal(test.a, test.b); res != test.result {
			t.Errorf("Equal(%v, %v) = %v; want %v", test.a, test.b, res, test.result)
		}
	}
}

func TestLessThan(t *testing.T) {
	tests := []struct {
		a, b   any
		result bool
	}{
		{1, 2, true},
		{2, 1, false},
		{1.0, 2.0, true},
		{"1", "2", true},
		{"2", "1", false},
	}

	for _, test := range tests {
		if res := LessThan(test.a, test.b); res != test.result {
			t.Errorf("LessThan(%v, %v) = %v; want %v", test.a, test.b, res, test.result)
		}
	}
}

func TestGreaterThan(t *testing.T) {
	tests := []struct {
		a, b   any
		result bool
	}{
		{2, 1, true},
		{1, 2, false},
		{2.0, 1.0, true},
		{"2", "1", true},
		{"1", "2", false},
	}

	for _, test := range tests {
		if res := GreaterThan(test.a, test.b); res != test.result {
			t.Errorf("GreaterThan(%v, %v) = %v; want %v", test.a, test.b, res, test.result)
		}
	}
}

func TestLessOrEqual(t *testing.T) {
	tests := []struct {
		a, b   any
		result bool
	}{
		{1, 2, true},
		{2, 1, false},
		{1.0, 1.0, true},
		{"1", "2", true},
		{"2", "1", false},
	}

	for _, test := range tests {
		if res := LessOrEqual(test.a, test.b); res != test.result {
			t.Errorf("LessOrEqual(%v, %v) = %v; want %v", test.a, test.b, res, test.result)
		}
	}
}

func TestGreaterOrEqual(t *testing.T) {
	tests := []struct {
		a, b   any
		result bool
	}{
		{2, 1, true},
		{1, 2, false},
		{1.0, 1.0, true},
		{"2", "1", true},
		{"1", "2", false},
	}

	for _, test := range tests {
		if res := GreaterOrEqual(test.a, test.b); res != test.result {
			t.Errorf("GreaterOrEqual(%v, %v) = %v; want %v", test.a, test.b, res, test.result)
		}
	}
}
