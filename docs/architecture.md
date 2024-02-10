# Intenticle Core API v0

Intenticle's main goal is to give control over HID devices, however, there are multiple concepts that have to be considered

## Scope

### Hardware

Hardware: HID devices may be available through any type of system bus, which includes USB, bluetooth, PS/2, etc. While we are only interested in HID functions, in USB devices in particular, some non-hid features are important to support:
* Serial ports (i.e. firmware updates, vendor-specific software integrations)
* Unidentified features (i.e. vendor-specific protocols)

In MVP, intenticle should support
* USB:
    * HID
    * Proxy everything else
* Bluetooth:
  * HID

Hardware API shall be specified using the following concepts:
* Bus - the source of the device
    * USB / Bluetooth
    * Both USB and Bluetooth should support input and output
* Device - a physical device, identified by it's input bus and HW address

### Operation Modes

The default operation of Intenticle is just a virtual USB device that sends no inputs. A user can attach USB devices to tentacle, so that tentacle takes full control over them. For example, if a keyboard is attached to intenticle, all keyboard inputs will stop being recognized by the system this keyboard is attached to, and will be streamed into intenticle ingestion stream.

There must be a mechanism to detach all devices from it, so they can continue their normal operation.

### HID subsystem

Linux hidapi provides a complete toolset to interpret USB and Bluetooth HID device inputs. Today it seems that using hidapi for HID devices is a good idea, however, USB device will have to be taken out of proxy mode for this to work: USB -> libhid -> core -> encode -> USB.

Pros:
* Easy to use input event stream
* No need to go into USB intricacies, this especially benefits a user

When attaching a USB device, intenticle should:
* Identify HID interfaces in a device and use libhid as a source
* Every other non-HID interface shoud have a separate event stream

### Remapping

Remapping is a key feature of Intenticle. Intenticle allows to use multiple input devices in a single configuration. For example, Mouse5 button can be used to activate certain layer of the keyboard (actually, a global layer). Using USB and Bluetooth for communication allows truly universal input device remapping without using vendor-specific software with 100% portability.

Remapping is achieved using an event-based approach, where multiple devices emit events into shared event stream. This event stream is processed in a modifier, which is free to emit its own events for downstream modifiers. Multiple modifiers can be chained together to achieve desired effects. For example, a modifier that replaces QWERTY to Colemak acts first, and a modifier that remaps F10 to Layer 2 acts second.

#### Compatibility with Keyboard firmware

Any keyboard can be used with Intenticle, however, some limitations apply:
* (assumption) For regular keyboards `capslock`, `numlock` and `scrolllock` keys don't have key press events, but only on and off commands. They can be used as a key, however, no actions can be assigned for `hold`
* For custom keyboards, firmware should be "dumbified", so it sends regular key presses for each key. For example, layer change key can be mapped to F13, so intenticle recieves a key press. It is possible to use multiple layers of keyboard mapping software (heck, even in Intenticle itself), however, it may lead to confusing results.

### Event Stream Consumer (TBD: better term)

Event stream consumer consumes and emits events. By default, all unrecognized key presses should be sent further down the chain "as-is", but sometimes this is not desirable, and consumer can "eat" the event without emitting one. Consumer might also emit an event without any incoming events (i.e. time-based event).

As performance is very critical, some sort of event subscription system should be considered, so consumers only receive events they are interested in. One of the common use cases is "layering", so in case a subscription system is implemented, each consumer should be able to subscribe and unsubscribe from events in runtime. Or alternatively, change a preconfigured set of events it's subscribed to.

Each consumer should aim to have as minimum latency as possible, but it will likely depend on the hardware configuration of the host. 

Event-based architecture makes it very easy to implement new consumers.

#### Event Stream Consumer example: remap one key


```go
func MapOneKey(emitter EventEmitter, in event) {
    emitter.Emit(out)
}
```


### Testing

As USB and Bluetooth integrations are not ready yet, virtual devices shall be used. This streamlines testing and development.


### Modules

It is very important to have some sort of modularity, but no details of this are known.

### UI

UI is web based, using lightweight JS framework. Interacts with API using both unary and stream requests. RPC over stram is also a possibility.
