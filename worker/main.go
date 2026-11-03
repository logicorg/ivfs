// Command worker runs the Temporal workers: finance, and provisioning.
//
// worker ensure-namespace instead registers the Temporal namespace the
// workers run in, when it is missing, and exits: the Kubernetes deployment
// runs it as a Job before the workers can start.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/logicorg/ivfs/finance/registration"
	"github.com/logicorg/ivfs/internal/pkg/probe"
	"github.com/logicorg/ivfs/internal/pkg/temporalenv"
	"github.com/logicorg/ivfs/internal/pkg/temporalnamespace"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/internal/pkg/workflowrun"
	"github.com/logicorg/ivfs/pkg/registry"
	provisioningregistration "github.com/logicorg/ivfs/provisioning/registration"
	provisioningworkflows "github.com/logicorg/ivfs/provisioning/workflows"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const ensureNamespaceCommand = "ensure-namespace"

func main() {
	var err error
	if len(os.Args) > 1 && os.Args[1] == ensureNamespaceCommand {
		err = ensureNamespace(os.Args[2:])
	} else {
		err = run()
	}
	if err != nil {
		log.Fatalln(err)
	}
}

// run runs both workers until the process is interrupted.
func run() error {
	ctx := context.Background()

	cfg, err := loadConfig(os.Getenv)
	if err != nil {
		return err
	}

	// Both workers connect as the role that owns the users' databases. The
	// superuser only bootstraps it, so the worker is never handed it.
	server := registry.Server{DSN: cfg.registryDSN}
	registryDB := server.Open(cfg.registryDB)
	defer registryDB.Close()

	if err := registry.Init(ctx, registryDB); err != nil {
		return fmt.Errorf("unable to initialize the registry: %w", err)
	}

	// Each finance activity opens the database of the user it runs as, and
	// none at all for a user the registry does not know (ADR 0013).
	pools := tenancy.NewRegistryPools(registryDB, server)
	defer pools.Close()

	options := cfg.temporal
	options.ContextPropagators = []workflow.ContextPropagator{tenancy.Propagator()}
	c, err := client.Dial(options)
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}
	defer c.Close()

	if cfg.healthAddr != "" {
		health := &http.Server{
			Addr: cfg.healthAddr,
			Handler: probe.Handler(map[string]probe.Check{
				"registry": registryDB.PingContext,
				"temporal": func(ctx context.Context) error {
					_, err := c.CheckHealth(ctx, &client.CheckHealthRequest{})
					return err
				},
			}),
			ReadHeaderTimeout: 10 * time.Second,
		}
		go func() {
			if err := health.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				log.Fatalln("health server failed:", err)
			}
		}()
		defer health.Close()
	}

	// Register activities and workflows with the worker. The interceptor
	// refuses anything not run as a user, behind that user's workflow IDs
	// (ADR 0013).
	w := worker.New(c, workflowrun.TaskQueueName, worker.Options{
		Interceptors: []interceptor.WorkerInterceptor{tenancy.Interceptor()},
	})

	registration.Register(w, pools)

	// The provisioning workflows run on a worker of their own, so their
	// activities, which take a uid as an argument, are never on a queue
	// where the uid comes from the context (ADR 0013).
	pw := worker.New(c, provisioningworkflows.TaskQueueName, worker.Options{})
	provisioningregistration.Register(pw, registryDB, server)
	if err := pw.Start(); err != nil {
		return fmt.Errorf("unable to start the provisioning worker: %w", err)
	}
	defer pw.Stop()

	log.Println("Starting worker...")
	if err := w.Run(worker.InterruptCh()); err != nil {
		return fmt.Errorf("unable to start worker: %w", err)
	}
	return nil
}

// ensureNamespace registers the namespace TEMPORAL_NAMESPACE names, unless
// it exists.
func ensureNamespace(args []string) error {
	flags := flag.NewFlagSet(ensureNamespaceCommand, flag.ContinueOnError)
	retention := flags.Duration("retention", 30*24*time.Hour,
		"how long Temporal keeps a closed run's history, if the namespace is created")
	if err := flags.Parse(args); err != nil {
		return err
	}

	options, err := temporalenv.Options(os.Getenv)
	if err != nil {
		return err
	}
	c, err := client.NewNamespaceClient(options)
	if err != nil {
		return fmt.Errorf("unable to create namespace client: %w", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	created, err := temporalnamespace.Ensure(ctx, c, options.Namespace, *retention)
	if err != nil {
		return err
	}
	if created {
		log.Printf("Registered namespace %s, keeping closed runs for %s", options.Namespace, *retention)
	} else {
		log.Printf("Namespace %s exists, left as it is", options.Namespace)
	}
	return nil
}
