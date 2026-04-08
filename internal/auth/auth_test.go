package auth

import "testing"

func TestFromAuthorization(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		in   string
		want Context
	}{
		{name: "empty header", in: "", want: Context{Kind: "anonymous"}},
		{name: "wrong scheme", in: "Basic abc", want: Context{Kind: "anonymous"}},
		{name: "missing token", in: "Bearer   ", want: Context{Kind: "anonymous"}},
		{name: "valid bearer", in: "Bearer user-123", want: Context{Kind: "user", UserID: "user-123"}},
		{name: "case insensitive prefix", in: "bearer user-456", want: Context{Kind: "user", UserID: "user-456"}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := FromAuthorization(testCase.in)
			if got != testCase.want {
				t.Fatalf("FromAuthorization(%q) = %#v, want %#v", testCase.in, got, testCase.want)
			}
		})
	}
}
