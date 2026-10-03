package foreground

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessProgramNameUsesGenericArgvZero(t *testing.T) {
	for _, test := range []struct {
		name, argvZero, want string
	}{
		{"unregistered title", "unlisted-cli", "unlisted-cli"},
		{"another unregistered title", "new-command", "new-command"},
		{"executable path", "/opt/bin/my-tool", "my-tool"},
		{"relative executable", "./my-tool", "my-tool"},
		{"unicode name", "工具", "工具"},
		{"runtime remains runtime", "/usr/bin/node", "node"},
		{"empty", "", ""},
		{"command line is not a name", "node /opt/cli.js", ""},
		{"path with whitespace", "/opt/my tools/cli.js", ""},
		{"title with arguments", "unlisted-cli --flag", ""},
		{"control sequence", "\x1b[31munlisted-cli", ""},
		{"newline", "unlisted-cli\n", ""},
		{"nul", "unlisted-cli\x00", ""},
		{"invalid utf8", "tool\xff", ""},
		{"root", "/", ""},
		{"dot", ".", ""},
		{"parent", "..", ""},
		{"option", "--eval", ""},
		{"backslash", "tool\\name", ""},
		{"environment assignment is not a name", "SECRET=/opt/bin/tool", ""},
		{"bounded name", strings.Repeat("a", 64), strings.Repeat("a", 64)},
		{"oversized name", strings.Repeat("a", 65), ""},
	} {
		t.Run(test.name, func(t *testing.T) { require.Equal(t, test.want, processProgramName(test.argvZero)) })
	}
}
