package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestCommandParts verifies that commandParts returns the full command path from root to leaf.
func TestCommandParts(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *cobra.Command
		expected []string
	}{
		{
			name: "single root command",
			build: func() *cobra.Command {
				return &cobra.Command{Use: "myapp"}
			},
			expected: []string{"myapp"},
		},
		{
			name: "two-level command",
			build: func() *cobra.Command {
				root := &cobra.Command{Use: "myapp"}
				child := &cobra.Command{Use: "serve"}
				root.AddCommand(child)
				return child
			},
			expected: []string{"myapp", "serve"},
		},
		{
			name: "three-level command",
			build: func() *cobra.Command {
				root := &cobra.Command{Use: "myapp"}
				parent := &cobra.Command{Use: "config"}
				child := &cobra.Command{Use: "set"}
				root.AddCommand(parent)
				parent.AddCommand(child)
				return child
			},
			expected: []string{"myapp", "config", "set"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.build()

			// Act
			parts := commandParts(cmd)

			// Assert
			assert.Equal(t, tt.expected, parts,
				"commandParts should return the full command path from root to leaf")
		})
	}
}
