// Package api is the HTTP API the web UI calls (ADR 0007). Each endpoint runs
// one finance workflow and returns its result as JSON; the workflows keep the
// validation and the rules, and the API only translates between HTTP and
// them.
//
// Each area of the API is a package of its own, such as currencies, with a
// file per endpoint, as mcp/tools is; gateway holds what they share. This
// package wires the areas together.
//
// Huma names each schema after its Go type, without the package, so the
// types an area uses as request or response bodies need names no other area
// uses, such as Currency and Category.
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/logicorg/ivfs/pkg/assetvalue"
	"github.com/logicorg/ivfs/web/api/accounts"
	"github.com/logicorg/ivfs/web/api/assets"
	"github.com/logicorg/ivfs/web/api/categories"
	"github.com/logicorg/ivfs/web/api/counterparties"
	"github.com/logicorg/ivfs/web/api/currencies"
	"github.com/logicorg/ivfs/web/api/gateway"
	"github.com/logicorg/ivfs/web/api/health"
	"github.com/logicorg/ivfs/web/api/holdings"
	"github.com/logicorg/ivfs/web/api/investments"
	"github.com/logicorg/ivfs/web/api/journal"
	"github.com/logicorg/ivfs/web/api/networth"
	"github.com/logicorg/ivfs/web/api/positions"
	"github.com/logicorg/ivfs/web/api/settings"
	"github.com/logicorg/ivfs/web/api/transactions"
	"github.com/logicorg/ivfs/web/auth"
	"go.temporal.io/sdk/client"
)

// NewHandler returns the gateway's HTTP handler: the API under /api, signing
// in under /auth, and app, the UI, for every other path. All of it is behind
// the guard against requests from other sites, and all but signing in and
// the health check need a session signer signed (ADR 0013). allowedHosts are
// the Host header values the gateway answers to, such as "ivfs.example", and
// sources the asset value providers (ADR 0014).
func NewHandler(c client.Client, allowedHosts []string, signer *auth.Signer, signIn *auth.Handlers,
	app http.Handler, sources assetvalue.Sources,
) http.Handler {
	router := chi.NewMux()
	router.Use(middleware.Recoverer, middleware.Logger, gateway.Guard(allowedHosts), auth.Require(signer))
	signIn.Register(router)
	newAPI(router, c, sources)
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		// /auth/ needs no session, so the UI is not served under it.
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/auth/") {
			gateway.WriteProblem(w, http.StatusNotFound, "no endpoint "+r.URL.Path)
			return
		}
		app.ServeHTTP(w, r)
	})
	// Only API and sign-in paths have endpoints, so only they can be served
	// with the wrong method; every other path is the UI's, through NotFound.
	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			if router.Match(chi.NewRouteContext(), method, r.URL.Path) {
				w.Header().Add("Allow", method)
			}
		}
		gateway.WriteProblem(w, http.StatusMethodNotAllowed, r.Method+" is not allowed on "+r.URL.Path)
	})
	return router
}

// Spec returns the API's OpenAPI document as indented JSON, as committed in
// openapi.json. It needs no Temporal client, since no endpoint runs.
func Spec() ([]byte, error) {
	spec, err := json.MarshalIndent(newAPI(chi.NewMux(), nil, assetvalue.Sources{}).OpenAPI(), "", "  ")
	if err != nil {
		return nil, err
	}
	return append(spec, '\n'), nil
}

func newAPI(router chi.Router, c client.Client, sources assetvalue.Sources) huma.API {
	api := humachi.New(router, gateway.NewConfig())
	health.Register(api, c)
	currencies.Register(api, c)
	currencies.RegisterRates(api, c, sources)
	categories.Register(api, c)
	counterparties.Register(api, c)
	accounts.Register(api, c)
	journal.Register(api, c)
	positions.Register(api, c)
	transactions.Register(api, c)
	assets.Register(api, c)
	assets.RegisterValues(api, c, sources)
	investments.Register(api, c)
	holdings.Register(api, c)
	settings.Register(api, c)
	networth.Register(api, c)

	gateway.PlainIntegers(api.OpenAPI())
	return api
}
