package skills

import (
	"fmt"
	"math/rand"
	"testing"
)

func BenchmarkSortedCategories(b *testing.B) {
	categories := make(map[string][]*Skill)

	// Create 100+ categories for meaningful benchmark
	for range 100 {
		category := fmt.Sprintf("cat_%d", rand.Int())
		categories[category] = []*Skill{}
	}

	builder := &SummaryBuilder{}

	b.ResetTimer()
	for b.Loop() {
		builder.sortedCategories(categories)
	}
}

func TestSortedCategories(t *testing.T) {
	categories := make(map[string][]*Skill)
	categories["zebra"] = []*Skill{}
	categories["apple"] = []*Skill{}
	categories["banana"] = []*Skill{}
	categories["orange"] = []*Skill{}

	builder := &SummaryBuilder{}
	result := builder.sortedCategories(categories)

	// Verify result is alphabetically sorted
	expected := []string{"apple", "banana", "orange", "zebra"}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d categories, got %d", len(expected), len(result))
	}

	for i, cat := range expected {
		if result[i] != cat {
			t.Errorf("Expected category at index %d to be %s, got %s", i, cat, result[i])
		}
	}
}
