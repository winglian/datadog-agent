package api

import (
	"errors"
	"expvar"
	"fmt"
	"net/http"

	"github.com/DataDog/datadog-agent/cmd/system-probe/api/module"
	"github.com/DataDog/datadog-agent/cmd/system-probe/config"
	"github.com/DataDog/datadog-agent/cmd/system-probe/modules"
	"github.com/DataDog/datadog-agent/cmd/system-probe/utils"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	gorilla "github.com/gorilla/mux"
)

func factoryByName(name config.ModuleName) (module.Factory, error) {
	for _, module := range modules.All {
		if module.Name == name {
			return module, nil
		}
	}
	return module.Factory{}, errors.New("not found")
}

// StartServer starts the HTTP server for the system-probe, which registers endpoints from all enabled modules.
func StartServer(cfg *config.Config, moduleName config.ModuleName) error {
	factory, err := factoryByName(moduleName)
	if err != nil {
		return fmt.Errorf("failed to start module %s: %s", moduleName, err)
	}

	mux := gorilla.NewRouter()
	err = module.Register(cfg, mux, factory)
	if err != nil {
		return fmt.Errorf("failed to create system probe: %s", err)
	}

	// if a debug port is specified, we expose the default handler to that port
	if cfg.DebugPort > 0 {
		go func() {
			err := http.ListenAndServe(fmt.Sprintf("localhost:%d", cfg.DebugPort), http.DefaultServeMux)
			if err != nil && err != http.ErrServerClosed {
				log.Errorf("Error creating debug HTTP server: %v", err)
			}
		}()
	}

	// Register stats endpoint
	mux.HandleFunc(fmt.Sprintf("/debug/%s/stats", moduleName), func(w http.ResponseWriter, req *http.Request) {
		stats := module.GetStats()
		utils.WriteAsJSON(w, stats)
	})

	setupConfigHandlers(mux)

	// Module-restart handler
	if factory.RestartEnabled {
		mux.HandleFunc(fmt.Sprintf("/module/%s/restart", moduleName), restartModuleHandler).Methods("POST")
	}

	mux.Handle(fmt.Sprintf("/debug/%s/vars", moduleName), http.DefaultServeMux)

	go func() {
		err = http.Serve(conn.GetListener(), mux)
		if err != nil && err != http.ErrServerClosed {
			log.Errorf("error creating HTTP server: %s", err)
		}
	}()

	return nil
}

func init() {
	expvar.Publish("modules", expvar.Func(func() interface{} {
		return module.GetStats()
	}))
}
