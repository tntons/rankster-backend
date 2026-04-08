package handlers

import "testing"

func TestTierKeyToScore(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		in   string
		want int
	}{
		{name: "S tier", in: "S", want: 5},
		{name: "A tier", in: "A", want: 4},
		{name: "B tier", in: "B", want: 3},
		{name: "C tier", in: "C", want: 2},
		{name: "D tier", in: "D", want: 1},
		{name: "unknown tier", in: "X", want: 0},
		{name: "lowercase unsupported", in: "s", want: 0},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := tierKeyToScore(testCase.in)
			if got != testCase.want {
				t.Fatalf("tierKeyToScore(%q) = %d, want %d", testCase.in, got, testCase.want)
			}
		})
	}
}
