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
	"github.com/neuroplastio/neio-agent/internal/hidsvc/linux"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
)

type Agent struct {
	config Config

	db        *badger.DB
	registry  *flowsvc.Registry
	configSvc *configsvc.Service
	hidSvc    *hidsvc.Service
	flowSvc   *flowsvc.Service
}

func NewAgent(config Config) (*Agent, error) {
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000000000")
	loggerConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := loggerConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	dbOptions := badger.DefaultOptions(filepath.Join(config.DataDir, "db"))
	dbOptions.Logger = &badgerLogger{l: logger.Named("badger")}

	db, err := badger.Open(dbOptions)
	// TODO: run GC on db
	if err != nil {
		return nil, fmt.Errorf("failed to open badger db: %w", err)
	}

	configSvc := configsvc.New(logger.Named("config"))
	linuxHid := linux.NewBackend(logger.Named("hid.linux"), configSvc, config.UhidConfig)
	hidSvc := hidsvc.New(db, logger.Named("hid"), time.Now, hidsvc.WithBackend("linux", linuxHid))

	registry := flowsvc.NewRegistry()
	nodes.Register(logger, registry)
	hidSvc.RegisterNodes(registry)
	actions.Register(registry)

	flowSvc := flowsvc.New(logger.Named("flow"), configSvc, config.FlowConfig, registry)
	return &Agent{
		config:    config,
		db:        db,
		registry:  registry,
		configSvc: configSvc,
		hidSvc:    hidSvc,
		flowSvc:   flowSvc,
	}, nil
}

func (a *Agent) Close() error {
	return nil
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

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return a.configSvc.Start(groupCtx)
	})
	group.Go(func() error {
		return a.hidSvc.Start(groupCtx)
	})
	group.Go(func() error {
		return a.flowSvc.Start(groupCtx)
	})

	err := group.Wait()
	if err != nil {
		return fmt.Errorf("agent failed: %w", err)
	}
	return nil
}

func (a *Agent) HID() *hidsvc.Service {
	return a.hidSvc
}
