package vl53l1x

import (
	"fmt"
	"time"
)

// RangeStatus represents the sensor’s reported status.
type RangeStatus uint8

const (
	RangeValid                RangeStatus = 0
	SigmaFail                 RangeStatus = 1
	SignalFail                RangeStatus = 2
	RangeValidMinRangeClipped RangeStatus = 3
	OutOfBoundsFail           RangeStatus = 4
	HardwareFail              RangeStatus = 5
	RangeValidNoWrapCheckFail RangeStatus = 6
	WrapTargetFail            RangeStatus = 7
	XtalkSignalFail           RangeStatus = 9
	SynchronizationInt        RangeStatus = 10
	MinRangeFail              RangeStatus = 13
	NoneStatus                RangeStatus = 255
)

// RangingData holds a single range measurement and related rate information.
type RangingData struct {
	RangeMM                 uint16
	RangeStatus             RangeStatus
	PeakSignalCountRateMCPS float32
	AmbientCountRateMCPS    float32
}

// String implement Stringer interface for RangeStatus
func (s RangeStatus) String() string {
	switch s {
	case RangeValid:
		return "range valid"
	case SigmaFail:
		return "sigma fail"
	case SignalFail:
		return "signal fail"
	case RangeValidMinRangeClipped:
		return "range valid, min range clipped"
	case OutOfBoundsFail:
		return "out of bounds fail"
	case HardwareFail:
		return "hardware fail"
	case RangeValidNoWrapCheckFail:
		return "range valid, no wrap check fail"
	case WrapTargetFail:
		return "wrap target fail"
	case XtalkSignalFail:
		return "xtalk signal fail"
	case SynchronizationInt:
		return "synchronization int"
	case MinRangeFail:
		return "min range fail"
	case NoneStatus:
		return "no update"
	default:
		return "unknown status"
	}
}

// StartContinuous begins continuous ranging with the given period (in ms).
func (v *VL53L1X) StartContinuous(periodMs uint32) error {

	v.log.Print("Start continuous mode")

	// Write inter-measurement period (periodMs * osc_calibrate_val)
	val := periodMs * uint32(v.oscCalibrateVal)

	if err := v.writeReg32Bit(SYSTEM_INTERMEASUREMENT_PERIOD, val); err != nil {
		return err
	}

	if err := v.writeReg(SYSTEM_INTERRUPT_CLEAR, 0x01); err != nil {
		return err
	}

	// 0x40 is mode_start timed
	return v.writeReg(SYSTEM_MODE_START, 0x40)
}

// StopContinuous stops continuous ranging.
func (v *VL53L1X) StopContinuous() error {

	v.log.Print("Stop continuous mode")

	// 0x80 is mode_start abort
	if err := v.writeReg(SYSTEM_MODE_START, 0x80); err != nil {
		return err
	}

	// In low-power auto mode, restore VHV configuration.
	v.calibrated = false

	if v.savedVHVInit != 0 {
		if err := v.writeReg(VHV_CONFIG_INIT, v.savedVHVInit); err != nil {
			return err
		}
	}

	if v.savedVHVTimeout != 0 {
		if err := v.writeReg(VHV_CONFIG_TIMEOUT_MACROP_LOOP_BOUND, v.savedVHVTimeout); err != nil {
			return err
		}
	}

	// remove phasecal override
	return v.writeReg(PHASECAL_CONFIG_OVERRIDE, 0x00)
}

// Read returns a range data read from sensor. If blocking is true, this function
// will wait for a new measurement to be captured.  If blocking is false then it
// reads existing measurement from register.
func (v *VL53L1X) Read(blocking bool) (RangingData, error) {

	if blocking {

		v.startTimeout()

		for {
			ready, err := v.dataReady()

			if err != nil {
				return RangingData{}, err
			}

			if ready {
				break
			}

			if v.checkTimeoutExpired() {
				v.didTimeout = true
				return RangingData{}, fmt.Errorf("timeout waiting for data")
			}

			time.Sleep(1 * time.Millisecond)
		}
	}

	if err := v.readResults(); err != nil {
		return RangingData{}, err
	}

	if !v.calibrated {
		if err := v.setupManualCalibration(); err != nil {
			return RangingData{}, err
		}

		v.calibrated = true
	}

	if err := v.updateDSS(); err != nil {
		return RangingData{}, err
	}

	rData := v.getRangingData()

	if err := v.writeReg(SYSTEM_INTERRUPT_CLEAR, 0x01); err != nil {
		return RangingData{}, err
	}

	return rData, nil
}

// ReadSingle performs a single-shot ranging measurement
func (v *VL53L1X) ReadSingle() (RangingData, error) {

	if err := v.writeReg(SYSTEM_INTERRUPT_CLEAR, 0x01); err != nil {
		return RangingData{}, err
	}

	// 0x10 is mode_start single shot
	if err := v.writeReg(SYSTEM_MODE_START, 0x10); err != nil {
		return RangingData{}, err
	}

	rData, err := v.Read(true)
	return rData, err

}

// ReadRangeContinuousMillimeters returns a range reading in millimeters
// when continuous mode is active
func (v *VL53L1X) ReadRangeContinuousMillimeters() (uint16, error) {
	rData, err := v.Read(true)
	return rData.RangeMM, err
}

// ReadRangeSingleMillimeters performs a single-shot range measurement and returns the reading in
// millimeters
func (v *VL53L1X) ReadRangeSingleMillimeters() (uint16, error) {
	rData, err := v.ReadSingle()
	return rData.RangeMM, err
}

// dataReady checks if the sensor has a new reading available. It assumes interrupt
// is active Low (GPIO_HV_MUX__CTRL bit 4 is 1)
func (v *VL53L1X) dataReady() (bool, error) {

	status, err := v.readReg(GPIO_TIO_HV_STATUS)

	if err != nil {
		return false, err
	}

	// Active low: data ready when bit 0 == 0.
	return (status & 0x01) == 0, nil
}

// readResults reads sensor measurement results into buffer
func (v *VL53L1X) readResults() error {

	// Begin reading at RESULT_RANGE_STATUS.
	addr := []byte{byte(RESULT_RANGE_STATUS >> 8), byte(RESULT_RANGE_STATUS)}

	if _, err := v.bus.WriteBytes(addr); err != nil {
		return err
	}

	buf := make([]byte, 17)

	n, err := v.bus.ReadBytes(buf)

	if err != nil {
		return err
	}

	if n < 17 {
		return fmt.Errorf("readResults: insufficient data read")
	}

	v.results.rangeStatus = buf[0]

	// report_status (buf[1]) -- not used

	v.results.streamCount = buf[2]
	v.results.dssActualEffectiveSpadsSD0 = uint16(buf[3])<<8 | uint16(buf[4])

	// peak_signal_count_rate_mcps_sd0 (buf[5], buf[6]) -- not used

	v.results.ambientCountRateMCPS_SD0 = uint16(buf[7])<<8 | uint16(buf[8])

	// sigma_sd0 (buf[9], buf[10]) and phase_sd0 (buf[11], buf[12]) -- not used

	v.results.finalCrosstalkCorrectedRangeMM_SD0 = uint16(buf[13])<<8 | uint16(buf[14])
	v.results.peakSignalCountRateCrosstalkCorrectedMCPS_SD0 = uint16(buf[15])<<8 | uint16(buf[16])

	return nil
}

// setupManualCalibration sets up ranges after the first one in low power auto
// mode by turning off FW calibration steps and programming static values.  based
// on VL53L1_low_power_auto_setup_manual_calibration()
func (v *VL53L1X) setupManualCalibration() error {

	// save original vhv configs
	initVal, err := v.readReg(VHV_CONFIG_INIT)

	if err != nil {
		return err
	}

	v.savedVHVInit = initVal
	timeoutVal, err := v.readReg(VHV_CONFIG_TIMEOUT_MACROP_LOOP_BOUND)

	if err != nil {
		return err
	}

	v.savedVHVTimeout = timeoutVal

	// disable VHV init
	if err := v.writeReg(VHV_CONFIG_INIT, v.savedVHVInit&0x7F); err != nil {
		return err
	}

	// set loop bound to tuning param
	newVal := (v.savedVHVTimeout & 0x03) + (3 << 2)

	if err := v.writeReg(VHV_CONFIG_TIMEOUT_MACROP_LOOP_BOUND, newVal); err != nil {
		return err
	}

	// override phasecal
	if err := v.writeReg(PHASECAL_CONFIG_OVERRIDE, 0x01); err != nil {
		return err
	}

	phStart, err := v.readReg(PHASECAL_RESULT_VCSEL_START)

	if err != nil {
		return err
	}

	return v.writeReg(CAL_CONFIG_VCSEL_START, phStart)
}

// updateDSS performs dynamic SPAD selection calculation/update based on
// VL53L1_low_power_auto_update_DSS()
func (v *VL53L1X) updateDSS() error {

	spadCount := v.results.dssActualEffectiveSpadsSD0

	if spadCount != 0 {
		// calc total rate per spad
		totalRatePerSpad := uint32(v.results.peakSignalCountRateCrosstalkCorrectedMCPS_SD0) +
			uint32(v.results.ambientCountRateMCPS_SD0)

		// clip to 16 bits
		if totalRatePerSpad > 0xFFFF {
			totalRatePerSpad = 0xFFFF
		}

		// shift up to take advantage of 32 bits
		totalRatePerSpad <<= 16
		totalRatePerSpad /= uint32(spadCount)

		if totalRatePerSpad != 0 {
			// get the target rate and shift up by 16
			requiredSpads := (uint32(TargetRate) << 16) / totalRatePerSpad

			// clip to 16 bit
			if requiredSpads > 0xFFFF {
				requiredSpads = 0xFFFF
			}

			// override DSS config
			return v.writeReg16Bit(DSS_CONFIG_MANUAL_EFFECTIVE_SPADS_SELECT, uint16(requiredSpads))
		}
	}

	// If we reached this point, it means something above would have resulted in a
	// divide by zero. We want to gracefully set a spad target, not just exit
	// with an error so fall back to a mid‐point target.
	return v.writeReg16Bit(DSS_CONFIG_MANUAL_EFFECTIVE_SPADS_SELECT, 0x8000)
}

// getRangingData gets range, status, rates from results buffer based on
// VL53L1_GetRangingMeasurementData()
func (v *VL53L1X) getRangingData() RangingData {

	rData := RangingData{}

	rangeVal := v.results.finalCrosstalkCorrectedRangeMM_SD0

	// apply a gain correction: (r * 2011 + 0x0400) / 0x0800
	rData.RangeMM = uint16((uint32(rangeVal)*2011 + 0x0400) / 0x0800)

	switch v.results.rangeStatus {

	case 17, 2, 1, 3:
		rData.RangeStatus = HardwareFail
	case 13:
		rData.RangeStatus = MinRangeFail
	case 18:
		rData.RangeStatus = SynchronizationInt
	case 5:
		rData.RangeStatus = OutOfBoundsFail
	case 4:
		rData.RangeStatus = SignalFail
	case 6:
		rData.RangeStatus = SigmaFail
	case 7:
		rData.RangeStatus = WrapTargetFail
	case 12:
		rData.RangeStatus = XtalkSignalFail
	case 8:
		rData.RangeStatus = RangeValidMinRangeClipped
	case 9:
		if v.results.streamCount == 0 {
			rData.RangeStatus = RangeValidNoWrapCheckFail
		} else {
			rData.RangeStatus = RangeValid
		}
	default:
		rData.RangeStatus = NoneStatus
	}

	// from SetSimpleData()
	rData.PeakSignalCountRateMCPS = v.countRateFixedToFloat(v.results.peakSignalCountRateCrosstalkCorrectedMCPS_SD0)
	rData.AmbientCountRateMCPS = v.countRateFixedToFloat(v.results.ambientCountRateMCPS_SD0)

	return rData
}

// countRateFixedToFloat converts count rate from fixed point 9.7 format to float
func (v *VL53L1X) countRateFixedToFloat(countRateFixed uint16) float32 {
	return float32(countRateFixed) / float32(1<<7)
}
