package bot

import "testing"

func TestGetResponse(t *testing.T) {

	tests := []struct {
		input    string
		expected string
	}{
		{"Hello bot!", "Hello! 👋 How can I help you?"},
		{"How are you?", "I'm doing great! 😎 Thanks for asking."},
		{"Goodbye", "Goodbye! 👋 Have a great day!"},
		{"What is your name?", "I'm your Go ChatBot 🤖"},
	}

	for _, test := range tests {

		result := GetResponse(test.input)

		if result != test.expected {
			t.Errorf(
				"GetResponse(%q) = %q, expected %q",
				test.input,
				result,
				test.expected,
			)
		}
	}
}
