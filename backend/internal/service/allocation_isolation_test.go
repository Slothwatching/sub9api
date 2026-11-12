package service

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Fork-local D139: package-wide background goroutines can contaminate the
// process-wide allocation counter. Keep the assertions in a fresh process.
func runAllocationCheckInSubprocess(t *testing.T) bool {
	t.Helper()
	const childEnv = "SUB9API_ALLOCATION_TEST_CHILD"
	if os.Getenv(childEnv) == t.Name() {
		return false
	}

	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^"+regexp.QuoteMeta(t.Name())+"$", "-test.count=1")
	cmd.Env = append(os.Environ(), childEnv+"="+t.Name())
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "isolated allocation check failed: %s", output)
	return true
}
