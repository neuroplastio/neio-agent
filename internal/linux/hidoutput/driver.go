package hidoutput

import (
	"io"

	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hiddesc"
)

// Driver implements an output HID device.
// Implementations: USB gadget, Linux uhid, bluetooth
// Uhid: https://github.com/psanford/uhid
// Bluetooth: https://gist.github.com/ukBaz/a47e71e7b87fbc851b27cde7d1c0fcf0
type Driver interface {
	Ouptuts() []Output
	Open(id string, config OpenConfig) (Handler, error)
}

type OpenConfig struct {
	ReportDescriptor hiddesc.ReportDescriptor
}

type Output struct {
	ID   string
	Name string
}

type Handler interface {
	io.ReadWriteCloser
}
