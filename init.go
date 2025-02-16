package vl53l1x

import (
	"fmt"
	"time"
)

// Init initialize sensor using sequence based on VL53L1_DataInit() and
// VL53L1X_StaticInit()
func (v *VL53L1X) Init() error {

	v.SetTimeout(time.Millisecond * 500)

	err := v.dataInit()

	if err != nil {
		return fmt.Errorf("Error on dataInit(), %w", err)
	}

	err = v.staticInit()

	if err != nil {
		return fmt.Errorf("Error on staticInit(), %w", err)
	}

	return v.warmSensor()
}

// warmSensor takes a single distance reading on initialization so calibration
// routines are activated.  This is done as the first reading measurement is
// slightly off
func (v *VL53L1X) warmSensor() error {

	if err := v.StartContinuous(v.timingBudget); err != nil {
		return fmt.Errorf("Start continuous failed: %v", err)
	}

	_, err := v.Read(true)

	if err != nil {
		return fmt.Errorf("dummy read failed: %w", err)
	}

	if err := v.StopContinuous(); err != nil {
		return fmt.Errorf("Stop continuous failed: %v", err)
	}

	return nil
}

// dataInit implements VL53L1X_DataInit() from C++ API code
func (v *VL53L1X) dataInit() error {

	// check model ID and module type registers (values specified in datasheet)
	model, err := v.readReg16Bit(IDENTIFICATION_MODEL_ID)

	if err != nil {
		return err
	}

	if model != 0xEACC {
		return fmt.Errorf("unexpected model ID: 0x%X", model)
	}

	// VL53L1_software_reset()
	if err := v.writeReg(SOFT_RESET, 0x00); err != nil {
		return err
	}

	time.Sleep(100 * time.Microsecond)

	if err := v.writeReg(SOFT_RESET, 0x01); err != nil {
		return err
	}

	// give it some time to boot; otherwise the sensor NACKs during the readReg()
	// call below
	time.Sleep(1 * time.Millisecond)

	// VL53L1_poll_for_boot_completion()
	v.startTimeout()

	for {
		sysStatus, err := v.readReg(FIRMWARE_SYSTEM_STATUS)

		if err != nil {
			return err
		}

		if (sysStatus&0x01) != 0 && v.lastStatus == 0 {
			break
		}

		if v.checkTimeoutExpired() {
			v.didTimeout = true
			return fmt.Errorf("timeout waiting for boot completion")
		}

		time.Sleep(1 * time.Millisecond)
	}

	// sensor uses 1V8 mode for I/O by default; switch to 2V8 mode
	val, err := v.readReg(PAD_I2C_HV_EXTSUP_CONFIG)

	if err != nil {
		return err
	}

	if err := v.writeReg(PAD_I2C_HV_EXTSUP_CONFIG, val|0x01); err != nil {
		return err
	}

	// Store oscillator info.
	fosc, err := v.readReg16Bit(OSC_MEASURED_FAST_OSC_FREQUENCY)

	if err != nil {
		return err
	}

	v.fastOscFrequency = fosc

	oscCal, err := v.readReg16Bit(RESULT_OSC_CALIBRATE_VAL)

	if err != nil {
		return err
	}

	v.oscCalibrateVal = oscCal

	return nil
}

// staticInit implements VL53L1X_StaticInit() begin from C++ API code
func (v *VL53L1X) staticInit() error {

	// Note that the API does not actually apply the configuration settings below
	// when VL53L1_StaticInit() is called: it keeps a copy of the sensor's
	// register contents in memory and doesn't actually write them until a
	// measurement is started. Writing the configuration here means we don't have
	// to keep it all in memory and avoids a lot of redundant writes later.

	// Static initialization (configuration settings).
	if err := v.writeReg16Bit(DSS_CONFIG_TARGET_TOTAL_RATE_MCPS, TargetRate); err != nil {
		return err
	}

	if err := v.writeReg(GPIO_TIO_HV_STATUS, 0x02); err != nil {
		return err
	}

	if err := v.writeReg(SIGMA_EST_EFFECTIVE_PULSE_WIDTH_NS, 8); err != nil {
		return err
	}

	if err := v.writeReg(SIGMA_EST_EFFECTIVE_AMBIENT_WIDTH_NS, 16); err != nil {
		return err
	}

	if err := v.writeReg(ALGO_CROSSTALK_COMP_VALID_HEIGHT_MM, 0x01); err != nil {
		return err
	}

	if err := v.writeReg(ALGO_RANGE_IGNORE_VALID_HEIGHT_MM, 0xFF); err != nil {
		return err
	}

	if err := v.writeReg(ALGO_RANGE_MIN_CLIP, 0); err != nil {
		return err
	}

	if err := v.writeReg(ALGO_CONSISTENCY_CHECK_TOLERANCE, 2); err != nil {
		return err
	}

	//  general config
	if err := v.writeReg16Bit(SYSTEM_THRESH_RATE_HIGH, 0x0000); err != nil {
		return err
	}

	if err := v.writeReg16Bit(SYSTEM_THRESH_RATE_LOW, 0x0000); err != nil {
		return err
	}

	if err := v.writeReg(DSS_CONFIG_APERTURE_ATTENUATION, 0x38); err != nil {
		return err
	}

	// timing config
	// most of these settings will be determined later by distance and timing
	// budget configuration
	if err := v.writeReg16Bit(RANGE_CONFIG_SIGMA_THRESH, 360); err != nil {
		return err
	}

	if err := v.writeReg16Bit(RANGE_CONFIG_MIN_COUNT_RATE_RTN_LIMIT_MCPS, 192); err != nil {
		return err
	}

	// dynamic config
	if err := v.writeReg(SYSTEM_GROUPED_PARAMETER_HOLD_0, 0x01); err != nil {
		return err
	}

	if err := v.writeReg(SYSTEM_GROUPED_PARAMETER_HOLD_1, 0x01); err != nil {
		return err
	}

	if err := v.writeReg(SD_CONFIG_QUANTIFIER, 2); err != nil {
		return err
	}

	// from VL53L1_preset_mode_timed_ranging_*
	if err := v.writeReg(SYSTEM_GROUPED_PARAMETER_HOLD, 0x00); err != nil {
		return err
	}

	if err := v.writeReg(SYSTEM_SEED_CONFIG, 1); err != nil {
		return err
	}

	// from VL53L1_config_low_power_auto_mode
	if err := v.writeReg(SYSTEM_SEQUENCE_CONFIG, 0x8B); err != nil {
		return err
	}

	// Write manual effective spads (200 << 8)
	if err := v.writeReg16Bit(DSS_CONFIG_MANUAL_EFFECTIVE_SPADS_SELECT, 200<<8); err != nil {
		return err
	}

	if err := v.writeReg(DSS_CONFIG_ROI_MODE_CONTROL, 2); err != nil {
		return err
	}

	// Default to range with a 50 ms timing budget.
	if err := v.SetDistanceMode(v.distanceMode); err != nil {
		return err
	}

	if err := v.SetMeasurementTimingBudget(v.timingBudget); err != nil {
		return err
	}

	// Set part‐to‐part range offset from MM_CONFIG_OUTER_OFFSET_MM.
	outerOffset, err := v.readReg16Bit(MM_CONFIG_OUTER_OFFSET_MM)

	if err != nil {
		return err
	}

	if err := v.writeReg16Bit(ALGO_PART_TO_PART_RANGE_OFFSET_MM, outerOffset*4); err != nil {
		return err
	}

	return nil
}
