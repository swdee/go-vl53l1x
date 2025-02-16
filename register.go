package vl53l1x

import "fmt"

const (
	// Basic registers
	SOFT_RESET uint16 = 0x0000

	// I2C address configuration
	I2C_SLAVE_DEVICE_ADDRESS uint16 = 0x0001

	// Identification and status registers
	IDENTIFICATION_MODEL_ID uint16 = 0x010F
	FIRMWARE_SYSTEM_STATUS  uint16 = 0x00E5

	// Oscillator and calibration registers
	OSC_MEASURED_FAST_OSC_FREQUENCY uint16 = 0x0006
	RESULT_OSC_CALIBRATE_VAL        uint16 = 0x00DE

	// DSS (Dynamic SPAD Selection) and related
	DSS_CONFIG_TARGET_TOTAL_RATE_MCPS        uint16 = 0x0024
	DSS_CONFIG_MANUAL_EFFECTIVE_SPADS_SELECT uint16 = 0x0054
	DSS_CONFIG_ROI_MODE_CONTROL              uint16 = 0x004F
	DSS_CONFIG_APERTURE_ATTENUATION          uint16 = 0x0057

	// SD (Single Detector) configuration registers for ROI
	SD_CONFIG_WOI_SD0           uint16 = 0x0078
	SD_CONFIG_WOI_SD1           uint16 = 0x0079
	SD_CONFIG_INITIAL_PHASE_SD0 uint16 = 0x007A
	SD_CONFIG_INITIAL_PHASE_SD1 uint16 = 0x007B

	// I/O voltage selection register
	PAD_I2C_HV_EXTSUP_CONFIG uint16 = 0x002E

	// GPIO status
	GPIO_TIO_HV_STATUS uint16 = 0x0031

	// Sigma estimator parameters
	SIGMA_EST_EFFECTIVE_PULSE_WIDTH_NS   uint16 = 0x0036
	SIGMA_EST_EFFECTIVE_AMBIENT_WIDTH_NS uint16 = 0x0037

	// Algorithm parameters
	ALGO_CROSSTALK_COMP_VALID_HEIGHT_MM uint16 = 0x0039
	ALGO_RANGE_IGNORE_VALID_HEIGHT_MM   uint16 = 0x003E
	ALGO_RANGE_MIN_CLIP                 uint16 = 0x003F
	ALGO_CONSISTENCY_CHECK_TOLERANCE    uint16 = 0x0040

	// Timing thresholds
	SYSTEM_THRESH_RATE_HIGH uint16 = 0x0050
	SYSTEM_THRESH_RATE_LOW  uint16 = 0x0052

	// Range configuration
	RANGE_CONFIG_SIGMA_THRESH                  uint16 = 0x0064
	RANGE_CONFIG_MIN_COUNT_RATE_RTN_LIMIT_MCPS uint16 = 0x0066
	RANGE_CONFIG_VCSEL_PERIOD_A                uint16 = 0x0060
	RANGE_CONFIG_VCSEL_PERIOD_B                uint16 = 0x0063
	// For short/medium/long modes – these values determine phase validity.
	RANGE_CONFIG_VALID_PHASE_HIGH uint16 = 0x0069 // approximate

	// Grouped parameters, seed and sequence config
	SYSTEM_GROUPED_PARAMETER_HOLD_0 uint16 = 0x0071
	SYSTEM_GROUPED_PARAMETER_HOLD_1 uint16 = 0x007C
	SD_CONFIG_QUANTIFIER            uint16 = 0x007E
	SYSTEM_GROUPED_PARAMETER_HOLD   uint16 = 0x0082
	SYSTEM_SEED_CONFIG              uint16 = 0x0077
	SYSTEM_SEQUENCE_CONFIG          uint16 = 0x0081

	// ROI (region of interest) registers
	ROI_CONFIG_USER_ROI_CENTRE_SPAD              uint16 = 0x007F
	ROI_CONFIG_USER_ROI_REQUESTED_GLOBAL_XY_SIZE uint16 = 0x0080

	// Timing timeout registers
	MM_CONFIG_OUTER_OFFSET_MM      uint16 = 0x0022
	PHASECAL_CONFIG_TIMEOUT_MACROP uint16 = 0x004B
	MM_CONFIG_TIMEOUT_MACROP_A     uint16 = 0x005A
	RANGE_CONFIG_TIMEOUT_MACROP_A  uint16 = 0x005E
	MM_CONFIG_TIMEOUT_MACROP_B     uint16 = 0x005C
	RANGE_CONFIG_TIMEOUT_MACROP_B  uint16 = 0x0061

	// Calibration and override registers
	PHASECAL_CONFIG_OVERRIDE uint16 = 0x004D
	CAL_CONFIG_VCSEL_START   uint16 = 0x0047

	// VHV configuration registers (for low‑power auto mode)
	VHV_CONFIG_INIT                      uint16 = 0x000B
	VHV_CONFIG_TIMEOUT_MACROP_LOOP_BOUND uint16 = 0x0008
	PHASECAL_RESULT_VCSEL_START          uint16 = 0x00D8

	// Interrupt and mode registers
	SYSTEM_INTERRUPT_CLEAR         uint16 = 0x0086
	SYSTEM_MODE_START              uint16 = 0x0087
	SYSTEM_INTERMEASUREMENT_PERIOD uint16 = 0x006C

	// Result registers – reading range, etc.
	RESULT_RANGE_STATUS uint16 = 0x0089

	// Algorithm part-to-part range offset
	ALGO_PART_TO_PART_RANGE_OFFSET_MM uint16 = 0x001E
)

// writeReg writes a 8 bit value to the register
func (v *VL53L1X) writeReg(reg uint16, value uint8) error {

	buf := []byte{byte(reg >> 8), byte(reg), value}

	if _, err := v.bus.WriteBytes(buf); err != nil {
		return err
	}

	v.lastStatus = 0
	return nil
}

// writeReg16Bit writes a 16 bit value to the register
func (v *VL53L1X) writeReg16Bit(reg uint16, value uint16) error {

	buf := []byte{byte(reg >> 8), byte(reg), byte(value >> 8), byte(value)}

	if _, err := v.bus.WriteBytes(buf); err != nil {
		return err
	}

	v.lastStatus = 0
	return nil
}

// writeReg32Bit writes a 32 bit value to the register
func (v *VL53L1X) writeReg32Bit(reg uint16, value uint32) error {

	buf := []byte{
		byte(reg >> 8), byte(reg),
		byte(value >> 24), byte(value >> 16),
		byte(value >> 8), byte(value),
	}

	if _, err := v.bus.WriteBytes(buf); err != nil {
		return err
	}

	v.lastStatus = 0
	return nil
}

// readReg reads an 8-bit value from a 16-bit register.
func (v *VL53L1X) readReg(reg uint16) (uint8, error) {

	// Write the register address.
	addr := []byte{byte(reg >> 8), byte(reg)}

	if _, err := v.bus.WriteBytes(addr); err != nil {
		return 0, err
	}

	// Read one byte.
	buf := make([]byte, 1)
	n, err := v.bus.ReadBytes(buf)

	if err != nil {
		return 0, err
	}

	if n < 1 {
		return 0, fmt.Errorf("readReg: insufficient data")
	}

	return buf[0], nil
}

// readReg16Bit reads a 16-bit value from a 16-bit register.
func (v *VL53L1X) readReg16Bit(reg uint16) (uint16, error) {

	addr := []byte{byte(reg >> 8), byte(reg)}

	if _, err := v.bus.WriteBytes(addr); err != nil {
		return 0, err
	}

	buf := make([]byte, 2)
	n, err := v.bus.ReadBytes(buf)

	if err != nil {
		return 0, err
	}

	if n < 2 {
		return 0, fmt.Errorf("readReg16Bit: insufficient data")
	}

	return uint16(buf[0])<<8 | uint16(buf[1]), nil
}

// readReg32Bit reads a 32-bit value from a 16-bit register.
func (v *VL53L1X) readReg32Bit(reg uint16) (uint32, error) {

	addr := []byte{byte(reg >> 8), byte(reg)}

	if _, err := v.bus.WriteBytes(addr); err != nil {
		return 0, err
	}

	buf := make([]byte, 4)
	n, err := v.bus.ReadBytes(buf)

	if err != nil {
		return 0, err
	}

	if n < 4 {
		return 0, fmt.Errorf("readReg32Bit: insufficient data")
	}

	return uint32(buf[0])<<24 | uint32(buf[1])<<16 | uint32(buf[2])<<8 | uint32(buf[3]), nil
}
