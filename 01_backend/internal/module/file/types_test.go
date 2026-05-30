package file

import "testing"

func TestSanitizeFilename(t *testing.T) {
	cases := map[string]string{
		"../a.txt": "a.txt",
		"":         "file",
		".":        "file",
		"a b.pdf":  "a b.pdf",
	}
	for in, want := range cases {
		if got := sanitizeFilename(in); got != want {
			t.Fatalf("sanitizeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestObjectKeyKeepsTenantScope(t *testing.T) {
	got := objectKey("tenant-1", "file-1", "../demo.txt")
	want := "tenants/tenant-1/files/file-1/demo.txt"
	if got != want {
		t.Fatalf("objectKey = %q, want %q", got, want)
	}
}

func TestPublicURL(t *testing.T) {
	if got := publicURL("https://cdn.example.com/", "a/b.txt"); got != "https://cdn.example.com/a/b.txt" {
		t.Fatalf("publicURL = %q", got)
	}
	if got := publicURL("", "a/b.txt"); got != "" {
		t.Fatalf("publicURL without base = %q", got)
	}
}
