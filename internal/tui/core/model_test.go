package core

import "testing"

func TestWorkspaceContextUsesFriendlyWorkspaceLabel(t *testing.T) {
	tests := []struct {
		name      string
		workspace string
		want      string
	}{
		{name: "named workspace path", workspace: "/Users/me/.config/hetki/workspaces/personal.yaml", want: "WORKSPACE: personal"},
		{name: "local hetki file uses parent dir", workspace: "/Users/me/projects/muxie/.hetki.yaml", want: "WORKSPACE: muxie"},
		{name: "plain label unchanged", workspace: "unmanaged", want: "WORKSPACE: unmanaged"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := workspaceContext(map[string]string{"workspace": tt.workspace})
			if got != tt.want {
				t.Fatalf("workspaceContext() = %q, want %q", got, tt.want)
			}
		})
	}
}
