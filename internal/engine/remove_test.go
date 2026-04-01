package engine

import "testing"

func TestRemoveCommandForMethod(t *testing.T) {
	tests := []struct {
		method string
		wantOK bool
		want   managedCommand
	}{
		{
			method: "homebrew",
			wantOK: true,
			want:   managedCommand{Name: "brew", Args: []string{"uninstall", "sift"}},
		},
		{
			method: "scoop",
			wantOK: true,
			want:   managedCommand{Name: "scoop", Args: []string{"uninstall", "sift"}},
		},
		{
			method: "winget",
			wantOK: true,
			want:   managedCommand{Name: "winget", Args: []string{"uninstall", "SIFT"}},
		},
		{
			method: "manual",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		got, ok := removeCommandForMethod(tt.method)
		if ok != tt.wantOK {
			t.Fatalf("%s: expected ok=%v, got %v", tt.method, tt.wantOK, ok)
		}
		if !ok {
			continue
		}
		if got.Name != tt.want.Name {
			t.Fatalf("%s: expected command %q, got %q", tt.method, tt.want.Name, got.Name)
		}
		if len(got.Args) != len(tt.want.Args) {
			t.Fatalf("%s: expected args %v, got %v", tt.method, tt.want.Args, got.Args)
		}
		for i := range got.Args {
			if got.Args[i] != tt.want.Args[i] {
				t.Fatalf("%s: expected args %v, got %v", tt.method, tt.want.Args, got.Args)
			}
		}
	}
}
