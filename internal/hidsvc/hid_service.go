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

	inputBus            *InputBus
	checkInputListeners chan Address
	inputListeners      *xsync.MapOf[Address, int]
	openedInputs        map[Address]*openedInputDevice
	connectedInputs     *xsync.MapOf[Address, struct{}]

	outputBus            *OutputBus
	checkOutputListeners chan Address
	outputListeners      *xsync.MapOf[Address, int]
	openedOutputs        map[Address]*openedOutputDevice
	connectedOutputs     *xsync.MapOf[Address, []byte]
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
	InputDeviceEvent struct {
		Report []byte
	}

	OutputEventType uint8
	OutputBusKey    struct {
		Type OutputEventType
		Addr Address
	}
	OutputBus         = bus.Bus[OutputBusKey, OutputDeviceEvent]
	OutputPublisher   = bus.Publisher[OutputDeviceEvent]
	OutputSubscriber  = bus.Subscriber[OutputBusKey, OutputDeviceEvent]
	OutputDeviceEvent struct {
		Report []byte
	}
)

const (
	InputConnected InputEventType = iota
	InputDisconnected
	InputOpened
	InputClosed
	InputReportRead
	InputReportWrite
)

const (
	OutputConnected OutputEventType = iota
	OutputDisconnected
	OutputOpened
	OutputClosed
	OutputReportRead
	OutputReportWrite
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

		inputBus:            bus.NewBus[InputBusKey, InputDeviceEvent](log),
		checkInputListeners: make(chan Address),
		inputListeners:      xsync.NewMapOf[Address, int](),
		openedInputs:        make(map[Address]*openedInputDevice),
		connectedInputs:     xsync.NewMapOf[Address, struct{}](),

		outputBus:            bus.NewBus[OutputBusKey, OutputDeviceEvent](log),
		checkOutputListeners: make(chan Address),
		outputListeners:      xsync.NewMapOf[Address, int](),
		openedOutputs:        make(map[Address]*openedOutputDevice),
		connectedOutputs:     xsync.NewMapOf[Address, []byte](),
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
	go func() {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		ch := s.inputBus.SubscribeEvents(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-ch:
				s.manageInputListeners(event)
			}
		}
	}()
	go func() {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		ch := s.outputBus.SubscribeEvents(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-ch:
				s.manageOutputListeners(event)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case addr := <-s.checkInputListeners:
				listeners, _ := s.inputListeners.Load(addr)
				if s.IsInputConnected(addr) {
					switch {
					case listeners == 0 && s.openedInputs[addr] != nil:
						s.openedInputs[addr].Close()
						delete(s.openedInputs, addr)
						s.log.Debug("Input closed", zap.String("addr", addr.String()))
					case listeners > 0 && s.openedInputs[addr] == nil:
						dev, err := s.openInputDevice(ctx, addr)
						if err != nil {
							s.log.Error("failed to open device", zap.Error(err))
							continue
						}
						s.openedInputs[addr] = dev
						s.log.Debug("Input opened", zap.String("addr", addr.String()))
					}
				} else {
					if s.openedInputs[addr] != nil {
						s.openedInputs[addr].Close()
						delete(s.openedInputs, addr)
						s.log.Debug("Input closed", zap.String("addr", addr.String()))
					}
				}
			case addr := <-s.checkOutputListeners:
				s.log.Debug("Checking output listeners", zap.String("addr", addr.String()))
				listeners, _ := s.outputListeners.Load(addr)
				if s.IsOutputConnected(addr) {
					switch {
					case listeners == 0 && s.openedOutputs[addr] != nil:
						s.openedOutputs[addr].Close()
						delete(s.openedOutputs, addr)
						s.log.Debug("Output closed", zap.String("addr", addr.String()))
					case listeners > 0 && s.openedOutputs[addr] == nil:
						desc, _ := s.connectedOutputs.Load(addr)
						if desc == nil {
							s.log.Error("output descriptor not found", zap.String("addr", addr.String()))
							continue
						}
						dev, err := s.openOutputDevice(ctx, addr, desc)
						if err != nil {
							s.log.Error("failed to open output device", zap.Error(err))
							continue
						}
						s.openedOutputs[addr] = dev
						s.log.Debug("output opened", zap.String("addr", addr.String()))
					}
				} else {
					if s.openedOutputs[addr] != nil {
						s.openedOutputs[addr].Close()
						delete(s.openedOutputs, addr)
						s.log.Debug("Output closed", zap.String("addr", addr.String()))
					}
				}
			}
		}
	}()
}

func (s *Service) Ready() <-chan struct{} {
	return s.ready
}

type openedInputDevice struct {
	ctx    context.Context
	cancel context.CancelFunc
	addr   Address
	handle BackendInputDeviceHandle
}

func (o *openedInputDevice) Close() {
	o.cancel()
}

type openedOutputDevice struct {
	ctx    context.Context
	cancel context.CancelFunc
	addr   Address
	handle BackendOutputDeviceHandle
}

func (o *openedOutputDevice) Close() {
	o.cancel()
}

func (s *Service) openOutputDevice(ctx context.Context, addr Address, desc []byte) (*openedOutputDevice, error) {
	ctx, cancel := context.WithCancel(ctx)
	handle, err := s.options.backends[addr.Backend].OpenOutput(addr.ID, desc)
	if err != nil {
		return nil, fmt.Errorf("failed to open device: %w", err)
	}
	dev := &openedOutputDevice{
		ctx:    ctx,
		cancel: cancel,
		addr:   addr,
		handle: handle,
	}
	sub := s.outputBus.Subscribe(ctx, OutputBusKey{
		Type: OutputReportWrite,
		Addr: addr,
	})
	go func() {
		defer handle.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-sub:
				_, err := handle.Write(msg.Message.Report)
				if err != nil {
					s.log.Error("failed to write to output device", zap.Error(err))
				}
			}
		}
	}()
	go func() {
		buf := make([]byte, 4096) // TODO: calculate from the descriptor
		for {
			n, err := handle.Read(buf)
			if err != nil {
				s.log.Error("failed to read from output device", zap.Error(err))
			}
			if ctx.Err() != nil {
				return
			}
			if n > 0 {
				b := make([]byte, n)
				copy(b, buf[:n])
				s.outputBus.Publish(ctx, OutputBusKey{
					Type: OutputReportRead,
					Addr: addr,
				}, OutputDeviceEvent{Report: b})
			}
		}
	}()
	return dev, nil
}

func (s *Service) openInputDevice(ctx context.Context, addr Address) (*openedInputDevice, error) {
	ctx, cancel := context.WithCancel(ctx)
	handle, err := s.options.backends[addr.Backend].OpenInput(addr.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to open device: %w", err)
	}
	initialReport, err := handle.GetInputReport()
	if err != nil {
		s.log.Error("failed to get initial report", zap.Error(err), zap.String("addr", addr.String()))
	}
	if len(initialReport) > 0 {
		s.inputBus.Publish(ctx, InputBusKey{
			Type: InputReportRead,
			Addr: addr,
		}, InputDeviceEvent{Report: initialReport})
	}
	sub := s.inputBus.Subscribe(ctx, InputBusKey{
		Type: InputReportWrite,
		Addr: addr,
	})
	dev := &openedInputDevice{
		ctx:    ctx,
		cancel: cancel,
		addr:   addr,
		handle: handle,
	}
	go func() {
		defer handle.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-sub:
				_, err := handle.Write(msg.Message.Report)
				if err != nil {
					s.log.Error("failed to write to device", zap.Error(err))
				}
			}
		}
	}()
	go func() {
		buf := make([]byte, 4096) // TODO: calculate from the descriptor
		for {
			// TODO: read blocks even after the device is closed. This leaks a goroutine.
			// Use epoll?
			n, err := handle.Read(buf)
			if err != nil {
				s.log.Error("failed to read from device", zap.Error(err))
			}
			if ctx.Err() != nil {
				return
			}
			if n > 0 {
				b := make([]byte, n)
				copy(b, buf[:n])
				s.inputBus.Publish(ctx, InputBusKey{
					Type: InputReportRead,
					Addr: addr,
				}, InputDeviceEvent{Report: b})
			}
		}
	}()
	return dev, nil
}

func (s *Service) manageInputListeners(event bus.Message[InputBusKey, bus.EventType]) {
	if event.Key.Type != InputReportRead {
		return
	}
	switch event.Message {
	case bus.EventTypeSubscribed:
		s.inputListeners.Compute(event.Key.Addr, func(listeners int, _ bool) (int, bool) {
			return listeners + 1, false
		})
	case bus.EventTypeUnsubscribed:
		s.inputListeners.Compute(event.Key.Addr, func(listeners int, _ bool) (int, bool) {
			return listeners - 1, false
		})
	}
	s.checkInputListeners <- event.Key.Addr
}

func (s *Service) manageOutputListeners(event bus.Message[OutputBusKey, bus.EventType]) {
	if event.Key.Type != OutputReportRead {
		return
	}
	switch event.Message {
	case bus.EventTypeSubscribed:
		s.outputListeners.Compute(event.Key.Addr, func(listeners int, _ bool) (int, bool) {
			return listeners + 1, false
		})
	case bus.EventTypeUnsubscribed:
		s.outputListeners.Compute(event.Key.Addr, func(listeners int, _ bool) (int, bool) {
			return listeners - 1, false
		})
	}
	s.checkOutputListeners <- event.Key.Addr
}

func (s *Service) handleBackendEvent(ctx context.Context, backendID string, event BackendEvent) error {
	switch {
	case event.InputsChanged != nil:
		s.log.Debug("devices changed", zap.String("backend", backendID))
		s.onBackendInputsChanged(ctx, backendID, event.InputsChanged)
	case event.OutputsChanged != nil:
		s.log.Debug("outputs changed", zap.String("backend", backendID))
		s.onBackendOutputsChanged(backendID, event.OutputsChanged)
	}
	return nil
}

type HidInputDevice struct {
	Address       Address            `json:"address"`
	BackendDevice BackendInputDevice `json:"backendDevice"`
	Name          string             `json:"name"`
	FirstSeenAt   time.Time          `json:"firstSeenAt"`
	LastSeenAt    time.Time          `json:"lastSeenAt"`
}

type HidOutputDevice struct {
	Address       Address             `json:"address"`
	BackendDevice BackendOutputDevice `json:"backendDevice"`
	Name          string              `json:"name"`
	FirstSeenAt   time.Time           `json:"firstSeenAt"`
	LastSeenAt    time.Time           `json:"lastSeenAt"`
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
	s.checkInputListeners <- addr
}

func (s *Service) onInputConnected(ctx context.Context, backendID string, bdev BackendInputDevice) {
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
	s.checkInputListeners <- dev.Address
}

func (s *Service) onBackendOutputsChanged(backendID string, event *BackendEventOutputsChanged) {
	for _, id := range event.Disconnected {
		s.onOutputDisconnected(backendID, id)
	}
	for _, dev := range event.Connected {
		s.onOutputConnected(backendID, dev)
	}
}

func (s *Service) onOutputDisconnected(backendID, id string) {
	s.connectedOutputs.Delete(Address{Backend: backendID, ID: id})
	s.log.Debug("output disconnected", zap.String("backend", backendID), zap.String("id", id))
}

func (s *Service) onOutputConnected(backendID string, bdev BackendOutputDevice) {
	dev, err := s.initializeOutputDevice(backendID, bdev)
	if err != nil {
		s.log.Error("failed to initialize device", zap.Error(err))
		return
	}
	s.log.Debug("output connected", zap.String("backend", backendID), zap.String("id", dev.Address.ID), zap.String("name", dev.Name), zap.Time("firstSeenAt", dev.FirstSeenAt))
	s.connectedOutputs.Store(dev.Address, nil)
}

var ErrDeviceNotFound = errors.New("device not found")

func (s *Service) inputDeviceKey(address Address) []byte {
	return []byte(fmt.Sprintf("hid/inputs/%s/%s", address.Backend, address.ID))
}

func (s *Service) outputDeviceKey(address Address) []byte {
	return []byte(fmt.Sprintf("hid/outputs/%s/%s", address.Backend, address.ID))
}

// TODO: storage model separation
func (s *Service) initializeInputDevice(backendID string, bdev BackendInputDevice) (HidInputDevice, error) {
	var dev HidInputDevice
	now := s.now()
	err := s.db.Update(func(txn *badger.Txn) error {
		addr := Address{Backend: backendID, ID: bdev.Address}
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

func (s *Service) initializeOutputDevice(backendID string, bdev BackendOutputDevice) (HidOutputDevice, error) {
	var dev HidOutputDevice
	now := s.now()
	err := s.db.Update(func(txn *badger.Txn) error {
		addr := Address{Backend: backendID, ID: bdev.Address}
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
	Connected    []BackendInputDevice
	Disconnected []string
}

type BackendEventOutputsChanged struct {
	Connected    []BackendOutputDevice
	Disconnected []string
}

type BackendInputDevice struct {
	Address          string `json:"address"`
	Name             string `json:"name"`
	ReportDescriptor []byte `json:"reportDescriptor"`
}

type HIDReportDescriptor struct {
	Hash uint64
	Data []byte
}

type BackendOutputDevice struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type Backend interface {
	Start(ctx context.Context, pub BackendPublisher) error
	Ready() <-chan struct{}
	OpenInput(id string) (BackendInputDeviceHandle, error)
	OpenOutput(id string, desc []byte) (BackendOutputDeviceHandle, error)
}

type BackendInputDeviceHandle interface {
	io.ReadWriteCloser
	GetInputReport() ([]byte, error)
}

type BackendOutputDeviceHandle interface {
	io.ReadWriteCloser
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

func (s *Service) GetInputDeviceHandle(addr Address, alias string) (*InputDeviceHandle, error) {
	dev, err := s.GetInputDevice(addr)
	if err != nil {
		return nil, err
	}
	return &InputDeviceHandle{
		alias: alias,
		dev:   dev,
		subscriber: s.inputBus.CreateSubscriber(InputBusKey{
			Type: InputReportRead,
			Addr: addr,
		}),
		publisher: s.inputBus.CreatePublisher(InputBusKey{
			Type: InputReportWrite,
			Addr: addr,
		}),
	}, nil
}

type InputDeviceHandle struct {
	alias      string
	dev        HidInputDevice
	subscriber InputSubscriber
	publisher  InputPublisher
}

func (h *InputDeviceHandle) Start(ctx context.Context, read chan<- []byte, write <-chan []byte) error {
	ch := h.subscriber(ctx)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-ch:
				read <- msg.Message.Report
			case report := <-write:
				h.publisher(ctx, InputDeviceEvent{Report: report})
			}
		}
	}()
	<-ctx.Done()
	return nil
}

func (h *InputDeviceHandle) InputDevice() HidInputDevice {
	return h.dev
}

type OutputDeviceHandle struct {
	alias         string
	dev           HidOutputDevice
	subscriber    OutputSubscriber
	publisher     OutputPublisher
	setDescriptor func(desc []byte)
}

func (s *Service) GetOutputDeviceHandle(addr Address, alias string) (*OutputDeviceHandle, error) {
	dev, err := s.GetOutputDevice(addr)
	if err != nil {
		return nil, err
	}
	return &OutputDeviceHandle{
		alias: alias,
		dev:   dev,
		subscriber: s.outputBus.CreateSubscriber(OutputBusKey{
			Type: OutputReportRead,
			Addr: addr,
		}),
		publisher: s.outputBus.CreatePublisher(OutputBusKey{
			Type: OutputReportWrite,
			Addr: addr,
		}),
		setDescriptor: func(desc []byte) {
			s.connectedOutputs.Store(addr, desc)
		},
	}, nil
}

func (h *OutputDeviceHandle) OutputDevice() HidOutputDevice {
	return h.dev
}

func (h *OutputDeviceHandle) Start(ctx context.Context, desc []byte, read chan<- []byte, write <-chan []byte) error {
	h.setDescriptor(desc)
	ch := h.subscriber(ctx)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-ch:
				read <- msg.Message.Report
			case report := <-write:
				h.publisher(ctx, OutputDeviceEvent{Report: report})
			}
		}
	}()
	<-ctx.Done()
	return nil
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
