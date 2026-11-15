# lazypad

Switch your PC from the desk to the couch with a gamepad combo.

lazypad is a small daemon that watches your gamepads. Hold a combo and it moves
the session to a TV: it turns the TV on, switches the display and audio, closes
apps you don't want running, and opens Steam Big Picture in gamescope. Hold the
combo again and everything goes back to the desktop.

Each place you play is a **mode**. A mode is built from a few pieces:

- **display**: where the picture goes. Either a physical output (`hdmi`) or a
  headless one for streaming with Sunshine (`virtual`).
- **tv**: how to turn the screen on and off. An Android TV over adb (`adb`) or
  an Apple TV over pyatv (`appletv`).
- **audio**: optional, the PipeWire/PulseAudio card profile and sink to use.
- **gamescope**: the arguments passed to gamescope.

## Requirements

- Linux with the [mango](https://github.com/DreamMaoMao/mangowc) Wayland
  compositor (it's the only supported compositor for now)
- Steam and gamescope
- Read access to the gamepads and write access to `/dev/uinput`. The
  `steam-devices` udev rules already grant this to the logged-in user;
  otherwise add yourself to the `input` group.
- Depending on the modes you configure:
  - `adb` tv: `adb`, with network debugging enabled on the TV
  - `appletv` tv: `atvremote` from [pyatv](https://pyatv.dev)
  - `virtual` display: [Sunshine](https://github.com/LizardByte/Sunshine)
  - `audio`: `pactl`
  - MangoHud combos: [MangoHud](https://github.com/flightlessmango/MangoHud)
- `notify-send`, so errors show up as notifications

## Install

```sh
go install github.com/henriquepw/lazypad@latest
lazypad install                # writes ~/.config/systemd/user/lazypad.service
```

Or, from a clone, `make install`, which installs to `~/.local/bin` and restarts
the service.

The daemon needs the Wayland session environment, so start it from the
compositor instead of enabling it at boot. In mango's `config.conf`:

```
exec_once=systemctl --user import-environment WAYLAND_DISPLAY DISPLAY XDG_CURRENT_DESKTOP MANGO_INSTANCE_SIGNATURE && systemctl --user restart lazypad
```

If a tool like `adb` isn't in the service's `PATH`, add a drop-in at
`~/.config/systemd/user/lazypad.service.d/path.conf`:

```ini
[Service]
Environment=PATH=%h/.local/bin:%h/Android/Sdk/platform-tools:/usr/local/bin:/usr/bin
```

## Usage

```
lazypad mode <name>     enter the mode
lazypad toggle <name>   enter the mode, or go back to the desktop when already in it
lazypad desktop         go back to the desktop
lazypad status          show the current mode
```

A mode's combo does the same as `lazypad toggle <name>`. The gamepad rumbles
once when entering a mode and twice when leaving it.

## Configuration

The config lives at `~/.config/lazypad/config.toml`. Restart the service after
changing it.

```toml
[gamepad]
vendors = [0x2dc8] # USB vendor ids of the gamepads to watch (0x2dc8 is 8BitDo)

# optional: show/hide MangoHud and toggle its fps limit
[gamepad.mangohud]
combo = ["BTN_SELECT", "BTN_THUMBR"]
hold = "0.5s"

[gamepad.fps_limit]
combo = ["BTN_SELECT", "BTN_THUMBL"]
hold = "0.5s"

# apps closed when entering any mode
[close]
processes = ["discord"]
flatpaks = ["com.spotify.Client"]

[modes.bedroom]
combo = ["BTN_SELECT", "BTN_START"]
hold = "1.5s"
exit_on_tv_off = true # go back to the desktop when the TV is turned off with its remote
gamescope = ["-e", "-f", "-W", "3840", "-H", "2160", "-r", "120", "--hdr-enabled"]

[modes.bedroom.display]
type = "hdmi"
output = "HDMI-A-1"
disable = ["DP-2"]    # desk outputs, turned off while in the mode

[modes.bedroom.tv]
type = "adb"
addr = "192.168.1.182:5555"
mac = "08:9b:27:12:26:54"  # for Wake-on-LAN, needed when the TV is in deep standby
input = "com.tcl.tvinput/.TvPassThroughService/HW15" # from `adb shell dumpsys tv_input`
wake_timeout = "20s"

[modes.bedroom.audio]
card = "alsa_card.pci-0000_09_00.1"
profile = "output:hdmi-stereo-extra3"
sink = "alsa_output.pci-0000_09_00.1.hdmi-stereo-extra3"

[modes.livingroom]
combo = ["BTN_THUMBL", "BTN_THUMBR"]
hold = "1.5s"
gamescope = ["-e", "-f", "-W", "3840", "-H", "2160", "-r", "60"]

[modes.livingroom.display]
type = "virtual"
output = "STREAM"

[modes.livingroom.tv]
type = "appletv"
addr = "192.168.1.234"
app = "com.moonlight-stream.Moonlight" # bundle id opened on wake
wake_timeout = "20s"
```

### Combos

A combo fires when all of its buttons are held for `hold`. The button names
come from `linux/input-event-codes.h`: `BTN_SOUTH`, `BTN_EAST`, `BTN_NORTH`,
`BTN_WEST`, `BTN_TL`, `BTN_TR`, `BTN_TL2`, `BTN_TR2`, `BTN_SELECT`,
`BTN_START`, `BTN_MODE`, `BTN_THUMBL` and `BTN_THUMBR`. Use `evtest` to see
which ones your gamepad sends.

To find a gamepad's vendor id, run `lsusb` or check
`/sys/class/input/event*/device/id/vendor`.

### Displays

**`hdmi`** enables `output` and disables the outputs in `disable`. With the
desk outputs off, gamescope can only open on the TV. Set the TV's mode in
mango with a disabled `monitor_rule`, so it stays off until lazypad needs it:

```
monitor_rule=name:HDMI-A-1,width:3840,height:2160,refresh:120,x:2560,y:0,disable:1
```

**`virtual`** creates a headless output named `output` and focuses it, so
gamescope opens there while the desk stays as it is. The output gets mango's
default 1920x1080 unless a `monitor_rule` matches its name:

```
monitor_rule=name:STREAM,width:3840,height:2160,refresh:60,x:2560,y:0
```

### TVs

**`adb`** sends Wake-on-LAN, connects with adb and wakes the TV, then opens
`input`. Some TVs ignore the `KEYCODE_TV_INPUT_HDMI_N` keys, so lazypad opens
the input through its passthrough URI instead. List the inputs with
`adb shell dumpsys tv_input`. lazypad runs its own adb server on port 5038, so
restarting it doesn't drop your other devices.

**`appletv`** turns the Apple TV on and opens `app`. Pair the Companion
protocol once before using it:

```sh
atvremote -s 192.168.1.234 --protocol companion pair
```

`exit_on_tv_off` polls the TV every 10 seconds. It works with both types, but
each `atvremote` call starts a Python process, so leave it off for Apple TVs
unless you need it.

## Streaming with Sunshine

A `virtual` mode pairs well with Sunshine and Moonlight. Point Sunshine at the
headless output with wlroots capture, in `~/.config/sunshine/sunshine.conf`:

```
capture = wlr
output_name = 1
```

`output_name` is an index, taken from the `[wlgrab] Monitor N is ...` lines in
Sunshine's log. While the virtual output doesn't exist, Sunshine falls back to
monitor 0.

Then add an app in `apps.json` that enters and leaves the mode, so a stream
started from Moonlight works too:

```json
{
  "name": "Steam",
  "prep-cmd": [
    { "do": "lazypad mode livingroom", "undo": "lazypad desktop", "elevated": "false" }
  ]
}
```

To use the combos through Moonlight, add the vendor id of Sunshine's virtual
gamepad to `vendors` (`0x057e` with `gamepad = switch`).

## MangoHud

`gamepad.mangohud` toggles the HUD through MangoHud's control socket, which
needs this in `MangoHud.conf`:

```
control=mangohud
```

The control socket has no command for the fps limit, so `gamepad.fps_limit`
presses MangoHud's keybind instead, through a virtual keyboard. Set it to
Alt+7:

```
toggle_fps_limit=Alt_L+7
```
