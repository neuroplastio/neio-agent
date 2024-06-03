// Package configsvc provides a service for watching configuration files and notifying clients of changes.
package configsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/ghodss/yaml"
	"go.uber.org/zap"
)

type subscriber func(event fsnotify.Event)

type Service struct {
	log *zap.Logger

	watcher     *fsnotify.Watcher
	mu          sync.Mutex
	subscribers []subscriber
	running     chan struct{}
	ready       chan struct{}
}

func New(log *zap.Logger) *Service {
	svc := &Service{
		log:   log,
		ready: make(chan struct{}),
	}
	return svc
}

func (s *Service) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}
	s.watcher = watcher
	defer s.watcher.Close()
	s.running = make(chan struct{})
	defer close(s.running)
	close(s.ready)
	s.log.Info("Config service started")
	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-s.watcher.Events:
			if !ok {
				return nil
			}
			s.mu.Lock()
			for _, sub := range s.subscribers {
				sub(event)
			}
			s.mu.Unlock()
		case err, ok := <-s.watcher.Errors:
			if !ok {
				return nil
			}
			s.log.Error("Watcher error", zap.Error(err))
		}
	}
}

func (s *Service) Ready() <-chan struct{} {
	return s.ready
}

// Register registers a configuration file to watch for changes and calls fn with the new configuration.
// It returns the initial configuration and an error if the file cannot be read.
// Service instance is used as a parameter instead of the method receiver to enable generic types.
func Register[T any](s *Service, path string, def T, fn func(config T, err error)) (T, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return def, fmt.Errorf("failed to get absolute path for %s: %w", path, err)
	}
	config, err := readConfig(absPath, def)
	if err != nil {
		return def, fmt.Errorf("failed to read config: %w", err)
	}

	dir := filepath.Dir(absPath)
	err = s.watcher.Add(dir)
	if err != nil {
		return def, fmt.Errorf("failed to add path to watcher %s: %w", path, err)
	}

	s.mu.Lock()
	s.subscribers = append(s.subscribers, func(event fsnotify.Event) {
		// TODO: debounce
		if event.Name == absPath && (event.Has(fsnotify.Write) || event.Has(fsnotify.Create)) {
			newConfig, err := readConfig(absPath, def)
			fn(newConfig, err)
		}
	})
	s.mu.Unlock()

	return config, nil
}

func RegisterWriteable[T any](s *Service, path string, def T, fn func(config T, err error) error) (T, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return def, fmt.Errorf("failed to get absolute path for %s: %w", path, err)
	}
	config, err := readConfig(absPath, def)
	switch {
	case os.IsNotExist(err):
		err = writeConfig(absPath, def)
		if err != nil {
			return def, fmt.Errorf("failed to initialize config: %w", err)
		}
		config = def
	case err != nil:
		return def, fmt.Errorf("failed to read config: %w", err)
	}
	return config, nil
}

func writeConfig[T any](path string, config T) error {
	jsonB, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	yamlB, err := yaml.JSONToYAML(jsonB)
	if err != nil {
		return fmt.Errorf("failed to convert json to yaml: %w", err)
	}

	err = os.WriteFile(path, yamlB, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

func readConfig[T any](path string, def T) (T, error) {
	yamlB, err := os.ReadFile(path)
	if err != nil {
		return def, fmt.Errorf("failed to read config file: %w", err)
	}

	jsonB, err := yaml.YAMLToJSON(yamlB)
	if err != nil {
		return def, fmt.Errorf("failed to convert yaml to json: %w", err)
	}
	err = json.Unmarshal(jsonB, &def)
	if err != nil {
		return def, fmt.Errorf("failed to unmarshal json: %w", err)
	}
	return def, nil
}
