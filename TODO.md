# TODO

- [x] Dig into acyclic graph data structure
- [ ] 


## Vertical Slice PoC

I need to proof that:

- [x] HID event proxying can be done with sub-millisecond latency
- [x] There is a user-friendly way of configuring profiles
  - [ ] Use node based UI to represent the flow and operations
    - [x] Pre-vis, wirewrames
    - [x] Data structure
    - [ ] Live example
- [ ] An API that can be used for CLI and Web based UI
- [ ] A wide range of devices can be supported without custom drivers
- [x] Using udevadm to connect/disconnect the device
  * https://stackoverflow.com/questions/63478999/how-to-make-linux-ignore-a-keyboard-while-keeping-it-available-for-my-program-to
  * https://askubuntu.com/questions/645/how-do-you-reset-a-usb-device-from-the-command-line
- [ ] Disable keyboard inputs in TTY


* Unbind inputs: `echo 'remove' | tee -a /sys/$(dirname $(dirname $(udevadm info --query=path /dev/hidraw11)))/input/input*/event*/uevent`
* Rebind inputs: `echo 'add' | tee -a /sys/$(dirname $(dirname $(udevadm info --query=path /dev/hidraw11)))/input/input*/event*/uevent`

* libudev: https://pkg.go.dev/github.com/jochenvg/go-udev



### Stack Flow

* Agent
  * Network-isolated
  * Managed via config file
  * Reloads on config file changes
  * Secure by definition
  * Has administrator priviliges (udevadm, writing to /sys)
* API
  * Manages config file
