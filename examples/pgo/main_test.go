package main

import "testing"

func TestHottestWordIsDeterministic(t *testing.T) {
	t.Parallel()
	text := corpus()

	word, count := hottestWord(text)
	if word == "" || count == 0 {
		t.Fatalf("hottestWord returned %q/%d, want a real result", word, count)
	}
	// Same input, same answer — the property that makes profile runs and
	// benchmark runs comparable.
	again, againCount := hottestWord(text)
	if word != again || count != againCount {
		t.Fatalf("non-deterministic result: %q/%d then %q/%d", word, count, again, againCount)
	}
}

func TestHottestWordCountsCorrectly(t *testing.T) {
	t.Parallel()
	word, count := hottestWord("b a a c a b ")
	if word != "a" || count != 3 {
		t.Fatalf("hottestWord = %q/%d, want \"a\"/3", word, count)
	}
}

func TestHottestWordBreaksTiesAlphabetically(t *testing.T) {
	t.Parallel()
	word, count := hottestWord("beta alpha beta alpha ")
	if word != "alpha" || count != 2 {
		t.Fatalf("hottestWord = %q/%d, want \"alpha\"/2", word, count)
	}
}

// BenchmarkHottestWord is what `make compare` runs with PGO off and on.
func BenchmarkHottestWord(b *testing.B) {
	text := corpus()
	b.ResetTimer()
	for b.Loop() {
		hottestWord(text)
	}
}
