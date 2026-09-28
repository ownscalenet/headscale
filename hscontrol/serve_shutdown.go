package hscontrol

import (
	"context"
	"net"
	"net/http"

	"github.com/juanfont/headscale/hscontrol/types"
	"github.com/rs/zerolog/log"
)

// serveResources holds the listeners and servers started by Serve, so the
// shutdown sequence lives in one place instead of crowding Serve itself.
// Fields are nil until Serve assigns them, so shutdown can run safely even
// if startup failed partway through.
type serveResources struct {
	socketServer        *http.Server
	socketListener      net.Listener
	httpServer          *http.Server
	httpListener        net.Listener
	debugHTTPServer     *http.Server
	debugHTTPListener   net.Listener
	insecureTLSServer   *http.Server
	insecureTLSListener net.Listener
	tlsBundle           *tlsBundle
}

// shutdown gracefully stops every resource Serve started, then closes the
// database. cancel is Serve's own context.CancelFunc; calling it first
// unblocks anything still select-ing on ctx (e.g. the STUN server).
func (r *serveResources) shutdown(ctx context.Context, cancel context.CancelFunc, h *Headscale) {
	info := func(msg string) { log.Info().Msg(msg) }

	info("shutting down headscale")

	cancel()
	h.ephemeralGC.Close()

	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		types.HTTPShutdownTimeout,
	)
	defer shutdownCancel()

	if r.debugHTTPServer != nil {
		info("shutting down debug http server")

		err := r.debugHTTPServer.Shutdown(shutdownCtx)
		if err != nil {
			log.Error().Err(err).Msg("failed to shutdown prometheus http")
		}
	}

	if r.httpServer != nil {
		info("shutting down main http server")

		err := r.httpServer.Shutdown(shutdownCtx)
		if err != nil {
			log.Error().Err(err).Msg("failed to shutdown http")
		}
	}

	if r.insecureTLSServer != nil {
		info("shutting down insecure TLS server")

		err := r.insecureTLSServer.Shutdown(shutdownCtx)
		if err != nil {
			log.Error().Err(err).Msg("failed to shutdown insecure TLS server")
		}
	}

	if r.tlsBundle != nil && r.tlsBundle.ACMEServer != nil {
		info("shutting down ACME HTTP-01 challenge server")

		err := r.tlsBundle.ACMEServer.Shutdown(shutdownCtx)
		if err != nil {
			log.Error().Err(err).Msg("failed to shutdown ACME HTTP-01 server")
		}

		r.tlsBundle.ACMEListener.Close()
	}

	if r.tlsBundle != nil && r.tlsBundle.Stop != nil {
		r.tlsBundle.Stop()
	}

	if h.challenges != nil {
		info("removing pending ACME challenge records")
		h.challenges.Close() //nolint:contextcheck // each delete has its own bounded context
	}

	info("closing batcher")
	h.mapBatcher.Close()

	info("waiting for netmap stream to close")
	h.clientStreamsOpen.Wait()

	if r.socketServer != nil {
		info("shutting down api server (socket)")

		if err := r.socketServer.Shutdown(shutdownCtx); err != nil { //nolint:noinlineerr
			log.Error().Err(err).Msg("failed to shutdown socket server")
		}
	}

	if h.extraRecordMan != nil {
		h.extraRecordMan.Close()
	}

	// Stop listening (and unlink the socket if unix type):
	info("closing network listeners")

	if r.debugHTTPListener != nil {
		r.debugHTTPListener.Close()
	}

	if r.httpListener != nil {
		r.httpListener.Close()
	}

	if r.insecureTLSListener != nil {
		r.insecureTLSListener.Close()
	}

	if r.socketListener != nil {
		r.socketListener.Close()
	}

	info("closing state and database")

	err := h.state.Close()
	if err != nil {
		log.Error().Err(err).Msg("failed to close state")
	}

	log.Info().Msg("Headscale stopped")
}
