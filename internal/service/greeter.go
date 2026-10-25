package service

import (
	"strings"
)

// Greeter creates a greeting for a name.
type Greeter struct{}

// Greet returns a greeting, using "World" when name is blank.
func (Greeter) Greet(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "World"
	}
	return "Hello, " + name + "!"
}
