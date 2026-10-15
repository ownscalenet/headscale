package derp

import (
	"sync"
	"testing"

	"github.com/juanfont/headscale/hscontrol/types"
	"tailscale.com/tailcfg"
)

// TestGetDERPMapConcurrent exercises GetDERPMap from many goroutines at once.
// It exists to catch data races on the shared, lazily-initialized
// derpRandomInst (run with `go test -race`); it does not assert on shuffle
// output, since concurrent shuffles may legitimately interleave in any order.
func TestGetDERPMapConcurrent(t *testing.T) {
	resetDerpRandomForTesting()

	cfg := types.DERPConfig{
		DERPMap: &tailcfg.DERPMap{
			Regions: map[tailcfg.DERPRegionID]*tailcfg.DERPRegion{
				1: {
					RegionID:   1,
					RegionCode: "nyc",
					RegionName: "New York City",
					Nodes: []*tailcfg.DERPNode{
						{Name: "1a", RegionID: 1, HostName: "derp1a.tailscale.com"},
						{Name: "1b", RegionID: 1, HostName: "derp1b.tailscale.com"},
						{Name: "1c", RegionID: 1, HostName: "derp1c.tailscale.com"},
						{Name: "1d", RegionID: 1, HostName: "derp1d.tailscale.com"},
					},
				},
				2: {
					RegionID:   2,
					RegionCode: "sfo",
					RegionName: "San Francisco",
					Nodes: []*tailcfg.DERPNode{
						{Name: "2a", RegionID: 2, HostName: "derp2a.tailscale.com"},
						{Name: "2b", RegionID: 2, HostName: "derp2b.tailscale.com"},
						{Name: "2c", RegionID: 2, HostName: "derp2c.tailscale.com"},
					},
				},
			},
		},
	}

	const goroutines = 50

	var wg sync.WaitGroup

	errs := make([]error, goroutines)

	for i := range goroutines {
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()

			derpMap, err := GetDERPMap(cfg)
			if err != nil {
				errs[idx] = err

				return
			}

			if len(derpMap.Regions) != 2 {
				t.Errorf("expected 2 regions, got %d", len(derpMap.Regions))
			}

			if len(derpMap.Regions[1].Nodes) != 4 {
				t.Errorf("expected 4 nodes in region 1, got %d", len(derpMap.Regions[1].Nodes))
			}

			if len(derpMap.Regions[2].Nodes) != 3 {
				t.Errorf("expected 3 nodes in region 2, got %d", len(derpMap.Regions[2].Nodes))
			}
		}(i)
	}

	wg.Wait()

	for _, err := range errs {
		if err != nil {
			t.Errorf("GetDERPMap returned error: %v", err)
		}
	}
}
