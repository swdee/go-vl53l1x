package vl53l1x

import "fmt"

// DistanceMode represents the selected ranging mode of sensor
type DistanceMode int

const (
	// Short distance mode is limited to 1.3m range in ambient and dark light
	Short DistanceMode = iota
	// Medium distance mode is limited to 2.9m in dark and 76cm in ambient light
	Medium
	// Long distance mode is limited to 3.6m in dark and 73cm in ambient light
	Long
)

// GetDistanceMode returns the sensors current DistanceMode setting
func (v *VL53L1X) GetDistanceMode() DistanceMode {
	return v.distanceMode
}

// SetDistanceMode configures the sensor for Short, Medium, or Long range.
func (v *VL53L1X) SetDistanceMode(mode DistanceMode) error {

	// save the existing timing budget.
	budget, err := v.GetMeasurementTimingBudget()

	if err != nil {
		return err
	}

	switch mode {
	case Short:
		// from VL53L1_preset_mode_standard_ranging_short_range()

		// timing config
		if err := v.writeReg(RANGE_CONFIG_VCSEL_PERIOD_A, 0x07); err != nil {
			return err
		}
		if err := v.writeReg(RANGE_CONFIG_VCSEL_PERIOD_B, 0x05); err != nil {
			return err
		}
		if err := v.writeReg(RANGE_CONFIG_VALID_PHASE_HIGH, 0x38); err != nil {
			return err
		}

		// dynamic config
		if err := v.writeReg(SD_CONFIG_WOI_SD0, 0x07); err != nil {
			return err
		}
		if err := v.writeReg(SD_CONFIG_WOI_SD1, 0x05); err != nil {
			return err
		}
		if err := v.writeReg(SD_CONFIG_INITIAL_PHASE_SD0, 6); err != nil {
			return err
		}
		if err := v.writeReg(SD_CONFIG_INITIAL_PHASE_SD1, 6); err != nil {
			return err
		}

	case Medium:
		// from VL53L1_preset_mode_standard_ranging()

		// timing config
		if err := v.writeReg(RANGE_CONFIG_VCSEL_PERIOD_A, 0x0B); err != nil {
			return err
		}
		if err := v.writeReg(RANGE_CONFIG_VCSEL_PERIOD_B, 0x09); err != nil {
			return err
		}
		if err := v.writeReg(RANGE_CONFIG_VALID_PHASE_HIGH, 0x78); err != nil {
			return err
		}

		// dynamic config
		if err := v.writeReg(SD_CONFIG_WOI_SD0, 0x0B); err != nil {
			return err
		}
		if err := v.writeReg(SD_CONFIG_WOI_SD1, 0x09); err != nil {
			return err
		}
		if err := v.writeReg(SD_CONFIG_INITIAL_PHASE_SD0, 10); err != nil {
			return err
		}
		if err := v.writeReg(SD_CONFIG_INITIAL_PHASE_SD1, 10); err != nil {
			return err
		}

	case Long:
		// from VL53L1_preset_mode_standard_ranging_long_range()

		// timing config
		if err := v.writeReg(RANGE_CONFIG_VCSEL_PERIOD_A, 0x0F); err != nil {
			return err
		}
		if err := v.writeReg(RANGE_CONFIG_VCSEL_PERIOD_B, 0x0D); err != nil {
			return err
		}
		if err := v.writeReg(RANGE_CONFIG_VALID_PHASE_HIGH, 0xB8); err != nil {
			return err
		}

		// dynamic config
		if err := v.writeReg(SD_CONFIG_WOI_SD0, 0x0F); err != nil {
			return err
		}
		if err := v.writeReg(SD_CONFIG_WOI_SD1, 0x0D); err != nil {
			return err
		}
		if err := v.writeReg(SD_CONFIG_INITIAL_PHASE_SD0, 14); err != nil {
			return err
		}
		if err := v.writeReg(SD_CONFIG_INITIAL_PHASE_SD1, 14); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unrecognized distance mode")
	}

	// reapply the timing budget
	if err := v.SetMeasurementTimingBudget(budget); err != nil {
		return err
	}

	v.distanceMode = mode
	return nil
}

// SetMeasurementTimingBudget sets the timing budget in milliseconds for one
// measurement, which is the time allowed for sensor to take one measurement
func (v *VL53L1X) SetMeasurementTimingBudget(budget uint32) error {

	// convert milliseconds to microseconds
	budgetUs := budget * 1000

	if budgetUs <= TimingGuard {
		return fmt.Errorf("timing budget too low")
	}

	rangeTimeoutUs := budgetUs - TimingGuard
	rangeTimeoutUs /= 2

	if rangeTimeoutUs > 1100000 {
		return fmt.Errorf("timing budget too high")
	}

	// Update timing for Range A CSEL Period
	vcselA, err := v.readReg(RANGE_CONFIG_VCSEL_PERIOD_A)

	if err != nil {
		return err
	}

	// Update Macro Period for Range A VCSEL Period
	macroPeriodUs := v.calcMacroPeriod(vcselA)

	// Update Phase timeout - uses Timing A
	phasecalTimeoutMclks := v.timeoutMicrosecondsToMclks(1000, macroPeriodUs)

	if phasecalTimeoutMclks > 0xFF {
		phasecalTimeoutMclks = 0xFF
	}

	if err := v.writeReg(PHASECAL_CONFIG_TIMEOUT_MACROP, uint8(phasecalTimeoutMclks)); err != nil {
		return err
	}

	// Update MM Timing A timeout
	mmTimeoutA := v.timeoutMicrosecondsToMclks(1, macroPeriodUs)

	if err := v.writeReg16Bit(MM_CONFIG_TIMEOUT_MACROP_A, v.encodeTimeout(mmTimeoutA)); err != nil {
		return err
	}

	// Update Range Timing A timeout
	rangeTimeoutA := v.timeoutMicrosecondsToMclks(rangeTimeoutUs, macroPeriodUs)

	if err := v.writeReg16Bit(RANGE_CONFIG_TIMEOUT_MACROP_A, v.encodeTimeout(rangeTimeoutA)); err != nil {
		return err
	}

	// Update timing for Range B VCSEL Period
	vcselB, err := v.readReg(RANGE_CONFIG_VCSEL_PERIOD_B)

	if err != nil {
		return err
	}

	macroPeriodUs = v.calcMacroPeriod(vcselB)

	// Update MM Timing B timeout
	mmTimeoutB := v.timeoutMicrosecondsToMclks(1, macroPeriodUs)

	if err := v.writeReg16Bit(MM_CONFIG_TIMEOUT_MACROP_B, v.encodeTimeout(mmTimeoutB)); err != nil {
		return err
	}

	// Update Range Timing B timeout
	rangeTimeoutB := v.timeoutMicrosecondsToMclks(rangeTimeoutUs, macroPeriodUs)

	if err := v.writeReg16Bit(RANGE_CONFIG_TIMEOUT_MACROP_B, v.encodeTimeout(rangeTimeoutB)); err != nil {
		return err
	}

	return nil
}

// GetMeasurementTimingBudget returns the current timing budget in milliseconds
func (v *VL53L1X) GetMeasurementTimingBudget() (uint32, error) {

	vcselA, err := v.readReg(RANGE_CONFIG_VCSEL_PERIOD_A)

	if err != nil {
		return 0, err
	}

	macroPeriodUs := v.calcMacroPeriod(vcselA)
	encoded, err := v.readReg16Bit(RANGE_CONFIG_TIMEOUT_MACROP_A)

	if err != nil {
		return 0, err
	}

	rangeTimeoutUs := v.timeoutMclksToMicroseconds(v.decodeTimeout(encoded), macroPeriodUs)
	budgetMs := (2*rangeTimeoutUs + TimingGuard) / 1000

	return budgetMs, nil
}

// decodeTimeout decode sequence step timeout in MCLKs from register value
// based on VL53L1_decode_timeout()
func (v *VL53L1X) decodeTimeout(regVal uint16) uint32 {
	return (uint32(regVal&0xFF) << (regVal >> 8)) + 1
}

// encodeTimeout encode sequence step timeout register value from timeout in MCLKs
// based on VL53L1_encode_timeout()
func (v *VL53L1X) encodeTimeout(timeoutMclks uint32) uint16 {
	var lsByte uint32
	var msByte uint16 = 0

	if timeoutMclks > 0 {
		lsByte = timeoutMclks - 1

		for lsByte&0xFFFFFF00 > 0 {
			lsByte >>= 1
			msByte++
		}

		return (msByte << 8) | uint16(lsByte&0xFF)
	}

	return 0
}

// timeoutMclksToMicroseconds convert sequence step timeout from macro periods
// to microseconds with given macro period in microseconds (12.12 format)
// based on VL53L1_calc_timeout_us()
func (v *VL53L1X) timeoutMclksToMicroseconds(timeoutMclks, macroPeriodUs uint32) uint32 {
	return ((timeoutMclks * macroPeriodUs) + 0x800) >> 12
}

// timeoutMicrosecondsToMclks convert sequence step timeout from microseconds
// to macro periods with given macro period in microseconds (12.12 format)
// based on VL53L1_calc_timeout_mclks()
func (v *VL53L1X) timeoutMicrosecondsToMclks(timeoutUs, macroPeriodUs uint32) uint32 {
	return (((timeoutUs << 12) + (macroPeriodUs >> 1)) / macroPeriodUs)
}

// calcMacroPeriod calculate macro period in microseconds (12.12 format) with
// given VCSEL period assumes fast_osc_frequency has been read and stored
// based on VL53L1_calc_macro_period_us()
func (v *VL53L1X) calcMacroPeriod(vcselPeriod uint8) uint32 {

	// Calculate PLL period in microseconds (using fast_osc_frequency)
	pllPeriodUs := (uint32(1) << 30) / uint32(v.fastOscFrequency)

	vcselPeriodPclks := (uint32(vcselPeriod) + 1) << 1

	// VL53L1_MACRO_PERIOD_VCSEL_PERIODS = 2304
	macroPeriodUs := 2304 * pllPeriodUs
	macroPeriodUs >>= 6
	macroPeriodUs *= vcselPeriodPclks
	macroPeriodUs >>= 6

	return macroPeriodUs
}
