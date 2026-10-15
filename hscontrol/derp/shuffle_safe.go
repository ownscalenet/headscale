package derp

// shuffleWithDerpRandomLock synchronizes use of the package-level
// derpRandomInst across concurrent callers of GetDERPMap. derpRandom()
// already guards lazy creation of the instance, but the returned *rand.Rand
// is not safe for concurrent use.
//
// This change will allow ownscale to instantiate multiple headscale instances
// without a race condition on derpRandomInst.
func shuffleWithDerpRandomLock(n int, swap func(i, j int)) {
	r := derpRandom()

	derpRandomMu.Lock()
	defer derpRandomMu.Unlock()

	r.Shuffle(n, swap)
}
