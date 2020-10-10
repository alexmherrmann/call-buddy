package telephono

import "testing"

func TestHttpMethodAsString(t *testing.T) {
	tests := []struct {
		method   HttpMethod
		shouldbe string
	}{
		{Get, "GET"},
		{Post, "POST"},
		{Delete, "DELETE"},
	}
	for _, test := range tests {
		t.Run(test.shouldbe, func(t *testing.T) {
			if test.method.String() != test.shouldbe {
				t.Error("asString Method didn't return correctly")
			}
		})
	}
}
