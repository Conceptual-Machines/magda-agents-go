package daw

import (
	"reflect"
	"testing"
)

// TestUndo tests the undo() function
func TestUndo(t *testing.T) {
	tests := []struct {
		name    string
		dslCode string
		want    []map[string]interface{}
		wantErr bool
	}{
		{
			name:    "simple undo",
			dslCode: `undo()`,
			want: []map[string]interface{}{
				{
					"action": "undo",
				},
			},
			wantErr: false,
		},
		// Note: Multiple statements in one DSL call may need to be handled differently
		// For now, we test undo() as a standalone call
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewFunctionalDSLParser()
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}

			got, err := parser.ParseDSL(tt.dslCode)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDSL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseDSL() = %v, want %v", got, tt.want)
			}
		})
	}
}

