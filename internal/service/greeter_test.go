package service

import "testing"

func TestGreeterGreet(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "provided name", in: "Ada", want: "Hello, Ada!"},
		{name: "blank name", in: "  ", want: "Hello, World!"},
		{name: "trims name", in: "  Lin  ", want: "Hello, Lin!"},
	}

	greeter := Greeter{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := greeter.Greet(tt.in); got != tt.want {
				t.Fatalf("Greet(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
