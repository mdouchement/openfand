package openfan

const (
	CommRequestCharacter  = '>'
	CommResponseCharacter = '<'
	CommEndCharacter      = '\n'
	CommAltEndCharacter   = '\r'
	CommMinMessageLength  = 3
	CommTxBufferLenASCII  = 128
	CommRxBufferLenASCII  = 128
	CommRxBufferLenHex    = CommRxBufferLenASCII/2 - 1
)

const (
	CommandFanAllGetRPM Command = 0x00
	CommandFanGetRPM    Command = 0x01
	CommandFanSetPWM    Command = 0x02
	CommandFanSetAllPWM Command = 0x03
	CommandFanSetRPM    Command = 0x04

	CommandHardwareInfo     Command = 0x05
	CommandFirmwareInfo     Command = 0x06
	CommandJumpToBootLoader Command = 0x07

	// Debug functions
	// They can be at the same time useful and dangerous in production
	CommandEMCDebugReg   Command = 0x08
	CommandEMCDebugRead  Command = 0x09
	CommandEMCDebugWrite Command = 0x0A
)

const (
	Fan1 Fan = iota // uint(0)
	Fan2
	Fan3
	Fan4
	Fan5
	Fan6
	Fan7
	Fan8
	Fan9
	Fan10
)
