package hidsvc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/goccy/go-yaml"
	"github.com/neuroplastio/neio-agent/flowapi"
	"github.com/neuroplastio/neio-agent/pkg/bus"
	"github.com/puzpuzpuz/xsync/v3"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

type Service struct {
	log        *zap.Logger
	db         *badger.DB
	options    serviceOptions
	now        func() time.Time
	ready      chan struct{}
	backendBus *BackendBus

	inputBus        *InputBus
	connectedInputs *xsync.MapOf[Address, struct{}]

	outputBus        *OutputBus
	connectedOutputs *xsync.MapOf[Address, []byte]
}

type (
	BackendBus       = bus.Bus[string, BackendEvent]
	BackendPublisher = bus.Publisher[BackendEvent]

	InputEventType uint8
	InputBusKey    struct {
		Type InputEventType
		Addr Address
	}
	InputBus         = bus.Bus[InputBusKey, InputDeviceEvent]
	InputPublisher   = bus.Publisher[InputDeviceEvent]
	InputSubscriber  = bus.Subscriber[InputBusKey, InputDeviceEvent]
	InputDeviceEvent struct{}

	OutputEventType uint8
	OutputBusKey    struct {
		Type OutputEventType
		Addr Address
	}
	OutputBus         = bus.Bus[OutputBusKey, OutputDeviceEvent]
	OutputPublisher   = bus.Publisher[OutputDeviceEvent]
	OutputSubscriber  = bus.Subscriber[OutputBusKey, OutputDeviceEvent]
	OutputDeviceEvent struct{}
)

const (
	InputConnected InputEventType = iota
	InputDisconnected
)

const (
	OutputConnected OutputEventType = iota
	OutputDisconnected
)

var defaultOptions = serviceOptions{
	backends:       make(map[string]Backend),
	backoffTimeout: 5 * time.Second,
}

type serviceOptions struct {
	backends       map[string]Backend
	backoffTimeout time.Duration
}

type Option func(*serviceOptions)

func WithBackend(name string, backend Backend) Option {
	return func(o *serviceOptions) {
		o.backends[name] = backend
	}
}

func WithBackoffTimeout(d time.Duration) Option {
	return func(o *serviceOptions) {
		o.backoffTimeout = d
	}
}

func New(db *badger.DB, log *zap.Logger, now func() time.Time, opts ...Option) *Service {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	return &Service{
		db:         db,
		log:        log,
		options:    options,
		now:        now,
		ready:      make(chan struct{}),
		backendBus: bus.NewBus[string, BackendEvent](log),

		inputBus:        bus.NewBus[InputBusKey, InputDeviceEvent](log),
		connectedInputs: xsync.NewMapOf[Address, struct{}](),

		outputBus:        bus.NewBus[OutputBusKey, OutputDeviceEvent](log),
		connectedOutputs: xsync.NewMapOf[Address, []byte](),
	}
}

func (s *Service) Start(ctx context.Context) error {
	err := s.backendBus.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start backend bus: %w", err)
	}
	select {
	case <-ctx.Done():
		return nil
	case <-s.backendBus.Ready():
	}

	err = s.inputBus.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start input bus: %w", err)
	}
	select {
	case <-ctx.Done():
		return nil
	case <-s.inputBus.Ready():
	}

	err = s.outputBus.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start output bus: %w", err)
	}
	select {
	case <-ctx.Done():
		return nil
	case <-s.outputBus.Ready():
	}

	s.consumeEvents(ctx)

	for backendID := range s.options.backends {
		go s.runBackend(ctx, backendID)
	}
	for _, backend := range s.options.backends {
		select {
		case <-ctx.Done():
			return nil
		case <-backend.Ready():
		}
	}
	close(s.ready)
	s.log.Info("Service started")
	<-ctx.Done()
	return nil
}

func (s *Service) consumeEvents(ctx context.Context) {
	go func() {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		ch := s.backendBus.Subscribe(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-ch:
				s.handleBackendEvent(ctx, msg.Key, msg.Message)
			}
		}
	}()
}

func (s *Service) Ready() <-chan struct{} {
	return s.ready
}

func (s *Service) handleBackendEvent(ctx context.Context, backendID string, event BackendEvent) error {
	switch {
	case event.InputsChanged != nil:
		s.log.Debug("devices changed", zap.String("backend", backendID))
		s.onBackendInputsChanged(ctx, backendID, event.InputsChanged)
	case event.OutputsChanged != nil:
		s.log.Debug("outputs changed", zap.String("backend", backendID))
		s.onBackendOutputsChanged(ctx, backendID, event.OutputsChanged)
	}
	return nil
}

type HidInputDevice struct {
	Address       Address       `json:"address"`
	BackendDevice BackendDevice `json:"backendDevice"`
	Name          string        `json:"name"`
	FirstSeenAt   time.Time     `json:"firstSeenAt"`
	LastSeenAt    time.Time     `json:"lastSeenAt"`
}

type HidOutputDevice struct {
	Address       Address       `json:"address"`
	BackendDevice BackendDevice `json:"backendDevice"`
	Name          string        `json:"name"`
	FirstSeenAt   time.Time     `json:"firstSeenAt"`
	LastSeenAt    time.Time     `json:"lastSeenAt"`
}

func (s *Service) onBackendInputsChanged(ctx context.Context, backendID string, event *BackendEventInputsChanged) {
	for _, id := range event.Disconnected {
		s.onInputDisconnected(ctx, backendID, id)
	}
	for _, dev := range event.Connected {
		s.onInputConnected(ctx, backendID, dev)
	}
}

func (s *Service) onInputDisconnected(ctx context.Context, backendID, id string) {
	addr := Address{Backend: backendID, ID: id}
	s.connectedInputs.Delete(addr)
	s.log.Debug("input disconnected", zap.String("backend", backendID), zap.String("id", id))
	s.inputBus.Publish(ctx, InputBusKey{
		Type: InputDisconnected,
		Addr: addr,
	}, InputDeviceEvent{})
}

func (s *Service) onInputConnected(ctx context.Context, backendID string, bdev BackendDevice) {
	dev, err := s.initializeInputDevice(backendID, bdev)
	if err != nil {
		s.log.Error("failed to initialize device", zap.Error(err))
		return
	}
	s.log.Debug("input connected", zap.String("backend", backendID), zap.String("id", dev.Address.ID), zap.String("name", dev.Name), zap.Time("firstSeenAt", dev.FirstSeenAt))
	s.connectedInputs.Store(dev.Address, struct{}{})
	s.inputBus.Publish(ctx, InputBusKey{
		Type: InputConnected,
		Addr: dev.Address,
	}, InputDeviceEvent{})
}

func (s *Service) onBackendOutputsChanged(ctx context.Context, backendID string, event *BackendEventOutputsChanged) {
	for _, id := range event.Disconnected {
		s.onOutputDisconnected(ctx, backendID, id)
	}
	for _, dev := range event.Connected {
		s.onOutputConnected(ctx, backendID, dev)
	}
}

func (s *Service) onOutputDisconnected(ctx context.Context, backendID, id string) {
	addr := Address{Backend: backendID, ID: id}
	s.connectedOutputs.Delete(addr)
	s.log.Debug("output disconnected", zap.String("backend", backendID), zap.String("id", id))
	s.outputBus.Publish(ctx, OutputBusKey{
		Type: OutputDisconnected,
		Addr: addr,
	}, OutputDeviceEvent{})
}

func (s *Service) onOutputConnected(ctx context.Context, backendID string, bdev BackendDevice) {
	dev, err := s.initializeOutputDevice(backendID, bdev)
	if err != nil {
		s.log.Error("failed to initialize device", zap.Error(err))
		return
	}
	s.log.Debug("output connected", zap.String("backend", backendID), zap.String("id", dev.Address.ID), zap.String("name", dev.Name), zap.Time("firstSeenAt", dev.FirstSeenAt))
	s.connectedOutputs.Store(dev.Address, nil)
	s.outputBus.Publish(ctx, OutputBusKey{
		Type: OutputConnected,
		Addr: dev.Address,
	}, OutputDeviceEvent{})
}

var ErrDeviceNotFound = errors.New("device not found")

func (s *Service) inputDeviceKey(address Address) []byte {
	return []byte(fmt.Sprintf("hid/inputs/%s/%s", address.Backend, address.ID))
}

func (s *Service) outputDeviceKey(address Address) []byte {
	return []byte(fmt.Sprintf("hid/outputs/%s/%s", address.Backend, address.ID))
}

func (s *Service) initializeInputDevice(backendID string, bdev BackendDevice) (HidInputDevice, error) {
	var dev HidInputDevice
	now := s.now()
	err := s.db.Update(func(txn *badger.Txn) error {
		addr := Address{Backend: backendID, ID: bdev.ID}
		key := s.inputDeviceKey(addr)
		item, err := txn.Get(key)
		switch {
		case errors.Is(err, badger.ErrKeyNotFound):
			dev = HidInputDevice{
				Name: bdev.Name,
			}
		case err != nil:
			return err
		default:
			err = item.Value(func(val []byte) error {
				return json.Unmarshal(val, &dev)
			})
			if err != nil {
				return fmt.Errorf("failed to unmarshal device: %w", err)
			}
		}
		dev.Address = addr
		dev.BackendDevice = bdev
		if dev.FirstSeenAt.IsZero() {
			dev.FirstSeenAt = now
		}
		dev.LastSeenAt = now
		b, err := json.Marshal(dev)
		if err != nil {
			return fmt.Errorf("failed to marshal device: %w", err)
		}
		return txn.Set(key, b)
	})
	if err != nil {
		return HidInputDevice{}, fmt.Errorf("failed to fetch device: %w", err)
	}
	return dev, nil
}

func (s *Service) initializeOutputDevice(backendID string, bdev BackendDevice) (HidOutputDevice, error) {
	var dev HidOutputDevice
	now := s.now()
	err := s.db.Update(func(txn *badger.Txn) error {
		addr := Address{Backend: backendID, ID: bdev.ID}
		key := s.outputDeviceKey(addr)
		item, err := txn.Get(key)
		switch {
		case errors.Is(err, badger.ErrKeyNotFound):
			dev = HidOutputDevice{
				Name: bdev.Name,
			}
		case err != nil:
			return err
		default:
			err = item.Value(func(val []byte) error {
				return json.Unmarshal(val, &dev)
			})
			if err != nil {
				return fmt.Errorf("failed to unmarshal device: %w", err)
			}
		}
		dev.Address = addr
		dev.BackendDevice = bdev
		if dev.FirstSeenAt.IsZero() {
			dev.FirstSeenAt = now
		}
		dev.LastSeenAt = now
		b, err := json.Marshal(dev)
		if err != nil {
			return fmt.Errorf("failed to marshal device: %w", err)
		}
		return txn.Set(key, b)
	})
	if err != nil {
		return HidOutputDevice{}, fmt.Errorf("failed to fetch device: %w", err)
	}
	return dev, nil
}

func (s *Service) runBackend(ctx context.Context, backendID string) {
	backend := s.options.backends[backendID]
	for {
		err := backend.Start(ctx, s.backendBus.CreatePublisher(backendID))
		if err != nil {
			s.log.Error("failed to start the backend", zap.String("backend", backendID), zap.Error(err))
		}
		t := time.NewTimer(s.options.backoffTimeout)
		// retry after backoff
		select {
		case <-ctx.Done():
			if !t.Stop() {
				<-t.C
			}
			return
		case <-t.C:
		}
	}
}

type BackendEvent struct {
	InputsChanged  *BackendEventInputsChanged
	OutputsChanged *BackendEventOutputsChanged
}

type BackendEventInputsChanged struct {
	Connected    []BackendDevice
	Disconnected []string
}

type BackendEventOutputsChanged struct {
	Connected    []BackendDevice
	Disconnected []string
}

type BackendDevice struct {
	ID   string
	Name string
}

type Backend interface {
	Start(ctx context.Context, pub BackendPublisher) error
	Ready() <-chan struct{}
	OpenInputDevice(id string) (InputDevice, error)
	OpenOutputDevice(id string, handler OutputDeviceHandler, descriptor []byte) (OutputDevice, error)
}

type Address struct {
	Backend string `yaml:"backend" json:"backend"`
	ID      string `yaml:"id" json:"id"`
}

func (a Address) String() string {
	return fmt.Sprintf("%s/%s", a.Backend, a.ID)
}

func (a Address) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func (a *Address) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var addr struct {
		Backend string `yaml:"backend"`
		ID      string `yaml:"id"`
	}
	err := json.Unmarshal(data, &addr)
	if err == nil {
		*a = Address{Backend: addr.Backend, ID: addr.ID}
		return nil
	}
	var s string
	err = json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	parsed, err := ParseAddress(s)
	if err != nil {
		return err
	}
	*a = parsed
	return nil
}

func (a Address) MarshalYAML() ([]byte, error) {
	return yaml.Marshal(a.String())
}

func (a *Address) UnmarshalYAML(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var addr struct {
		Backend string `yaml:"backend"`
		ID      string `yaml:"id"`
	}
	err := yaml.Unmarshal(data, &addr)
	if err == nil {
		*a = Address{Backend: addr.Backend, ID: addr.ID}
		return nil
	}
	var s string
	err = yaml.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	parsed, err := ParseAddress(s)
	if err != nil {
		return err
	}
	*a = parsed
	return nil
}

func ParseAddress(s string) (Address, error) {
	var addr Address
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return Address{}, fmt.Errorf("invalid address: %s", s)
	}
	addr.Backend = parts[0]
	addr.ID = strings.ReplaceAll(parts[1], ".", ":")
	return addr, nil
}

func (s *Service) ListInputDevices() ([]HidInputDevice, error) {
	var devices []HidInputDevice
	err := s.db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()
		prefix := []byte("hid/inputs/")
		for iter.Seek(prefix); iter.ValidForPrefix(prefix); iter.Next() {
			item := iter.Item()
			var dev HidInputDevice
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &dev)
			})
			if err != nil {
				return err
			}
			devices = append(devices, dev)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}
	return devices, nil
}

func (s *Service) GetInputDevice(addr Address) (HidInputDevice, error) {
	var dev HidInputDevice
	err := s.db.View(func(txn *badger.Txn) error {
		key := s.inputDeviceKey(addr)
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &dev)
		})
	})
	if err != nil {
		return HidInputDevice{}, fmt.Errorf("failed to get device: %w", err)
	}
	return dev, nil
}

func (s *Service) GetOutputDevice(addr Address) (HidOutputDevice, error) {
	var dev HidOutputDevice
	err := s.db.View(func(txn *badger.Txn) error {
		key := s.outputDeviceKey(addr)
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &dev)
		})
	})
	if err != nil {
		return HidOutputDevice{}, fmt.Errorf("failed to get device: %w", err)
	}
	return dev, nil
}

var ErrDeviceNotConnected = errors.New("device not connected")

func (s *Service) OpenInputDevice(addr Address) (InputDevice, error) {
	dev, err := s.options.backends[addr.Backend].OpenInputDevice(addr.ID)
	if err != nil {
		return nil, fmt.Errorf("error opening input device: %w", err)
	}
	return dev, nil
}

func (s *Service) OpenOutputDevice(addr Address, handler OutputDeviceHandler, descriptor []byte) (OutputDevice, error) {
	dev, err := s.options.backends[addr.Backend].OpenOutputDevice(addr.ID, handler, descriptor)
	if err != nil {
		return nil, fmt.Errorf("error opening output device: %w", err)
	}
	return dev, nil
}

func (s *Service) IsInputConnected(addr Address) bool {
	if _, ok := s.connectedInputs.Load(addr); ok {
		return true
	}
	return false
}

func (s *Service) IsOutputConnected(addr Address) bool {
	if _, ok := s.connectedOutputs.Load(addr); ok {
		return true
	}
	return false
}

func (s *Service) RegisterNodes(reg flowapi.Registry) {
	reg.MustRegisterNodeType("input", InputNodeType{
		log: s.log.Named("input"),
		hid: s,
	})
	reg.MustRegisterNodeType("output", OutputNodeType{
		log: s.log.Named("output"),
		hid: s,
	})
}

type InputDevice interface {
	io.ReadWriteCloser
	Acquire() (func(), error)
	GetReportDescriptor() ([]byte, error)
	GetInputReport(reportID uint8) ([]byte, error)
	GetFeatureReport(reportID uint8) ([]byte, error)
	SetFeatureReport(data []byte) (int, error)
}

type OutputDeviceHandler interface {
	GetInputReport(reportID uint8) ([]byte, error)
	GetOutputReport(reportID uint8) ([]byte, error)
	GetFeatureReport(reportID uint8) ([]byte, error)
	SetOutputReport(reportID uint8, data []byte) error
	SetFeatureReport(reportID uint8, data []byte) error
}

type OutputDevice interface {
	io.ReadWriteCloser
	GetOutputReport(reportID uint8) ([]byte, error)
}
