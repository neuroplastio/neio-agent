package linux

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jochenvg/go-udev"
	"github.com/neuroplastio/neio-agent/internal/configsvc"
	"github.com/neuroplastio/neio-agent/internal/hidsvc"
	"github.com/neuroplastio/neio-agent/pkg/bits"
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

func (h *hidapiDevice) GetFeatureReport(reportID uint8) ([]byte, error) {
	buf := make([]byte, 4096) // TODO: configurable size
	buf[0] = reportID
	n, err := h.dev.GetFeatureReport(buf)
	if err != nil {
		return nil, err
	}
	if reportID == 0 {
		return buf[1:n], nil
	}
	return buf[:n], nil
}

func (h *hidapiDevice) SetFeatureReport(buf []byte) (int, error) {
	return h.dev.SendFeatureReport(buf)
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

func (b *Backend) OpenOutputDevice(id string, handler hidsvc.OutputDeviceHandler, descriptor []byte) (hidsvc.OutputDevice, error) {
	if !strings.HasPrefix(id, "uhid:") {
		return nil, fmt.Errorf("invalid output device address: %s", id)
	}
	id = strings.TrimPrefix(id, "uhid:")
	config, ok := b.uhidDevices.Load(id)
	if !ok {
		return nil, fmt.Errorf("device not found: %s", id)
	}
	b.log.Debug("Uhid desc size", zap.Any("desc", len(descriptor)))
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

	dev := &uhidDevice{
		handler: handler,
		b:       b,
		log:     b.log,
		ctx:     ctx,
		cancel:  cancel,
		dev:     uhidDev,
		events:  events,
		readCh:  make(chan []byte, 8),
	}
	go dev.run()
	return dev, nil
}

type uhidDevice struct {
	b       *Backend
	log     *zap.Logger
	ctx     context.Context
	cancel  context.CancelFunc
	dev     *uhid.Device
	events  chan uhid.Event
	handler hidsvc.OutputDeviceHandler

	readCh chan []byte
}

type UhidReportType uint8

const (
	UhidReportTypeFeature UhidReportType = 0
	UhidReportTypeOutput  UhidReportType = 1
	UhidReportTypeInput   UhidReportType = 2
)

type GetReportRequest struct {
	RequestID  uint32
	ReportID   uint8
	ReportType UhidReportType
}

const uhidReportSize = 4096

type GetReportReply struct {
	EventType uhid.EventType
	RequestID uint32
	Error     uint16
	Size      uint16
	Data      [uhidReportSize]byte
}

type SetReportRequest struct {
	RequestID  uint32
	ReportID   uint8
	ReportType UhidReportType
	Size       uint16
	Data       [uhidReportSize]byte
}

type SetReportReply struct {
	EventType uhid.EventType
	RequestID uint32
	Error     uint16
}

func (h *uhidDevice) run() {
	for {
		select {
		case <-h.ctx.Done():
			return
		case event := <-h.events:
			switch event.Type {
			case uhid.Output:
				data := make([]byte, len(event.Data))
				copy(data, event.Data)
				select {
				case h.readCh <- data:
				default:
					h.log.Warn("Dropped uhid output event")
				}
			case uhid.GetReport:
				reader := bytes.NewReader(event.Data)
				getReport := GetReportRequest{}
				err := binary.Read(reader, binary.LittleEndian, &getReport)
				if err != nil {
					h.log.Error("failed to read GetReport request", zap.Error(err))
					continue
				}
				h.log.Debug("GetReport request", zap.Any("request", getReport))
				var data []byte
				switch getReport.ReportType {
				case UhidReportTypeFeature:
					data, err = h.handler.GetFeatureReport(getReport.ReportID)
				case UhidReportTypeInput:
					data, err = h.handler.GetInputReport(getReport.ReportID)
				case UhidReportTypeOutput:
					data, err = h.handler.GetOutputReport(getReport.ReportID)
				default:
					err = fmt.Errorf("unsupported report type: %d", getReport.ReportType)
				}
				var reply GetReportReply
				if err != nil {
					h.log.Error("failed to get feature report", zap.Error(err))
					reply = GetReportReply{
						EventType: uhid.GetReportReply,
						RequestID: getReport.RequestID,
						Error:     1,
					}
				} else {
					reply = GetReportReply{
						EventType: uhid.GetReportReply,
						RequestID: getReport.RequestID,
						Size:      uint16(len(data)),
					}
					copy(reply.Data[:], data)
				}
				h.log.Debug("GetReport reply", zap.Any("data", bits.New(data, 0).String()))
				err = h.dev.WriteEvent(reply)
				if err != nil {
					h.log.Error("failed to write GetReport reply", zap.Error(err))
				}
			case uhid.SetReport:
				reader := bytes.NewReader(event.Data)
				setReport := SetReportRequest{}
				err := binary.Read(reader, binary.LittleEndian, &setReport)
				if err != nil {
					h.log.Error("failed to read SetReport request", zap.Error(err))
					continue
				}
				h.log.Debug("SetReport request", zap.Any("data", setReport.Data[:setReport.Size]))
				data := make([]byte, setReport.Size)
				copy(data, setReport.Data[:])
				var reply SetReportReply
				switch setReport.ReportType {
				case UhidReportTypeFeature:
					err = h.handler.SetFeatureReport(setReport.ReportID, data)
				case UhidReportTypeOutput:
					err = h.handler.SetOutputReport(setReport.ReportID, data)
				default:
					err = fmt.Errorf("unsupported report type: %d", setReport.ReportType)
				}
				if err != nil {
					h.log.Error("failed to set report", zap.Error(err))
					reply = SetReportReply{
						EventType: uhid.SetReportReply,
						RequestID: setReport.RequestID,
						Error:     1,
					}
				} else {
					reply = SetReportReply{
						EventType: uhid.SetReportReply,
						RequestID: setReport.RequestID,
					}
				}
				h.log.Debug("SetReport reply", zap.Any("error", reply.Error))
				err = h.dev.WriteEvent(reply)
				if err != nil {
					h.log.Error("failed to write SetReport reply", zap.Error(err))
				}
			}
		}
	}
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
		case data := <-h.readCh:
			n := copy(buf, data)
			return n, nil
		}
	}
}

func (h *uhidDevice) GetOutputReport(reportID uint8) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}
