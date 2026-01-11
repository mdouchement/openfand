# openfand

**openfand** is a list of tools used to manage several fans leveraging [OpenFanController](https://github.com/SasaKaranovic/OpenFanController) hardware.

**This tool is not recommended to manage critical fans like CPU. It may crash anytime leaving your fan unmanaged.** It is safer to let the motherboard managing them.

![curves](https://github.com/user-attachments/assets/469f0567-4acd-408f-9e82-384b09415c56)

- `openfand`\
This daemon is used to manage the fan speed based on HWMON sensors.\
It takes the higher PWM evaluated from each temperature monitored for a fan.
- `openfand show-sensors`\
List the availabe temperature sensors usable in the config file.
- `openfand show-curves`\
Displays the fans' curve in the termial. It requires your terminal to support [SIXEL](https://www.arewesixelyet.com/).
- `openfanctl monitor`\
Display a TUI monitor interface.


It currently only supports GNU/Linux.\
The [openfan](https://github.com/mdouchement/openfand/openfan) should work on several platform since [go.bug.st/serial](https://go.bug.st/serial) (https://github.com/bugst/go-serial) is cross-platform.

## Installation

Read [INSTALLATION.md](INSTALLATION.md) for the installation process.
