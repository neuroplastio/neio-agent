package agent

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/neuroplastio/neio-agent/components/actions"
	"github.com/neuroplastio/neio-agent/components/nodes"
	"github.com/neuroplastio/neio-agent/internal/configsvc"
	"github.com/neuroplastio/neio-agent/internal/flowsvc"
	"github.com/neuroplastio/neio-agent/internal/hidsvc"
	"github.com/neuroplastio/neio-agent/internal/linux"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
)

type Agent struct {
	config Config
}

func NewAgent(config Config) *Agent {
	return &Agent{config: config}
}

type badgerLogger struct {
	l *zap.Logger
}

func (l badgerLogger) Errorf(msg string, args ...any) {
	l.l.Error(fmt.Sprintf(msg, args...))
}

func (l badgerLogger) Warningf(msg string, args ...any) {
	l.l.Warn(fmt.Sprintf(msg, args...))
}

func (l badgerLogger) Infof(msg string, args ...any) {
	l.l.Info(fmt.Sprintf(msg, args...))
}

func (l badgerLogger) Debugf(msg string, args ...any) {
	l.l.Debug(fmt.Sprintf(msg, args...))
}

// Run starts the agent and blocks until the context is cancelled.
// Agent startup will fail if the configuration is not valid.
// In case configuration becomes invalid after the startup, it will remain running with the last valid configuration.
func (a *Agent) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000000000")
	loggerConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := loggerConfig.Build()
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	dbOptions := badger.DefaultOptions(filepath.Join(a.config.DataDir, "db"))
	dbOptions.Logger = &badgerLogger{l: logger.Named("badger")}

	db, err := badger.Open(dbOptions)
	// TODO: run GC on db
	if err != nil {
		return fmt.Errorf("failed to open badger db: %w", err)
	}
	defer db.Close()

	configSvc := configsvc.New(logger.Named("config"))
	linuxHid := linux.NewBackend(logger.Named("hid.linux"), configSvc, a.config.UhidConfig)
	hidSvc := hidsvc.New(db, logger.Named("hid"), time.Now, hidsvc.WithBackend("linux", linuxHid))

	registry := flowsvc.NewRegistry()
	nodes.Register(logger, registry)
	hidSvc.RegisterNodes(registry)
	actions.Register(registry)

	flowSvc := flowsvc.New(logger.Named("flow"), configSvc, a.config.FlowConfig, registry)

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
