package mirror_test

import (
	"testing"

	"flutter-gradle-tool/internal/mirror"
)

func TestFindByName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "exact", input: "aliyun", expected: "aliyun"},
		{name: "case-insensitive", input: "Tencent", expected: "tencent"},
		{name: "trimmed", input: "  huaweicloud  ", expected: "huaweicloud"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			source := mirror.FindByName(tc.input)
			if source == nil {
				t.Fatalf("FindByName(%q) returned nil", tc.input)
			}
			if source.Name != tc.expected {
				t.Fatalf("FindByName(%q) = %q, want %q", tc.input, source.Name, tc.expected)
			}
		})
	}
}

func TestFindByNameUnknown(t *testing.T) {
	t.Parallel()

	if source := mirror.FindByName("missing"); source != nil {
		t.Fatalf("FindByName returned %+v, want nil", source)
	}
}

func TestBuiltinSourcesNames(t *testing.T) {
	t.Parallel()

	got := mirror.Names()
	want := []string{"official", "tencent", "aliyun", "huaweicloud"}
	if len(got) != len(want) {
		t.Fatalf("Names() length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Names()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
