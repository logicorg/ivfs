// Command web is the HTTP gateway the web UI calls (ADR 0007). It signs users
// in with OpenID Connect (ADR 0013), runs the finance workflows on Temporal
// as the signed-in user, and serves their results as JSON.
//
// It is configured from the environment: see config.go. With -openapi, it
// writes the API's OpenAPI document to a file and exits, without reading the
// configuration or connecting to Temporal.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/logicorg/ivfs/internal/pkg/probe"
	"github.com/logicorg/ivfs/internal/pkg/temporalenv"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/assetvalue"
	"github.com/logicorg/ivfs/web/api"
	"github.com/logicorg/ivfs/web/auth"
	"github.com/logicorg/ivfs/web/providers/manual"
	"github.com/logicorg/ivfs/web/ui"
	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080",
		"the address to listen on, behind the proxy that serves "+publicURLVariable+" over HTTPS")
	openapiPath := flag.String("openapi", "", "write the OpenAPI document to this file and exit")
	flag.Parse()

	if *openapiPath != "" {
		if err := writeOpenAPI(*openapiPath); err != nil {
			log.Fatalln("Unable to write the OpenAPI document:", err)
		}
		return
	}
	if err := serve(*addr); err != nil {
		log.Fatalln(err)
	}
}

// serve runs the gateway on addr until it is interrupted.
func serve(addr string) error {
	cfg, err := loadConfig(os.Getenv)
	if err != nil {
		return err
	}

	options, err := temporalenv.Options(os.Getenv)
	if err != nil {
		return err
	}
	options.ContextPropagators = []workflow.ContextPropagator{tenancy.Propagator()}

	// The client connects on first use, so the gateway starts while Temporal
	// is down and /api/health reports it, rather than the UI not loading.
	c, err := temporalclient.NewLazyClient(options)
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}
	defer c.Close()

	// The provider is discovered on the first sign-in, so the gateway starts
	// while it is down, as it does while Temporal is.
	idp := &auth.Provider{
		Issuer:       cfg.issuer,
		ClientID:     cfg.clientID,
		ClientSecret: cfg.clientSecret,
		PublicURL:    cfg.publicURL,
		Group:        cfg.group,
		LogoutURL:    cfg.logoutURL,
		HTTPClient:   cfg.providerClient,
	}
	signIn := auth.NewHandlers(cfg.signer, idp, auth.EnsureUserWith(c))

	// The kubelet probes with the pod's IP as Host, which the guard refuses,
	// so liveness sits in front of it. It checks nothing: a Temporal outage
	// is waited out, not fixed by a restart, and /api/health reports it.
	mux := http.NewServeMux()
	mux.Handle("GET /livez", probe.Handler(nil))
	mux.Handle("/", api.NewHandler(c, []string{cfg.publicHost}, cfg.signer, signIn, ui.Handler(), valueSources(cfg, c)))

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

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

	log.Printf("Listening on %s, serving %s", addr, cfg.publicURL)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}

// valueSources returns the configured asset value providers, the manual one
// over c.
func valueSources(cfg config, c temporalclient.Client) assetvalue.Sources {
	sources := assetvalue.Sources{Timeout: cfg.valueTimeout}
	for _, p := range cfg.valueProviders {
		var provider assetvalue.Provider = manual.New(c)
		if p.client != nil {
			provider = p.client
		}
		sources.Providers = append(sources.Providers, assetvalue.Source{Name: p.name, Provider: provider})
	}
	return sources
}

func writeOpenAPI(path string) error {
	spec, err := api.Spec()
	if err != nil {
		return err
	}
	return os.WriteFile(path, spec, 0o600)
}
