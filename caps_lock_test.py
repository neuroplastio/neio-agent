import usb_hid
from adafruit_hid.keyboard import Keyboard
from adafruit_hid.keycode import Keycode
import time

# Initialize Keyboard
kbd = Keyboard(usb_hid.devices)

kbd.release(Keycode.CAPS_LOCK)
time.sleep(1)

measurements = ""

for i in range(10):
    state = kbd.led_on(Keyboard.LED_CAPS_LOCK)
    kbd.press(Keycode.CAPS_LOCK)
    start = time.monotonic_ns()
    cycles = 0
    while True:
        cycles = cycles + 1
        duration = time.monotonic_ns() - start
        if kbd.led_on(Keyboard.LED_CAPS_LOCK) != state:
            measurements = measurements + str(duration) + "\t" + str(cycles) + "\n"
            kbd.release(Keycode.CAPS_LOCK)
            time.sleep(0.2)
            break
        if duration > 25000000:
            kbd.release(Keycode.CAPS_LOCK)
            time.sleep(0.05)
            break
        
with open('measurements.tsv', 'w', encoding="utf-8") as f:
    f.write(measurements)