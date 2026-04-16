package cmd

import "testing"

func TestParseMeasurement(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  float64
		ok    bool
	}{
		{name: "plain number", input: "39.5", want: 39.5, ok: true},
		{name: "with commas", input: "1,234", want: 1234, ok: true},
		{name: "frequency mhz", input: "606.5 MHz", want: 606.5e6, ok: true},
		{name: "frequency hz", input: "609000000 Hz", want: 609000000, ok: true},
		{name: "symbol rate msps", input: "5.12 Msym/s", want: 5.12e6, ok: true},
		{name: "invalid", input: "n/a", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseMeasurement(tt.input)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if tt.ok && got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseLockStatus(t *testing.T) {
	tests := []struct {
		input string
		want  float64
		ok    bool
	}{
		{input: "Locked", want: 1, ok: true},
		{input: "Not Locked", want: 0, ok: true},
		{input: "Unlocked", want: 0, ok: true},
		{input: "Online", want: 1, ok: true},
		{input: "", ok: false},
	}

	for _, tt := range tests {
		got, ok := parseLockStatus(tt.input)
		if ok != tt.ok {
			t.Fatalf("input %q ok = %v, want %v", tt.input, ok, tt.ok)
		}
		if tt.ok && got != tt.want {
			t.Fatalf("input %q got %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestChannelLabel(t *testing.T) {
	if got := channelLabel(" 12 "); got != "12" {
		t.Fatalf("got %q, want %q", got, "12")
	}
	if got := channelLabel(""); got != "unknown" {
		t.Fatalf("got %q, want %q", got, "unknown")
	}
}
