package vl53l1x

import "fmt"

// SetROISize sets the region‐of‐interest size given the width and height of the
// 16x16 SPAD array
func (v *VL53L1X) SetROISize(width, height uint8) error {

	// check SPAD array bounds
	if width > 16 {
		width = 16
	}

	if height > 16 {
		height = 16
	}

	// enforce a minimum ROI size of 4x4
	if width < 4 || height < 4 {
		return fmt.Errorf("ROI size must be at least 4x4")
	}

	// force ROI to be centered if width or height > 10, matching what the ULD API
	// does.
	if width > 10 || height > 10 {
		if err := v.writeReg(ROI_CONFIG_USER_ROI_CENTRE_SPAD, 199); err != nil {
			return err
		}
	}

	val := ((height - 1) << 4) | (width - 1)

	return v.writeReg(ROI_CONFIG_USER_ROI_REQUESTED_GLOBAL_XY_SIZE, val)
}

// GetROISize returns the current ROI width and height
func (v *VL53L1X) GetROISize() (width, height uint8, err error) {

	regVal, err := v.readReg(ROI_CONFIG_USER_ROI_REQUESTED_GLOBAL_XY_SIZE)

	if err != nil {
		return 0, 0, err
	}

	width = (regVal & 0x0F) + 1
	height = (regVal >> 4) + 1

	return width, height, nil
}

// SetROICenter sets the center SPAD number of the region of interest (ROI)
// based on VL53L1X_SetROICenter() from STSW-IMG009 Ultra Lite Driver
//
// ST user manual UM2555 explains ROI selection in detail, so we recommend
// reading that document carefully. Here is a table of SPAD locations from
// UM2555 (199 is the default/center):
//
// 128,136,144,152,160,168,176,184,  192,200,208,216,224,232,240,248
// 129,137,145,153,161,169,177,185,  193,201,209,217,225,233,241,249
// 130,138,146,154,162,170,178,186,  194,202,210,218,226,234,242,250
// 131,139,147,155,163,171,179,187,  195,203,211,219,227,235,243,251
// 132,140,148,156,164,172,180,188,  196,204,212,220,228,236,244,252
// 133,141,149,157,165,173,181,189,  197,205,213,221,229,237,245,253
// 134,142,150,158,166,174,182,190,  198,206,214,222,230,238,246,254
// 135,143,151,159,167,175,183,191,  199,207,215,223,231,239,247,255
//
// 127,119,111,103, 95, 87, 79, 71,   63, 55, 47, 39, 31, 23, 15,  7
// 126,118,110,102, 94, 86, 78, 70,   62, 54, 46, 38, 30, 22, 14,  6
// 125,117,109,101, 93, 85, 77, 69,   61, 53, 45, 37, 29, 21, 13,  5
// 124,116,108,100, 92, 84, 76, 68,   60, 52, 44, 36, 28, 20, 12,  4
// 123,115,107, 99, 91, 83, 75, 67,   59, 51, 43, 35, 27, 19, 11,  3
// 122,114,106, 98, 90, 82, 74, 66,   58, 50, 42, 34, 26, 18, 10,  2
// 121,113,105, 97, 89, 81, 73, 65,   57, 49, 41, 33, 25, 17,  9,  1
// 120,112,104, 96, 88, 80, 72, 64,   56, 48, 40, 32, 24, 16,  8,  0 <- Pin 1
//
// This table is oriented as if looking into the front of the sensor (or top of
// the chip). SPAD 0 is closest to pin 1 of the VL53L1X, which is the corner
// closest to the VDD pin on the Pololu VL53L1X carrier board:
//
//	+--------------+
//	|             O| GPIO1
//	|              |
//	|             O|
//	| 128    248   |
//	|+----------+ O|
//	||+--+  +--+|  |
//	|||  |  |  || O|
//	||+--+  +--+|  |
//	|+----------+ O|
//	| 120      0   |
//	|             O|
//	|              |
//	|             O| VDD
//	+--------------+
//
// However, note that the lens inside the VL53L1X inverts the image it sees
// (like the way a camera works). So for example, to shift the sensor's FOV to
// sense objects toward the upper left, you should pick a center SPAD in the
// lower right.
func (v *VL53L1X) SetROICenter(spadNumber uint8) error {
	return v.writeReg(ROI_CONFIG_USER_ROI_CENTRE_SPAD, spadNumber)
}

// GetROICenter returns the current center SPAD
func (v *VL53L1X) GetROICenter() (uint8, error) {
	return v.readReg(ROI_CONFIG_USER_ROI_CENTRE_SPAD)
}
