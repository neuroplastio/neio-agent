package bus

import (
	"context"
	"fmt"

	"github.com/puzpuzpuz/xsync/v3"
	"go.uber.org/zap"
)

type key interface {
	comparable
}

type message interface {
	any
}

type Message[K key, M message] struct {
	Key     K
	Message M
}

type Publisher[M message] func(ctx context.Context, msg M)
type Subscriber[K key, M message] func(ctx context.Context) <-chan Message[K, M]

type Bus[K key, M message] struct {
	log         *zap.Logger
	concurrency int
	ready       chan struct{}

	ch         chan Message[K, M]
	keySubs    *xsync.MapOf[K, map[chan Message[K, M]]struct{}]
	globalSubs *xsync.MapOf[chan Message[K, M], struct{}]

	eventCh      chan Message[K, EventType]
	eventSubs    *xsync.MapOf[chan Message[K, EventType], struct{}]
	keyEventSubs *xsync.MapOf[K, map[chan Message[K, EventType]]struct{}]
}

type EventType uint8

const (
	EventTypeSubscribed EventType = iota
	EventTypeUnsubscribed
)

func NewBus[K key, M message](logger *zap.Logger) *Bus[K, M] {
	return &Bus[K, M]{
		log:         logger,
		ready:       make(chan struct{}),
		concurrency: 1,

		ch:         make(chan Message[K, M]),
		keySubs:    xsync.NewMapOf[K, map[chan Message[K, M]]struct{}](),
		globalSubs: xsync.NewMapOf[chan Message[K, M], struct{}](),

		eventCh:      make(chan Message[K, EventType]),
		eventSubs:    xsync.NewMapOf[chan Message[K, EventType], struct{}](),
		keyEventSubs: xsync.NewMapOf[K, map[chan Message[K, EventType]]struct{}](),
	}
}

func (b *Bus[K, M]) Start(ctx context.Context) error {
	if b.concurrency < 1 {
		return fmt.Errorf("concurrency must be at least 1")
	}
	// TODO: thread pool?
	for i := 0; i < b.concurrency; i++ {
		b.startWorker(ctx)
	}
	close(b.ready)
	return nil
}

func (b *Bus[K, M]) startWorker(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-b.ch:
				b.process(ctx, msg)
			}
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-b.eventCh:
				b.processEvent(ctx, msg)
			}
		}
	}()
}

func (b *Bus[K, M]) Ready() <-chan struct{} {
	return b.ready
}

func (b *Bus[K, M]) Publish(ctx context.Context, key K, msg M) {
	select {
	case <-ctx.Done():
		return
	case b.ch <- Message[K, M]{key, msg}:
	}
}

func (b *Bus[K, M]) CreatePublisher(key K) Publisher[M] {
	return func(ctx context.Context, msg M) {
		b.Publish(ctx, key, msg)
	}
}

func (b *Bus[K, M]) CreateSubscriber(key ...K) Subscriber[K, M] {
	return func(ctx context.Context) <-chan Message[K, M] {
		return b.Subscribe(ctx, key...)
	}
}

func (b *Bus[K, M]) publishEvent(ctx context.Context, key K, e EventType) {
	select {
	case <-ctx.Done():
		return
	case b.eventCh <- Message[K, EventType]{key, e}:
	}
}

func (b *Bus[K, M]) process(ctx context.Context, msg Message[K, M]) {
	b.globalSubs.Range(func(sub chan Message[K, M], _ struct{}) bool {
		select {
		case <-ctx.Done():
			return false
		case sub <- msg:
		}
		return true
	})
	subs, ok := b.keySubs.Load(msg.Key)
	if !ok {
		return
	}
	for sub := range subs {
		select {
		case <-ctx.Done():
			return
		case sub <- msg:
		}
	}
}

func (b *Bus[K, M]) processEvent(ctx context.Context, msg Message[K, EventType]) {
	b.eventSubs.Range(func(sub chan Message[K, EventType], _ struct{}) bool {
		select {
		case <-ctx.Done():
			return false
		case sub <- msg:
		}
		return true
	})
	subs, ok := b.keyEventSubs.Load(msg.Key)
	if !ok {
		return
	}
	for sub := range subs {
		select {
		case <-ctx.Done():
			return
		case sub <- msg:
		}
	}
}

func (b *Bus[K, M]) Subscribe(ctx context.Context, key ...K) <-chan Message[K, M] {
	ch := make(chan Message[K, M])
	if len(key) == 0 {
		b.globalSubs.Store(ch, struct{}{})
		var zeroKey K
		b.publishEvent(ctx, zeroKey, EventTypeSubscribed)
		go func() {
			<-ctx.Done()
			close(ch)
			b.globalSubs.Delete(ch)
			b.publishEvent(ctx, zeroKey, EventTypeUnsubscribed)
		}()
		return ch
	}
	for _, k := range key {
		b.keySubs.Compute(k, func(val map[chan Message[K, M]]struct{}, ok bool) (map[chan Message[K, M]]struct{}, bool) {
			if !ok {
				val = make(map[chan Message[K, M]]struct{}, 64)
			}
			val[ch] = struct{}{}
			return val, false
		})
		b.publishEvent(ctx, k, EventTypeSubscribed)
	}
	go func() {
		<-ctx.Done()
		close(ch)
		for _, k := range key {
			b.keySubs.Compute(k, func(val map[chan Message[K, M]]struct{}, ok bool) (map[chan Message[K, M]]struct{}, bool) {
				delete(val, ch)
				return val, false
			})
			b.publishEvent(ctx, k, EventTypeUnsubscribed)
		}
	}()
	return ch
}

func (b *Bus[K, M]) SubscribeEvents(ctx context.Context, key ...K) <-chan Message[K, EventType] {
	ch := make(chan Message[K, EventType])
	if len(key) == 0 {
		b.eventSubs.Store(ch, struct{}{})
		go func() {
			<-ctx.Done()
			close(ch)
			b.eventSubs.Delete(ch)
		}()
		return ch
	}
	for _, k := range key {
		b.keyEventSubs.Compute(k, func(val map[chan Message[K, EventType]]struct{}, ok bool) (map[chan Message[K, EventType]]struct{}, bool) {
			if !ok {
				val = make(map[chan Message[K, EventType]]struct{}, 64)
			}
			val[ch] = struct{}{}
			return val, false
		})
	}
	go func() {
		<-ctx.Done()
		close(ch)
		for _, k := range key {
			b.keyEventSubs.Compute(k, func(val map[chan Message[K, EventType]]struct{}, ok bool) (map[chan Message[K, EventType]]struct{}, bool) {
				delete(val, ch)
				return val, false
			})
		}
	}()
	return ch
}
