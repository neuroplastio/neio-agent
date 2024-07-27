package nodes

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/neuroplastio/neio-agent/flowapi"
	"github.com/neuroplastio/neio-agent/hidapi"
	"github.com/neuroplastio/neio-agent/hidapi/hidusage"
	"go.uber.org/zap"
)

type SplitType struct {
	log *zap.Logger
}

func (st SplitType) Descriptor() flowapi.NodeTypeDescriptor {
	return flowapi.NodeTypeDescriptor{
		DisplayName: "Split",

		UpstreamType:   flowapi.NodeLinkTypeMany,
		DownstreamType: flowapi.NodeLinkTypeMany,
	}
}

func (st SplitType) CreateNode(p flowapi.NodeProvider) (flowapi.Node, error) {
	s := &Split{
		log: st.log.With(zap.String("nodeId", p.Info().ID)),
	}
	return s, nil
}

type matchItem struct {
	matcher hidusage.Matcher
	nodeID  string
}

type Split struct {
	log        *zap.Logger
	matchItems []matchItem
}

func (s *Split) Configure(c flowapi.NodeConfigurator) error {
	var config yaml.MapSlice
	err := c.Unmarshal(&config)
	if err != nil {
		return fmt.Errorf("failed to unmarshal usage page map: %w", err)
	}
	for _, item := range config {
		nodeID, ok := item.Key.(string)
		if !ok {
			return fmt.Errorf("invalid node ID: %v", item.Key)
		}
		itemsAny, ok := item.Value.([]any)
		if !ok {
			return fmt.Errorf("invalid patterns: %v", item.Value)
		}
		patterns := make([]string, 0, len(itemsAny))
		for _, pattern := range itemsAny {
			p, ok := pattern.(string)
			if !ok {
				return fmt.Errorf("invalid pattern: %v", pattern)
			}
			patterns = append(patterns, p)
		}
		matcher, err := hidusage.NewMatcher(patterns...)
		if err != nil {
			return fmt.Errorf("invalid matchers %v: %w", patterns, err)
		}
		s.matchItems = append(s.matchItems, matchItem{
			nodeID:  nodeID,
			matcher: matcher,
		})
	}
	return nil
}

func (s *Split) Run(ctx context.Context, up flowapi.Stream, down flowapi.Stream) error {
	in := up.Subscribe(ctx)
	events := make(map[string]*hidapi.Event)
	for {
		select {
		case ev := <-in:
			for _, usage := range ev.HID.Usages() {
				nodeID := ""
				for _, item := range s.matchItems {
					if item.matcher(usage.Usage.Page(), usage.Usage.ID()) {
						event, ok := events[item.nodeID]
						if !ok {
							event = hidapi.NewEvent()
							events[item.nodeID] = event
						}
						event.AddUsage(usage)
						nodeID = item.nodeID
						break
					}
				}
				if nodeID == "" {
					s.log.Warn("no match for usage", zap.String("usage", usage.Usage.String()))
					continue
				}
				event, ok := events[nodeID]
				if !ok {
					event = hidapi.NewEvent()
					events[nodeID] = event
				}
				event.AddUsage(usage)
			}
			for nodeID, event := range events {
				down.Publish(nodeID, flowapi.Event{
					HID: event,
				})
			}
			clear(events)
		case <-ctx.Done():
			return nil
		}
	}
}
