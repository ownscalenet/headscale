package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/juanfont/headscale/hscontrol"
	"github.com/rs/zerolog/log"
)

// installReloadHandler starts a goroutine that reloads the ACL policy on
// SIGHUP until ctx is cancelled. The returned func stops the signal
// registration; callers should defer it.
func installReloadHandler(ctx context.Context, app *hscontrol.Headscale) func() {
	reload := make(chan os.Signal, 1)
	signal.Notify(reload, syscall.SIGHUP)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-reload:
				log.Info().Msg("Received SIGHUP, reloading ACL policy")

				if err := app.ReloadPolicy(); err != nil { //nolint:noinlineerr
					log.Error().Err(err).Msg("reloading policy")
				}
			}
		}
	}()

	return func() { signal.Stop(reload) }
}
