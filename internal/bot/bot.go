package bot

import "strings"

func GetResponse(message string) string {

	message = strings.ToLower(strings.TrimSpace(message))

	switch {
	case strings.Contains(message, "hello"),
		strings.Contains(message, "hi"),
		strings.Contains(message, "hey"):
		return "Hello! 👋 How can I help you?"

	case strings.Contains(message, "how are you"):
		return "I'm doing great! 😎 Thanks for asking."

	case strings.Contains(message, "bye"),
		strings.Contains(message, "goodbye"):
		return "Goodbye! 👋 Have a great day!"

	case strings.Contains(message, "your name"):
		return "I'm your Go ChatBot 🤖"

	default:
		return "I'm still learning. 🤖"
	}
}
