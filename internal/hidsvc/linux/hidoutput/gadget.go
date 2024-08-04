package hidoutput

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"sync"

	"github.com/neuroplastio/neio-agent/pkg/usbhid/hiddesc"
	gadget "github.com/openstadia/go-usb-gadget"
	o "github.com/openstadia/go-usb-gadget/option"
)

type GadgetDriver struct {
	mu  sync.Mutex
	seq int
}

func NewGadgetDriver() *GadgetDriver {
	return &GadgetDriver{}
}

func (d *GadgetDriver) Outputs() []Output {
	udcs := gadget.GetUdcs()
	outputs := make([]Output, 0, len(udcs))
	for _, udc := range udcs {
		outputs = append(outputs, Output{
			ID:   udc,
			Name: fmt.Sprintf("USB Gadget %s", filepath.Base(udc)),
		})
	}
	return outputs
}

type gadgetHandler struct {
	path        string
	gadget      *gadget.Gadget
	config      *gadget.Config
	hidFunction *gadget.HidFunction
	binding     *gadget.Binding
	rw          io.ReadWriter
}

func (h *gadgetHandler) Close() error {
	if h.gadget != nil {
		h.gadget.Disable()
	}
	if h.binding != nil {
		h.binding.Close()
	}
	if h.hidFunction != nil {
		h.hidFunction.Close()
	}
	if h.config != nil {
		h.config.Close()
	}
	if h.gadget != nil {
		h.gadget.Close()
	}
	return nil
}

func (h *gadgetHandler) Read(p []byte) (n int, err error) {
	return h.rw.Read(p)
}

func (h *gadgetHandler) Write(p []byte) (n int, err error) {
	return h.rw.Write(p)
}

func (d *GadgetDriver) Open(id string, config OpenConfig) (Handler, error) {
	h := &gadgetHandler{
		path: id,
	}
	var err error
	defer func() {
		if err != nil {
			h.Close()
		}
	}()
	d.mu.Lock()
	d.seq++
	seq := d.seq
	d.mu.Unlock()
	h.gadget = gadget.CreateGadget(fmt.Sprintf("nplastio-%d", seq))
	h.gadget.SetAttrs(&gadget.GadgetAttrs{
		BcdUSB:          o.Some[uint16](0x0200),
		BDeviceClass:    o.None[uint8](),
		BDeviceSubClass: o.None[uint8](),
		BDeviceProtocol: o.None[uint8](),
		BMaxPacketSize0: o.None[uint8](),
		IdVendor:        o.Some[uint16](0x1d6b),
		IdProduct:       o.Some[uint16](0x0104),
		BcdDevice:       o.Some[uint16](0x0100),
	})

	h.gadget.SetStrs(&gadget.GadgetStrs{
		SerialNumber: "fedcba9876543210",
		Manufacturer: "Tobias Girstmair",
		Product:      "iSticktoit.net USB Device",
	}, gadget.LangUsEng)

	h.config = gadget.CreateConfig(h.gadget, fmt.Sprintf("nplastio-%d", seq), seq)
	h.config.SetAttrs(&gadget.ConfigAttrs{
		BmAttributes: o.None[uint8](),
		BMaxPower:    o.Some[uint8](250),
	})

	h.config.SetStrs(&gadget.ConfigStrs{
		Configuration: "Config 1: ECM network",
	}, gadget.LangUsEng)

	h.hidFunction = gadget.CreateHidFunction(h.gadget, fmt.Sprintf("nplastio-%d", seq))

	buf := bytes.NewBuffer(nil)
	err = hiddesc.NewDescriptorEncoder(buf, &config.ReportDescriptor).Encode()
	if err != nil {
		return nil, fmt.Errorf("failed to encode report descriptor: %w", err)
	}

	h.hidFunction.SetAttrs(&gadget.HidFunctionAttrs{
		Subclass:     0,
		Protocol:     0,
		ReportLength: uint16(config.ReportDescriptor.MaxReportSize()),
		ReportDesc:   buf.Bytes(),
	})

	h.binding = gadget.CreateBinding(h.config, h.hidFunction, h.hidFunction.Name())

	h.gadget.Enable(h.path)
	h.rw, err = h.hidFunction.GetReadWriter()
	if err != nil {
		return nil, fmt.Errorf("failed to get read writer: %w", err)
	}

	return h, nil
}
