// go-vl53l1x is an I2C driver for the ST VL53L1X time‐of‐flight sensor.
package vl53l1x

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/swdee/go-i2c"
)

const (
	// Address is the default address of the sensor on I2C bus
	Address uint8 = 0x29
	// TimingGuard is used in measurement timing budget calculations and is
	// given in microseconds
	TimingGuard uint32 = 4528
	// TargetRate is used in DSS calculations
	TargetRate uint16 = 0x0A00
)

// resultBuffer holds raw values read from the sensor
type resultBuffer struct {
	rangeStatus                                   uint8
	streamCount                                   uint8
	dssActualEffectiveSpadsSD0                    uint16
	ambientCountRateMCPS_SD0                      uint16
	finalCrosstalkCorrectedRangeMM_SD0            uint16
	peakSignalCountRateCrosstalkCorrectedMCPS_SD0 uint16
}

// VL53L1X represents a single VL53L1X sensor instance.
type VL53L1X struct {
	// bus is the I2C interface
	bus *i2c.Options

	ioTimeout    time.Duration
	didTimeout   bool
	timeoutStart time.Time

	fastOscFrequency uint16
	oscCalibrateVal  uint16

	calibrated      bool
	savedVHVInit    uint8
	savedVHVTimeout uint8

	distanceMode DistanceMode
	// timing budget in milliseconds
	timingBudget uint32

	lastStatus uint8

	results resultBuffer

	// log logger for debugging
	log *log.Logger
}

// New returns a new VL53L1X sensor instance configured with the specified
// DistanceMode and Timing Budget interval in milliseconds
func New(i2c *i2c.Options, mode DistanceMode, budget uint32) (*VL53L1X, error) {

	v, err := new(i2c, mode, budget)

	if err != nil {
		return nil, err
	}

	// create null logger
	v.log = log.New(io.Discard, "", log.LstdFlags)

	// finish device setup
	err = v.setup()

	return v, err
}

// New creates sensor instance with logger to be used for debugging configured
// with the specified DistanceMode and Timing Budget interval in milliseconds
func NewWithLog(i2c *i2c.Options, mode DistanceMode, budget uint32,
	log *log.Logger) (*VL53L1X, error) {

	v, err := new(i2c, mode, budget)

	if err != nil {
		return nil, err
	}

	// set logger
	v.log = log

	// finish device setup
	err = v.setup()

	return v, err
}

// new returns a new VL53L1X sensor instance
func new(i2c *i2c.Options, mode DistanceMode, budget uint32) (*VL53L1X, error) {

	addr := i2c.GetAddr()

	if addr == 0 {
		return nil, fmt.Errorf("I2C device is not initiated")
	}

	v := &VL53L1X{
		bus:          i2c,
		ioTimeout:    0, // no timeout by default
		calibrated:   false,
		distanceMode: mode,
		timingBudget: budget,
	}

	return v, nil
}

// setup completes New instance creation and is a common function for New() and
// NewWithLog()
func (v *VL53L1X) setup() error {

	v.log.Printf("Starting Setup()")

	// initialize device
	err := v.Init()

	if err != nil {
		return fmt.Errorf("Failed to Init device: %w", err)
	}

	v.log.Printf("Device Init()'d")

	return nil
}

// SetAddress change default address of sensor and reopen I2C-connection.
func (v *VL53L1X) SetAddress(newAddr uint8) error {

	if err := v.writeReg(I2C_SLAVE_DEVICE_ADDRESS, newAddr&0x7F); err != nil {
		return err
	}

	// open new connection
	i2c, err := i2c.New(newAddr, v.bus.GetDev())

	if err != nil {
		return err
	}

	// close existing connection
	v.bus.Close()

	// replace with new connection
	v.bus = i2c
	return nil
}
