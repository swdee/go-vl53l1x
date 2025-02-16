package main

import (
	"flag"
	"fmt"
	"github.com/swdee/go-i2c"
	"github.com/swdee/go-vl53l1x"
	"log"
	"time"
)

func main() {

	i2cbus := flag.String("b", "/dev/i2c-0", "Path to I2C bus to use")
	flag.Parse()

	// Open I2C bus (adjust bus number and default address as needed)
	i2c, err := i2c.New(vl53l1x.Address, *i2cbus)

	if err != nil {
		log.Fatal(err)
	}

	defer i2c.Close()

	// create new sensor instance running in Short mode with timing budget 50ms
	sensor, err := vl53l1x.New(i2c, vl53l1x.Short, 50)

	if err != nil {
		log.Fatal(err)
	}

	// define a region of interest.  This is not necessary so can be commented
	// out if not required.
	setROI(sensor)

	// Start continuous ranging, it is recommend by ST for the Period to be 5ms
	// longer than the Timing Budget (50 + 5ms = 55ms)
	if err := sensor.StartContinuous(55); err != nil {
		log.Fatalf("Start continuous failed: %v", err)
	}

	// Read a measurement
	for i := 0; i < 10; i++ {

		data, err := sensor.Read(true)

		if err != nil {
			log.Printf("Read error: %v", err)
		} else {
			fmt.Printf("Distance: %d mm (status: %s)\n", data.RangeMM,
				data.RangeStatus.String())
		}

		time.Sleep(200 * time.Millisecond)
	}

	// Stop continuous ranging
	if err := sensor.StopContinuous(); err != nil {
		log.Fatalf("Stop continuous failed: %v", err)
	}

	// close I2C connection
	i2c.Close()
}

// setROI sets the region of interest
func setROI(sensor *vl53l1x.VL53L1X) {

	// set sensor region of interest
	err := sensor.SetROISize(12, 12)

	if err != nil {
		log.Fatalf("Setting ROI Size failed: %v\n", err)
	}

	err = sensor.SetROICenter(199)

	if err != nil {
		log.Fatalf("Setting ROI Center failed: %v\n", err)
	}

	// load current region of interest setting
	width, height, err := sensor.GetROISize()

	if err != nil {
		log.Fatalf("Get ROI size: %v", err)
	}

	centerPad, err := sensor.GetROICenter()

	if err != nil {
		log.Fatalf("Get ROI center: %v", err)
	}

	log.Printf("Get ROI size: %dx%d\n", width, height)
	log.Printf("Get ROI center: %d\n", centerPad)
}
