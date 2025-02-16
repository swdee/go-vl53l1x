package vl53l1x

import "time"

// SetTimeout set the timeout duration for reading sensor values
func (v *VL53L1X) SetTimeout(timeout time.Duration) {
	v.ioTimeout = timeout
}

// TimeoutOccurred reports whether a timeout has occurred
func (v *VL53L1X) TimeoutOccurred() bool {
	tmp := v.didTimeout
	v.didTimeout = false
	return tmp
}

// startTimeout starts the timeout counter
func (v *VL53L1X) startTimeout() {
	v.timeoutStart = time.Now()
}

// checkTimeoutExpired checks if timeout has expired
func (v *VL53L1X) checkTimeoutExpired() bool {
	return (v.ioTimeout > 0) && (time.Since(v.timeoutStart) > v.ioTimeout)
}
