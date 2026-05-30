package kb

import "testing"

func TestSplitTextChunks(t *testing.T) {
	got := SplitTextChunks("abcdef", 3, 1)
	want := []string{"abc", "cde", "ef"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("chunk %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSplitTextChunksEmpty(t *testing.T) {
	if got := SplitTextChunks("  ", 10, 0); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestIsPlainTextFile(t *testing.T) {
	if !isPlainTextFile("text/plain", "a.bin") {
		t.Fatal("text/plain should be supported")
	}
	if !isPlainTextFile("", "README.md") {
		t.Fatal("markdown extension should be supported")
	}
	if isPlainTextFile("application/pdf", "a.pdf") {
		t.Fatal("pdf should not be supported by MVP parser")
	}
}
