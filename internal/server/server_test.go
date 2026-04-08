package server

import "testing"

func TestSanitizeSlug(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		in   string
		want string
	}{
		{name: "keeps lowercase letters and hyphens", in: "latte-art", want: "latte-art"},
		{name: "normalizes casing and trims slashes", in: " /Alice/ ", want: "alice"},
		{name: "removes unsupported characters", in: "Cold Brew!!!", want: "coldbrew"},
		{name: "falls back when empty", in: "   ", want: "rankster"},
		{name: "falls back when nothing valid remains", in: "@@@", want: "rankster"},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizeSlug(testCase.in)
			if got != testCase.want {
				t.Fatalf("sanitizeSlug(%q) = %q, want %q", testCase.in, got, testCase.want)
			}
		})
	}
}

func TestPrettyLabel(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		in   string
		want string
	}{
		{name: "capitalizes hyphenated words", in: "cold-brew", want: "Cold Brew"},
		{name: "keeps single word", in: "latte", want: "Latte"},
		{name: "preserves empty segments as spacing", in: "double--shot", want: "Double  Shot"},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := prettyLabel(testCase.in)
			if got != testCase.want {
				t.Fatalf("prettyLabel(%q) = %q, want %q", testCase.in, got, testCase.want)
			}
		})
	}
}

func TestFirstRune(t *testing.T) {
	t.Parallel()

	if got := firstRune("rankster"); got != "r" {
		t.Fatalf("firstRune(%q) = %q, want %q", "rankster", got, "r")
	}

	if got := firstRune(""); got != "" {
		t.Fatalf("firstRune(empty) = %q, want empty string", got)
	}
}
