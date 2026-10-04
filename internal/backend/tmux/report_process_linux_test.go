package tmux

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReportProcessStatParsesParentAndBirthOnly(t *testing.T) {
	fields := []string{"S", "12", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "123456"}
	valid := "42 (command with ) brackets) " + strings.Join(fields, " ")
	p, err := parseReportProcessStat(valid, 42)
	require.NoError(t, err)
	require.Equal(t, reportProcess{pid: 42, parent: 12, birth: "123456"}, p)
	for _, value := range []string{"", "42 bad", strings.Replace(valid, "123456", "invalid", 1),
		strings.Replace(valid, "123456", "0", 1), strings.Replace(valid, "S 12", "Z 12", 1),
		strings.Replace(valid, "S 12", "x 12", 1), strings.Replace(valid, "S 12", "SS 12", 1),
		strings.Replace(valid, "S 12", "S -1", 1), strings.Replace(valid, "S 12", "S 2147483648", 1)} {
		_, err := parseReportProcessStat(value, 42)
		require.Error(t, err)
	}
	_, err = parseReportProcessStat(valid, 43)
	require.Error(t, err)
}
