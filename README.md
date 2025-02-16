# go-vl53l1x

go-vl53l1x is an I2C driver for the ST VL53L1X Time-of-Flight sensor used for  
distance ranging.


## Usage

To use in your Go project, get the library
```
go get github.com/swdee/go-vl53l1x
```



## Locate Sensor Device

Find which I2C bus the VL53L1X sensor is located by running `i2cdetect`.  In the
below case the sensor `0x29` is running on bus `0` which maps to device `/dev/i2c-0`.
```
$ i2cdetect -y 0

     0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
00:                         -- -- -- -- -- -- -- -- 
10: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- 
20: -- -- -- -- -- -- -- -- -- 29 -- -- -- -- -- -- 
30: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- 
40: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- 
50: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- 
60: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- 
70: -- -- -- -- -- -- -- --              
```


## Example

To read distance in millimeters from the sensor.

```
// connect to sensor on I2C bus
i2c, _ := i2c.New(vl53l1x.Address, "/dev/i2c-0")

// initalize sensor in Short Distance Mode with 50ms Timing Budget	
sensor,  := vl53l1x.New(i2c, vl53l1x.Short, 50)

// read distance as single shot
distance, _ := sensor.ReadRangeSingleMillimeters()

fmt.Printf("Distance (mm): %d\n", distance)
```

Note: Error handling has been skipped for brevity.

For a more complex example using Continuous Polling and Region's of Interest
see the [example here](example/main.go).


## Distance Mode

The VL53L1X has three distance modes (DM): short, medium, and long.

Long distance mode allows the longest possible ranging distance of 4 m to be 
reached. However, this maximum ranging distance is impacted by ambient light.

Short distance mode is more immune to ambient light, but its maximum ranging 
distance is typically limited to 1.3m.

| Distance Mode | Max. distance in the dark (cm) | Max distance under strong ambient light (cm) |
|---------------|--------------------------------|----------------------------------------------|
| Short         | 136                            | 135                                          |
| Medium        | 290                            | 76                                           |
| Long          | 360                            | 73                                           |


Set your preferred distance when initializing the sensor.
```
sensor,  := vl53l1x.New(i2c, vl53l1x.Long, 50)
```


## Timing Budget

The VL53L1X timing budget can be set from 20 ms up to 1000 ms.

* 20 ms is the minimum timing budget and can be used only in short distance mode.
* 33 ms is the minimum timing budget which can work for all distance modes.
* 140 ms is the timing budget which allows the maximum distance of 4 m (in the 
  dark on a white chart) to be reached with long distance mode.

Increasing the timing budget increases the maximum distance the device can range 
and improves the repeatability error. However, average power consumption 
augments accordingly.


Set your preferred timing budget when initializing the sensor.
```
sensor,  := vl53l1x.New(i2c, vl53l1x.Long, 33)
```


## Continous Polling Mode

When using continous polling mode, it is recommened on the ST community forum to set
the inter-measurement Period to be 5ms longer than the Timing Budget.

So if your Timing Budget is 33ms than setting a Period of 38ms is preferred.
```
sensor.StartContinuous(38)
```


## Region of Interest (ROI) zone

The Field-of-View of the sensor can be modified by setting up a ROI that
effects the area of the SPAD sensor to limit monitoring to a set zone.  See
the official datasheet [UM2555](https://www.st.com/resource/en/user_manual/um2555-vl53l1x-ultra-lite-driver-multiple-zone-implementation-stmicroelectronics.pdf)
on more details about this.


### ROI Size

The Region must be atleast 4x4 in size and center must be specified so that the
entire region fits within the SPAD array.

To specify a ROI size use the following where the width is `16` and height is `6`.
```
sensor.SetROISize(16, 6)
```



### ROI Center

The SPAD locations of the 16x16 array are specified as the following where `199`
is the default center.

```
 128,136,144,152,160,168,176,184,  192,200,208,216,224,232,240,248
 129,137,145,153,161,169,177,185,  193,201,209,217,225,233,241,249
 130,138,146,154,162,170,178,186,  194,202,210,218,226,234,242,250
 131,139,147,155,163,171,179,187,  195,203,211,219,227,235,243,251
 132,140,148,156,164,172,180,188,  196,204,212,220,228,236,244,252
 133,141,149,157,165,173,181,189,  197,205,213,221,229,237,245,253
 134,142,150,158,166,174,182,190,  198,206,214,222,230,238,246,254
 135,143,151,159,167,175,183,191,  199,207,215,223,231,239,247,255

 127,119,111,103, 95, 87, 79, 71,   63, 55, 47, 39, 31, 23, 15,  7
 126,118,110,102, 94, 86, 78, 70,   62, 54, 46, 38, 30, 22, 14,  6
 125,117,109,101, 93, 85, 77, 69,   61, 53, 45, 37, 29, 21, 13,  5
 124,116,108,100, 92, 84, 76, 68,   60, 52, 44, 36, 28, 20, 12,  4
 123,115,107, 99, 91, 83, 75, 67,   59, 51, 43, 35, 27, 19, 11,  3
 122,114,106, 98, 90, 82, 74, 66,   58, 50, 42, 34, 26, 18, 10,  2
 121,113,105, 97, 89, 81, 73, 65,   57, 49, 41, 33, 25, 17,  9,  1
 120,112,104, 96, 88, 80, 72, 64,   56, 48, 40, 32, 24, 16,  8,  0 <- Pin 1
```

This table is oriented as if looking into the front of the sensor (or top of
the chip). SPAD 0 is closest to pin 1 of the VL53L1X.

To specify a center for your ROI, select the integer from the table above
eg: `59` and use.

```
sensor.SetROICenter(59)
```


## Background

This code is a port of the [C++ library](https://github.com/pololu/vl53l1x-arduino)


## References

* [VL53L1X Datasheet](https://www.st.com/resource/en/datasheet/vl53l1x.pdf)    
* [VL53L1X API user manual UM2356](https://www.st.com/resource/en/user_manual/um2356-vl53l1x-api-user-manual-stmicroelectronics.pdf)  


