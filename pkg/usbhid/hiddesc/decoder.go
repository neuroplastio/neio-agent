package hiddesc

import (
	"errors"
	"fmt"
	"io"
)

type DescriptorDecoderOption func(o *descriptorDecoderOptions)

type descriptorDecoderOptions struct {
	bufferSize int
}

func WithBufferSize(size int) DescriptorDecoderOption {
	return func(o *descriptorDecoderOptions) {
		o.bufferSize = size
	}
}

// Descriptor parser reads sequence of bytes and produces ReportDescriptor.
type DescriptorDecoder struct {
	reader  io.Reader
	err     error
	options descriptorDecoderOptions
	buf     []byte
	size    int
	state   *reportDescriptorState
}

func NewDescriptorDecoder(r io.Reader, opts ...DescriptorDecoderOption) *DescriptorDecoder {
	options := descriptorDecoderOptions{
		bufferSize: 1024,
	}
	for _, opt := range opts {
		opt(&options)
	}
	return &DescriptorDecoder{
		reader:  r,
		options: options,
		buf:     make([]byte, options.bufferSize),
	}
}

type commandFn func(state *reportDescriptorState, payload []byte) error

type globalState struct {
	usagePage       uint16
	logicalMinimum  int32
	logicalMaximum  int32
	physicalMinimum int32
	physicalMaximum int32
	unitExponent    uint32
	unit            uint32
	reportID        uint8
	reportCount     uint32
	reportSize      uint32
}

type localState struct {
	usage        []uint16
	usageMinimum uint16
	usageMaximum uint16

	designatorIndex   uint8
	designatorMinimum uint8
	designatorMaximum uint8

	stringIndex   uint8
	stringMinimum uint8
	stringMaximum uint8
}

type reportDescriptorState struct {
	global      *globalState
	local       *localState
	globalStack []globalState

	collection      *Collection
	collections     []Collection
	collectionStack []Collection

	command           Tag
	commandFn         commandFn
	commandPayloadLen uint8
	commandPayload    []byte
}

func (d *DescriptorDecoder) initState() {
	d.state = &reportDescriptorState{
		global: &globalState{},
		local:  &localState{},
	}
}

var commandMap = map[Tag]commandFn{
	TagInput:         cmdInput,
	TagOutput:        cmdOutput,
	TagFeature:       cmdFeature,
	TagCollection:    cmdCollection,
	TagEndCollection: cmdEndCollection,

	TagUsagePage:       cmdUsagePage,
	TagLogicalMinimum:  cmdLogicalMinimum,
	TagLogicalMaximum:  cmdLogicalMaximum,
	TagPhysicalMinimum: cmdPhysicalMinimum,
	TagPhysicalMaximum: cmdPhysicalMaximum,
	TagUnitExponent:    cmdUnitExponent,
	TagUnit:            cmdUnit,
	TagReportSize:      cmdReportSize,
	TagReportID:        cmdReportID,
	TagReportCount:     cmdReportCount,
	TagPush:            cmdPush,
	TagPop:             cmdPop,

	TagUsage:             cmdUsage,
	TagUsageMinimum:      cmdUsageMinimum,
	TagUsageMaximum:      cmdUsageMaximum,
	TagDesignatorIndex:   cmdDesignatorIndex,
	TagDesignatorMinimum: cmdDesignatorMinimum,
	TagDesignatorMaximum: cmdDesignatorMaximum,
	TagStringIndex:       cmdStringIndex,
	TagStringMinimum:     cmdStringMinimum,
	TagStringMaximum:     cmdStringMaximum,
	TagDelimiter:         cmdDelimiter,
}

func (d *DescriptorDecoder) parseBytes() error {
	for d.size > 0 {
		b := d.buf[0]
		d.buf = d.buf[1:]
		d.size--

		switch {
		case d.state.command == 0:
			// new command
			tag := Tag(b)
			d.state.command = tag.TagPrefix()
			d.state.commandFn = commandMap[d.state.command]
			if d.state.commandFn == nil {
				return fmt.Errorf("unknown command code: %x", b)
			}
			switch tag.PayloadSize() {
			case TagItemSize0:
				d.state.commandPayloadLen = 0
			case TagItemSize8:
				d.state.commandPayloadLen = 1
			case TagItemSize16:
				d.state.commandPayloadLen = 2
			case TagItemSize32:
				d.state.commandPayloadLen = 4
			}
			d.state.commandPayload = make([]byte, 0, d.state.commandPayloadLen)
		default:
			// adding payload to command
			d.state.commandPayload = append(d.state.commandPayload, b)
		}
		if len(d.state.commandPayload) == int(d.state.commandPayloadLen) {
			// command complete, execute and reset command state
			err := d.state.commandFn(d.state, d.state.commandPayload)
			if err != nil {
				return fmt.Errorf("failed to execute command: %w", err)
			}
			d.state.command = 0
			d.state.commandPayload = nil
			d.state.commandFn = nil
			d.state.commandPayloadLen = 0
		}
	}
	return nil
}

func (d *reportDescriptorState) descriptor() ReportDescriptor {
	return ReportDescriptor{
		Collections: d.collections,
	}
}

func (d *DescriptorDecoder) Decode() (ReportDescriptor, error) {
	if d.err != nil {
		return ReportDescriptor{}, d.err
	}
	d.initState()
	for {
		size, err := d.reader.Read(d.buf)
		if size > 0 {
			d.size = size
			err := d.parseBytes()
			if err != nil {
				return ReportDescriptor{}, err
			}
		}
		if size == 0 || errors.Is(err, io.EOF) {
			return d.state.descriptor(), nil
		}
		if err != nil {
			d.err = fmt.Errorf("failed to read descriptor: %w", err)
			return ReportDescriptor{}, d.err
		}
	}
}
