package openfan

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/mdouchement/logger"
	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

var (
	ErrNotFound   = errors.New("device not found/plugged")
	ErrInvalidPWM = errors.New("invalid PWM value")
)

type Controller struct {
	sync   sync.Mutex
	pname  string
	serial serial.Port
	log    logger.Logger
	wbuf   []byte
	rbuf   []byte
}

func OpenAuto() (*Controller, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, err
	}

	var port *enumerator.PortDetails
	for _, p := range ports {
		if p.VID == "2e8a" && p.PID == "000a" {
			// There are 2 entries for this match:
			// - 2e8a 000a /dev/ttyACM0 DE645CB69B6E7933
			// - 2e8a 000a /dev/ttyACM1 DE645CB69B6E7933
			// Let's take the first one which is the behavior of https://github.com/SasaKaranovic/OpenFanController
			port = p
		}
	}
	if port == nil {
		return nil, ErrNotFound
	}

	fmt.Printf("Found OpenFan on %s - PID: %s - VID: %s - SN: %s\n", port.Name, port.VID, port.PID, port.SerialNumber)
	return Open(port.Name)
}

func Open(port string) (*Controller, error) {
	c := &Controller{
		pname: port,
		wbuf:  make([]byte, CommRxBufferLenASCII),
		rbuf:  make([]byte, CommRxBufferLenASCII*32), // aka 4096 which is plenty (512 is enough)
	}

	var err error
	c.serial, err = serial.Open(port, &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	})
	if err != nil {
		return nil, err
	}

	c.serial.SetReadTimeout(200 * time.Millisecond)

	if err = c.serial.ResetInputBuffer(); err != nil {
		return nil, err
	}

	if err = c.serial.ResetOutputBuffer(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Controller) SetLogger(l logger.Logger) {
	c.log = l
}

func (c *Controller) Close() error {
	if err := c.serial.ResetInputBuffer(); err != nil {
		return err
	}

	if err := c.serial.ResetOutputBuffer(); err != nil {
		return err
	}

	return c.serial.Close()
}

func (c *Controller) Port() string {
	return c.pname
}

func (c *Controller) HardwareInfo() (*HardwareInfo, error) {
	response, err := c.Run(CommandHardwareInfo)
	if err != nil {
		return nil, fmt.Errorf("hardware_info: %w", err)
	}

	var hw HardwareInfo
	for p := range bytes.SplitSeq(response, []byte{'\r', '\n'}) {
		kv := bytes.Split(p, []byte{':'})
		if len(kv) != 2 {
			continue
		}

		switch string(kv[0]) {
		case "HW_REV":
			hw.Revision = string(kv[1])
		case "MCU":
			hw.MCU = string(kv[1])
		case "USB":
			hw.USB = string(kv[1])
		case "FAN_CHANNELS_TOTAL":
			hw.FanChannelsTotal = string(kv[1])
		case "FAN_CHANNELS_ARCH":
			hw.FanChannelsArch = string(kv[1])
		case "FAN_CHANNELS_DRIVER":
			hw.FanChannelsDriver = string(kv[1])
		}
	}

	return &hw, nil
}

func (c *Controller) FirmwareInfo() (*FirmwareInfo, error) {
	response, err := c.Run(CommandFirmwareInfo)
	if err != nil {
		return nil, fmt.Errorf("software_info: %w", err)
	}

	var fw FirmwareInfo
	for p := range bytes.SplitSeq(response, []byte{'\r', '\n'}) {
		kv := bytes.Split(p, []byte{':'})
		if len(kv) != 2 {
			continue
		}

		switch string(kv[0]) {
		case "FW_REV":
			fw.Revision = string(kv[1])
		case "PROTOCOL_VERSION":
			fw.ProtocolVersion = string(kv[1])
		}
	}

	return &fw, nil
}

func (c *Controller) RPMs() (map[Fan]uint16, error) {
	response, err := c.Run(CommandFanAllGetRPM)
	if err != nil {
		return nil, fmt.Errorf("fan_all_get_rpm: %w", err)
	}

	rpms := make(map[Fan]uint16)
	for p := range bytes.SplitSeq(response, []byte{';'}) {
		kv := bytes.Split(p, []byte{':'})
		if len(kv) != 2 {
			continue
		}

		k, err := strconv.ParseUint(string(kv[0]), 16, 8)
		if err != nil {
			return nil, fmt.Errorf("fan_all_get_rpm: %s: k: %w", kv[0], err)
		}

		v, err := strconv.ParseUint(string(kv[1]), 16, 16)
		if err != nil {
			return nil, fmt.Errorf("fan_all_get_rpm: %s: v: %w", kv[0], err)
		}

		rpms[Fan(k)] = uint16(v)
	}

	return rpms, nil
}

func (c *Controller) RPM(f Fan) (uint16, error) {
	f1, f2 := f2x(f)
	response, err := c.Run(CommandFanGetRPM, f1, f2)
	if err != nil {
		return 0, fmt.Errorf("fan_get_rpm: %w", err)
	}

	kv := bytes.Split(response, []byte{':'})
	if len(kv) != 2 {
		return 0, errors.New("fan_get_rpm: invalid response format")
	}

	v, err := strconv.ParseUint(string(kv[1]), 16, 16)
	if err != nil {
		return 0, fmt.Errorf("fan_get_rpm: %w", err)
	}

	return uint16(v), nil
}

func (c *Controller) SetRPM(f Fan, rpm uint16) (uint16, error) {
	f1, f2 := f2x(f)
	rpm1, rpm2, rpm3, rpm4 := f4x(rpm)

	response, err := c.Run(CommandFanSetRPM, f1, f2, rpm1, rpm2, rpm3, rpm4)
	if err != nil {
		return 0, fmt.Errorf("fan_set_rpm: %w", err)
	}

	kv := bytes.Split(response, []byte{':'})
	if len(kv) != 2 {
		return 0, errors.New("fan_set_rpm: invalid response format")
	}

	v, err := strconv.ParseUint(string(kv[1]), 16, 16)
	if err != nil {
		return 0, fmt.Errorf("fan_set_rpm: %w", err)
	}

	return uint16(v), nil
}

func (c *Controller) SetPWM(f Fan, pwm int) (int, error) {
	if pwm < 0 || pwm > 100 {
		return 0, ErrInvalidPWM
	}

	f1, f2 := f2x(f)
	pwm = pwm*255/100 + 1
	pwm = min(pwm, 255)
	pwm1, pwm2 := f2x(uint8(pwm))

	response, err := c.Run(CommandFanSetPWM, f1, f2, pwm1, pwm2)
	if err != nil {
		return 0, fmt.Errorf("fan_set_pwm: %w", err)
	}

	kv := bytes.Split(response, []byte{':'})
	if len(kv) != 2 {
		return 0, errors.New("fan_set_pwm: invalid response format")
	}

	v, err := strconv.ParseUint(string(kv[1]), 16, 8)
	if err != nil {
		return 0, fmt.Errorf("fan_set_pwm: %w", err)
	}

	v = v * 100 / 255
	return int(v), nil
}

func (c *Controller) SetAllPWM(pwm int) (int, error) {
	if pwm < 0 || pwm > 100 {
		return 0, ErrInvalidPWM
	}

	pwm = pwm*255/100 + 1
	pwm = min(pwm, 255)
	pwm1, pwm2 := f2x(uint8(pwm))

	response, err := c.Run(CommandFanSetAllPWM, pwm1, pwm2)
	if err != nil {
		return 0, fmt.Errorf("fan_set_all_pwm: %w", err)
	}

	v, err := strconv.ParseUint(string(response), 16, 8)
	if err != nil {
		return 0, fmt.Errorf("fan_set_all_pwm: %w", err)
	}

	v = v * 100 / 255
	return int(v), nil
}

func (c *Controller) Run(command Command, payload ...byte) ([]byte, error) {
	c.sync.Lock()
	defer c.sync.Unlock()

	l := 5 + len(payload)
	c.wbuf[0] = CommRequestCharacter
	c.wbuf[1], c.wbuf[2] = f2x(command)
	for i, b := range payload {
		c.wbuf[3+i] = b
	}
	c.wbuf[l-2] = CommAltEndCharacter // Not mandatory but let's follow how responses are formatted
	c.wbuf[l-1] = CommEndCharacter

	//

	n, err := c.serial.Write(c.wbuf[:l])
	if err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}
	if n != l && c.log != nil {
		c.log.Warnf("Invalid write: %d of %d", n, l)
	}

	//

	n = 0
	i, l := 0, 0
	for {
		n, err = c.serial.Read(c.rbuf[n*i : CommRxBufferLenASCII+n*i])
		if err != nil {
			return nil, fmt.Errorf("read: %w", err)
		}

		i++
		l += n
		if n < CommRxBufferLenASCII {
			break
		}
	}

	n = bytes.IndexByte(c.rbuf, CommResponseCharacter)
	logs := c.rbuf[:n]
	response := c.rbuf[n:l]

	//

	if c.log != nil {
		for p := range bytes.SplitSeq(logs, []byte{'\r', '\n'}) {
			if len(p) == 0 {
				continue
			}
			c.log.Debug(string(p))
		}

		for p := range bytes.SplitSeq(response, []byte{'\r', '\n'}) {
			if len(p) == 0 {
				continue
			}
			c.log.Debug(string(p))
		}
	}

	//

	if len(response) > 4 {
		response = response[4:]
	}

	response = bytes.TrimSpace(response)

	copied := make([]byte, len(response))
	copy(copied, response)
	return copied, nil
}
