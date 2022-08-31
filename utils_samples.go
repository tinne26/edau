package edau

// Reads the first 4 bytes from the given slice and converts them from L16,
// 2 channel, little-endian format to 2 channel float64 values in the [-1, 1]
// range. Will panic if len(buffer) < 4.
func GetSampleAsF64(buffer []byte) (float64, float64) {
	left, right := GetSampleAsI16(buffer)
	return NormalizeF64(float64(left)), NormalizeF64(float64(right))
}

// Reads the first 4 bytes from the given slice (the first sample) and returns
// the left and right channel values. Results fall in the [-32768, 32767] range.
// Will panic if len(buffer) < 4.
func GetSampleAsI16(buffer []byte) (int16, int16) {
	left  := (int16(buffer[1]) << 8) | int16(buffer[0])
	right := (int16(buffer[3]) << 8) | int16(buffer[2])
	return left, right
}

// Stores the given unnormalized ([-32768, 32767]) values as a L16, 2 channel,
// little-endian sample right at the start of the given slice. Values out of range
// will be clipped. Will panic if len(buffer) < 4.
func StoreF64SampleAsL16(buffer []byte, left, right float64) {
	StoreL16Sample(buffer, clipFloatToI16(left), clipFloatToI16(right))
}

// Stores the given normalized ([-1, 1]) values as a L16, 2 channel, little-endian
// sample right at the start of the given slice. Values out of range will be clipped.
// Will panic if len(buffer) < 4.
func StoreNormF64SampleAsL16(buffer []byte, left, right float64) {
	StoreL16Sample(buffer, normFloatToI16(left), normFloatToI16(right))
}

// Stores the given values as a L16, 2 channel, little-endian sample right at the
// start of the given slice. Will panic if len(buffer) < 4.
func StoreL16Sample(buffer []byte, left int16, right int16) {
	buffer[0] = byte(left)       // left sample low byte
	buffer[1] = byte(left  >> 8) // left sample high byte
	buffer[2] = byte(right)      // right sample low byte
	buffer[3] = byte(right >> 8) // right sample high byte
}

func clipFloatToI16(value float64) int16 {
	if value >=  32767 { return  32767 }
	if value <= -32768 { return -32768 }
	return int16(value)
}

func normFloatToI16(value float64) int16 {
	if value >= 0 {
		if value >=  1.0 { return  32767 }
		return int16(value*32767.0)
	} else { // value < 0
		if value <= -1.0 { return -32768 }
		return int16(value*32768.0)
	}
}

// Normalize a float64 value from [-32768, 32767] to [-1, 1].
func NormalizeF64(value float64) float64 {
	// TODO: this is quite wasteful, isn't it?
	if value >= 0 {
		if value >=  32767 { return  1.0 }
		return value/32767.0
	} else { // value < 0
		if value <= -32768 { return -1.0 }
		return value/32768.0
	}
}