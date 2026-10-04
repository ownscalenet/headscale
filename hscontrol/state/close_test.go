package state

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/juanfont/headscale/hscontrol/types"
	"github.com/stretchr/testify/require"
)

// countGoroutines returns how many goroutines have fn in their stack.
func countGoroutines(fn string) int {
	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]

	n := 0
	for g := range strings.SplitSeq(string(buf), "\n\n") {
		if strings.Contains(g, fn) {
			n++
		}
	}

	return n
}

// TestCloseStopsAuthCache proves Close stops the auth cache's cleanup
// goroutine, which otherwise outlives the State, and finishes pending auth
// requests.
func TestCloseStopsAuthCache(t *testing.T) {
	const cacheGoroutine = "golang-lru/v2/expirable."

	before := countGoroutines(cacheGoroutine)

	s, err := NewState(persistTestConfig(t.TempDir() + "/headscale.db"))
	require.NoError(t, err)
	require.Equal(t, before+1, countGoroutines(cacheGoroutine))

	id, err := types.NewAuthID()
	require.NoError(t, err)

	req := types.NewAuthRequest()
	s.SetAuthCacheEntry(id, req)

	require.NoError(t, s.Close())

	select {
	case verdict := <-req.WaitForAuth():
		require.ErrorIs(t, verdict.Err, ErrRegistrationExpired)
	case <-time.After(time.Second):
		t.Fatal("pending auth request not finished by Close")
	}

	require.Eventually(t, func() bool {
		return countGoroutines(cacheGoroutine) == before
	}, time.Second, 10*time.Millisecond)
}
