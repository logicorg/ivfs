// Command yahoo is an asset value provider (ADR 0014): it answers v1 of
// docs/specs/asset-value-provider.v1.yaml with the daily closes of Yahoo
// Finance, for assets with a ticker and for currency pairs.
//
// The spec has no authentication, so it listens where only the gateway can
// reach it.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	addr := flag.String("addr", ":8080", "the address to listen on, reachable from the gateway alone")
	upstream := flag.String("yahoo", "https://query1.finance.yahoo.com", "the base URL of Yahoo's chart API")
	flag.Parse()
	if err := serve(*addr, *upstream); err != nil {
		log.Fatalln(err)
	}
}

// serve answers on addr from the chart API at upstream until it is
// interrupted.
func serve(addr, upstream string) error {
	baseURL, err := url.Parse(upstream)
	if err != nil || (baseURL.Scheme != "https" && baseURL.Scheme != "http") || baseURL.Host == "" {
		return fmt.Errorf("-yahoo %q is not an absolute URL, such as https://query1.finance.yahoo.com", upstream)
	}
	// Bounded on its own: a request to Yahoo outlives a gateway that gave up
	// on it, so that its answer is cached.
	charts := newChartClient(baseURL, &http.Client{Timeout: 15 * time.Second})
	server := &http.Server{Addr: addr, Handler: newHandler(charts), ReadHeaderTimeout: 10 * time.Second}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Shutdown: %v", err)
		}
	}()

	log.Printf("Listening on %s, asking %s", addr, baseURL)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}
