package agent

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/neuroplastio/neuroplastio/components/actions"
	"github.com/neuroplastio/neuroplastio/components/nodes"
	"github.com/neuroplastio/neuroplastio/internal/configsvc"
	"github.com/neuroplastio/neuroplastio/internal/flowsvc"
	"github.com/neuroplastio/neuroplastio/internal/hidsvc"
	"github.com/neuroplastio/neuroplastio/internal/linux"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Agent struct {
	config Config
}

func NewAgent(config Config) *Agent {
	return &Agent{config: config}
}

// Run starts the agent and blocks until the context is cancelled.
// Agent startup will fail if the configuration is not valid.
// In case configuration becomes invalid after the startup, it will remain running with the last valid configuration.
func (a *Agent) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	db, err := badger.Open(badger.DefaultOptions(filepath.Join(a.config.DataDir, "db")))
	// TODO: run GC on db
	// TODO: inject logger
	if err != nil {
		return fmt.Errorf("failed to open badger db: %w", err)
	}
	defer db.Close()

	configSvc := configsvc.New(logger)
	linuxHid := linux.NewBackend(logger, configSvc, a.config.UhidConfig)
	hidSvc := hidsvc.New(db, logger, time.Now, hidsvc.WithBackend("linux", linuxHid))

	registry := flowsvc.NewRegistry()
	nodes.Register(logger, registry)
	hidSvc.RegisterNodes(registry)
	actions.Register(registry)

	flowSvc := flowsvc.New(logger, configSvc, a.config.FlowConfig, registry)

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return configSvc.Start(groupCtx)
	})
	group.Go(func() error {
		return hidSvc.Start(groupCtx)
	})
	group.Go(func() error {
		return flowSvc.Start(groupCtx)
	})

	err = group.Wait()
	if err != nil {
		return fmt.Errorf("agent failed: %w", err)
	}
	return nil
}
