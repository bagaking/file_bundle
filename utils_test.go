package main

import "testing"

func TestShrinkContent(t *testing.T) {
	tests := []struct {
		name       string
		shrinkMode bool
		input      string
		want       string
	}{
		{
			name:       "preserves content when disabled",
			shrinkMode: false,
			input:      "  alpha  \n\n\n beta\t\n",
			want:       "  alpha  \n\n\n beta\t\n",
		},
		{
			name:       "trims lines and collapses repeated empty lines",
			shrinkMode: true,
			input:      "  alpha  \n\n   \n beta\t\n",
			want:       "alpha\n\nbeta\n",
		},
		{
			name:       "keeps a single empty line between blocks",
			shrinkMode: true,
			input:      "first\n\n\nsecond",
			want:       "first\n\nsecond",
		},
	}

	originalShrink := shrink
	t.Cleanup(func() {
		shrink = originalShrink
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shrink = tt.shrinkMode

			got := shrinkContent([]byte(tt.input))
			if got != tt.want {
				t.Errorf("shrinkContent(%q) with shrink=%t = %q, want %q", tt.input, tt.shrinkMode, got, tt.want)
			}
		})
	}
}
