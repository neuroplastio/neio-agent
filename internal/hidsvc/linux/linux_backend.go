package linux

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jochenvg/go-udev"
	"github.com/neuroplastio/neio-agent/internal/configsvc"
	"github.com/neuroplastio/neio-agent/internal/hidsvc"
	"github.com/psanford/uhid"
	"github.com/puzpuzpuz/xsync/v3"
	"github.com/sstallion/go-hid"
	"go.uber.org/zap"
)

var defaultBackendOptions = backendOptions{
	pollInterval: 1 * time.Second,
}

type backendOptions struct {
	pollInterval time.Duration
}

func WithPollInterval(d time.Duration) Option {
	return func(o *backendOptions) {
		o.pollInterval = d
	}
}

type Option func(*backendOptions)

// Backend implements the hidsvc.Backend interface for Linux Kernel.
// It uses hidapi, udev and uhid kernel modules to communicate with HID devices.
type Backend struct {
	log     *zap.Logger
	options backendOptions

	config   *configsvc.Service
	uhidPath string

	hidDevices  *xsync.MapOf[HidAddress, hid.DeviceInfo]
	uhidDevices *xsync.MapOf[string, UhidDeviceConfig]

	openedInputs *xsync.MapOf[HidAddress, *hidapiDevice]

	udev *udev.Udev

	ready chan struct{}

	publisher hidsvc.BackendPublisher
}

type HidAddress struct {
	VendorID  uint16
	ProductID uint16
	Interface int
}

func (a HidAddress) String() string {
	return fmt.Sprintf("%04x:%04x:%d", a.VendorID, a.ProductID, a.Interface)
}

func ParseHidAddress(s string) (HidAddress, error) {
	var addr HidAddress
	_, err := fmt.Sscanf(s, "%04x:%04x:%d", &addr.VendorID, &addr.ProductID, &addr.Interface)
	if err != nil {
		return HidAddress{}, err
	}
	return addr, nil
}

func NewBackend(log *zap.Logger, configSvc *configsvc.Service, uhidPath string, opts ...Option) *Backend {
	options := defaultBackendOptions
	for _, opt := range opts {
		opt(&options)
	}

	return &Backend{
		options:      options,
		log:          log,
		config:       configSvc,
		uhidPath:     uhidPath,
		ready:        make(chan struct{}),
		hidDevices:   xsync.NewMapOf[HidAddress, hid.DeviceInfo](),
		uhidDevices:  xsync.NewMapOf[string, UhidDeviceConfig](),
		openedInputs: xsync.NewMapOf[HidAddress, *hidapiDevice](),
	}
}

type UhidConfig struct {
	Uhid []UhidDeviceConfig `yaml:"uhid"`
}

type UhidDeviceConfig struct {
	ID        string `yaml:"id"`
	Name      string `yaml:"name"`
	VendorID  uint32 `yaml:"vendorId"`
	ProductID uint32 `yaml:"productId"`
}

func (b *Backend) Ready() <-chan struct{} {
	return b.ready
}

func (b *Backend) Start(ctx context.Context, publisher hidsvc.BackendPublisher) error {
	hid.Init()
	b.udev = &udev.Udev{}

	b.publisher = publisher

	b.log.Info("Starting Linux HID backend")
	select {
	case <-ctx.Done():
		return nil
	case <-b.config.Ready():
	}

	uhidConfig, err := configsvc.Register(b.config, b.uhidPath, UhidConfig{}, func(cfg UhidConfig, err error) {
		b.onUhidConfigChange(ctx, cfg, err)
	})
	if err != nil {
		return fmt.Errorf("failed to register UHID config: %w", err)
	}

	err = b.refreshUhidDevices(ctx, uhidConfig)
	if err != nil {
		return fmt.Errorf("failed to refresh UHID devices: %w", err)
	}

	err = b.refreshHidDevices(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh HID devices: %w", err)
	}

	close(b.ready)
	b.log.Info("Linux HID backend started")

	pollTicker := time.NewTicker(b.options.pollInterval)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-pollTicker.C:
			err := b.refreshHidDevices(ctx)
			if err != nil {
				b.log.Error("failed to refresh HID devices", zap.Error(err))
				continue
			}
		}
	}
}

func (b *Backend) onUhidConfigChange(ctx context.Context, cfg UhidConfig, err error) {
	if err != nil {
		b.log.Error("failed to parse UHID config", zap.Error(err))
		return
	}
	b.refreshUhidDevices(ctx, cfg)
}

func (b *Backend) refreshUhidDevices(ctx context.Context, cfg UhidConfig) error {
	newDevices := make(map[string]UhidDeviceConfig)
	for _, dev := range cfg.Uhid {
		newDevices[dev.ID] = dev
	}
	var disconnected []string
	var connected []hidsvc.BackendDevice
	b.uhidDevices.Range(func(id string, dev UhidDeviceConfig) bool {
		if _, ok := newDevices[id]; !ok {
			disconnected = append(disconnected, fmt.Sprintf("uhid:%s", id))
			b.uhidDevices.Delete(id)
			return true
		}
		delete(newDevices, id)
		return true
	})
	for id, dev := range newDevices {
		b.uhidDevices.Store(id, dev)
		connected = append(connected, hidsvc.BackendDevice{
			ID:   fmt.Sprintf("uhid:%s", id),
			Name: dev.Name,
		})
	}
	if len(connected) > 0 || len(disconnected) > 0 {
		b.publisher(ctx, hidsvc.BackendEvent{
			OutputsChanged: &hidsvc.BackendEventOutputsChanged{
				Connected:    connected,
				Disconnected: disconnected,
			},
		})
	}
	return nil
}

func (b *Backend) refreshHidDevices(ctx context.Context) error {
	newDevices, err := b.enumerateHidDevices()
	// TODO: exclude known uhid output devices
	if err != nil {
		return err
	}
	var disconnected []string
	var connected []hidsvc.BackendDevice
	b.hidDevices.Range(func(addr HidAddress, dev hid.DeviceInfo) bool {
		if _, ok := newDevices[addr]; !ok {
			disconnected = append(disconnected, addr.String())
			b.hidDevices.Delete(addr)
			return true
		}
		delete(newDevices, addr)
		return true
	})

	for addr, device := range newDevices {
		b.hidDevices.Store(addr, device)
		connected = append(connected, hidsvc.BackendDevice{
			ID:   addr.String(),
			Name: generateName(device),
		})
	}

	if len(connected) > 0 || len(disconnected) > 0 {
		b.publisher(ctx, hidsvc.BackendEvent{
			InputsChanged: &hidsvc.BackendEventInputsChanged{
				Connected:    connected,
				Disconnected: disconnected,
			},
		})
	}

	return nil
}

func generateName(device hid.DeviceInfo) string {
	var parts []string
	if device.MfrStr != "" {
		parts = append(parts, device.MfrStr)
	}
	if device.ProductStr != "" {
		parts = append(parts, device.ProductStr)
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%04x:%04x", device.VendorID, device.ProductID)
	}
	return strings.Join(parts, " ")
}

func (b *Backend) enumerateHidDevices() (map[HidAddress]hid.DeviceInfo, error) {
	devices := make(map[HidAddress]hid.DeviceInfo)
	err := hid.Enumerate(hid.VendorIDAny, hid.ProductIDAny, func(device *hid.DeviceInfo) error {
		addr := HidAddress{
			VendorID:  device.VendorID,
			ProductID: device.ProductID,
			Interface: device.InterfaceNbr,
		}
		devices[addr] = *device
		return nil
	})
	if err != nil {
		return nil, err
	}
	return devices, nil
}

func (b *Backend) OpenInputDevice(id string) (hidsvc.InputDevice, error) {
	addr, err := ParseHidAddress(id)
	if err != nil {
		return nil, err
	}

	info, ok := b.hidDevices.Load(addr)
	if !ok {
		return nil, fmt.Errorf("device not found: %s", id)
	}
	dev, err := hid.OpenPath(info.Path)
	if err != nil {
		return nil, err
	}

	handle := &hidapiDevice{
		b:    b,
		log:  b.log,
		info: info,
		dev:  dev,
	}
	return handle, nil
}

type hidapiDevice struct {
	b    *Backend
	log  *zap.Logger
	info hid.DeviceInfo
	dev  *hid.Device
}

func (h *hidapiDevice) Acquire() (func(), error) {
	hidrawDev := h.b.udev.NewDeviceFromSubsystemSysname("hidraw", filepath.Base(h.info.Path))
	if hidrawDev == nil {
		return nil, fmt.Errorf("hidraw device %s not found in udev", h.info.Path)
	}
	hidDev := hidrawDev.Parent()
	e := h.b.udev.NewEnumerate()
	e.AddMatchSubsystem("input")
	e.AddMatchParent(hidDev)
	inputs, err := e.Devices()
	if err != nil {
		return nil, fmt.Errorf("failed to list enumerate devices: %w", err)
	}
	var detachedInputs []string
	for _, inputDev := range inputs {
		syspath := inputDev.Syspath()
		if !strings.HasPrefix(filepath.Base(syspath), "event") {
			continue
		}
		err := os.WriteFile(syspath+"/uevent", []byte("remove"), 0644)
		if err != nil {
			h.log.Error("failed to detach the input", zap.Error(err))
			continue
		}
		detachedInputs = append(detachedInputs, syspath)
	}
	return func() {
		for _, input := range detachedInputs {
			err := os.WriteFile(input+"/uevent", []byte("add"), 0644)
			if err != nil {
				h.log.Error("failed to attach the input", zap.Error(err))
			}
		}
	}, nil
}

func (h *hidapiDevice) Read(buf []byte) (int, error) {
	n, err := h.dev.Read(buf)
	return n, err
}

func (h *hidapiDevice) GetInputReport(reportID uint8) ([]byte, error) {
	buf := make([]byte, 4096) // TODO: configurable size
	buf[0] = reportID
	n, err := h.dev.GetInputReport(buf)
	if err != nil {
		return nil, err
	}
	if reportID == 0 {
		return buf[1:n], nil
	}
	return buf[:n], nil
}

func (h *hidapiDevice) Close() error {
	return h.dev.Close()
}

func (h *hidapiDevice) Write(buf []byte) (int, error) {
	return h.dev.Write(buf)
}

func (h *hidapiDevice) GetReportDescriptor() ([]byte, error) {
	buf := make([]byte, 4096) // TODO: configurable size
	n, err := h.dev.GetReportDescriptor(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func (b *Backend) OpenOutputDevice(id string, descriptor []byte) (hidsvc.OutputDevice, error) {
	if !strings.HasPrefix(id, "uhid:") {
		return nil, fmt.Errorf("invalid output device address: %s", id)
	}
	id = strings.TrimPrefix(id, "uhid:")
	config, ok := b.uhidDevices.Load(id)
	if !ok {
		return nil, fmt.Errorf("device not found: %s", id)
	}
	uhidDev, err := uhid.NewDevice(id, descriptor)
	if err != nil {
		return nil, fmt.Errorf("failed to create uhid device: %w", err)
	}

	uhidDev.Data.Bus = 0x03
	uhidDev.Data.VendorID = config.VendorID
	uhidDev.Data.ProductID = config.ProductID

	ctx, cancel := context.WithCancel(context.Background())
	events, err := uhidDev.Open(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to open uhid device: %w", err)
	}

	return &uhidDevice{
		b:      b,
		log:    b.log,
		ctx:    ctx,
		cancel: cancel,
		dev:    uhidDev,
		events: events,
	}, nil
}

type uhidDevice struct {
	b      *Backend
	log    *zap.Logger
	ctx    context.Context
	cancel context.CancelFunc
	dev    *uhid.Device
	events chan uhid.Event
}

func (h *uhidDevice) Close() error {
	h.cancel()
	return h.dev.Close()
}

func (h *uhidDevice) Write(buf []byte) (int, error) {
	err := h.dev.InjectEvent(buf)
	if err != nil {
		return 0, err
	}
	return len(buf), nil
}

func (h *uhidDevice) Read(buf []byte) (int, error) {
	for {
		select {
		case <-h.ctx.Done():
			return 0, h.ctx.Err()
		case event := <-h.events:
			if event.Type != uhid.Output {
				continue
			}
			n := copy(buf, event.Data)
			return n, nil
		}
	}
}

func (h *uhidDevice) GetOutputReport(reportID uint8) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}
