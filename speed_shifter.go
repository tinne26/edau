package edau

// TODO: speed transitions. for smootheness, we need a progressive transition which
//       needs to be dealt with internally. And even then ebitengine will introduce
//       125ms of delay or so. We can change buffer size to reduce to 50, which gets
//       us to an average of 25ms. this is also broken by seeks.

import "io"
import "math"
import "sync"

// A valid SpeedShifter can only be created through the [NewDefaultSpeedShifter] or 
// [NewSpeedShifter] functions.
type SpeedShifter struct {
	mutex sync.Mutex
	source io.Reader
	speed float64
	windowSize int
	interpolator InterpolatorFunc

	fracPos float64
	leftoverSamples  int // from previous reads, not consumed yet
	lookaheadSamples int // lookahead bytes ready for interpolation
	
	leftWindow  circularWindow
	rightWindow circularWindow
	auxReadBuffer []byte
}

// Creates a default SpeedShifter.
func NewDefaultSpeedShifter(source io.Reader) *SpeedShifter {
	// Parameters are source, initial speed, window size and interpolator function.
	return NewSpeedShifter(source, 1.0, 6, InterpHermite6Pt3Ord)
}

func NewSpeedShifter(source io.Reader, speed float64, windowSize int, interpolator InterpolatorFunc) *SpeedShifter {
	if windowSize < 2 {
		panic("NewSpeedShifter windowSize must be at least 2")
	}
	if windowSize % 2 != 0 {
		panic("NewSpeedShifter windowSize must be multiple of 2 (may unrestrict in the future)")
		// Note: odd window sizes require interpolating around [center - 0.5, center + 0.5],
		//       so it's a bit trickier than even sizes, and the reason I didn't add it yet
	}

	const numChannels = 2
	bufferSize := windowSize*8
	buffer := make([]float64, bufferSize*numChannels)
	shifter := &SpeedShifter {
		source: source,
		speed: speed,
		windowSize: windowSize,
		interpolator: interpolator,
		leftWindow:  circularWindow{ winSize: windowSize, buffer: buffer[ : bufferSize] },
		rightWindow: circularWindow{ winSize: windowSize, buffer: buffer[bufferSize : ] },
		auxReadBuffer: nil,
	}
	
	shifter.internalReset()
	return shifter
}

func (self *SpeedShifter) Speed() float64 {
	self.mutex.Lock()
	speed := self.speed
	self.mutex.Unlock()
	return speed
}

func (self *SpeedShifter) SetSpeed(speed float64) {
	self.mutex.Lock()
	self.speed = speed
	self.mutex.Unlock()
}

// Implements io.Reader. This method tries to fill the buffer as much as possible.
func (self *SpeedShifter) Read(buffer []byte) (int, error) {
	// do not read incomplete samples (always read a number of bytes multiple of 4)
	buffer = buffer[0 : len(buffer) - (len(buffer) & 0b11)]

	// keep reading until an error happens or we fill the buffer
	bytesServed := 0
	for bytesServed < len(buffer) {
		// single read from underlying buffer
		n, err := self.singleRead(buffer[bytesServed : ])
		bytesServed += n
		if err != nil { return bytesServed, err }
	}

	return bytesServed, nil
}

// Like Read, but only reads once from the underlying source. If the given
// buffer can't be filled, this method doesn't retry, it simply returns what
// it got.
func (self *SpeedShifter) singleRead(buffer []byte) (int, error) {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	// base case
	if len(buffer) == 0 { return 0, nil }

	// general case
	requiredLookahead := self.windowSize/2
	pendingLookahead  := requiredLookahead - self.lookaheadSamples
	readCompensation  := (pendingLookahead << 2) - (self.leftoverSamples << 2)
	bytesToRead := int(math.Ceil(float64(len(buffer))*self.speed + float64(readCompensation)))
	bytesToRead += bytesToRead & 0b11 // align to sample size

	// acquire aux buffer for reading
	minReadBufferSize := (self.leftoverSamples << 2) + bytesToRead
	var readBuffer []byte
	if len(self.auxReadBuffer) < minReadBufferSize {
		if cap(self.auxReadBuffer) >= minReadBufferSize {
			readBuffer = self.auxReadBuffer[0 : minReadBufferSize]
		} else {
			readBuffer = make([]byte, minReadBufferSize)
		}
	} else {
		readBuffer = self.auxReadBuffer[0 : minReadBufferSize]
	}

	// copy leftoverSamples to the start of the buffer
	leftoverBytes := (self.leftoverSamples << 2)
	if self.leftoverSamples > 0 {
		copy(readBuffer[:], self.auxReadBuffer[len(self.auxReadBuffer) - leftoverBytes : ])
	}

	// read from underlying source
	var srcBytesRead int
	var err error
	if bytesToRead > 0 {
		srcBytesRead, err = self.source.Read(readBuffer[leftoverBytes : ])
	}
	srcBytesRead += leftoverBytes
	readBuffer = readBuffer[0 : srcBytesRead]

	// set read buffer as self.auxReadBuffer for next iterations
	self.auxReadBuffer = readBuffer

	// ensure lookahead is properly set
	for self.lookaheadSamples < self.windowSize/2 {
		if len(readBuffer) == 0 { return 0, err }
		left, right := GetSampleAsI16(readBuffer)
		self.leftWindow.Push(float64(left))
		self.rightWindow.Push(float64(right))
		self.lookaheadSamples += 1
		readBuffer = readBuffer[4 : ]
	}

	// process the bytes
	bytesServed := 0
	interpPosBase := float64(self.windowSize/2 - 1)
	for len(readBuffer) > 0 && bytesServed < len(buffer) {
		// add sample
		if self.fracPos < 1.0 {
			left  := self.interpolator(self.leftWindow.Get(),  interpPosBase + self.fracPos)
			right := self.interpolator(self.rightWindow.Get(), interpPosBase + self.fracPos)
			StoreF64SampleAsL16(buffer[bytesServed : ], left, right)
			bytesServed += 4
			self.fracPos += self.speed
		}

		// advance position
		for self.fracPos >= 1.0 && len(readBuffer) > 0 {
			left, right := GetSampleAsI16(readBuffer)
			self.leftWindow.Push(float64(left))
			self.rightWindow.Push(float64(right))
			readBuffer = readBuffer[4 : ]
			self.fracPos -= 1.0
		}
	}
	
	// update leftover samples
	if (srcBytesRead - bytesServed) & 0b11 != 0 { panic("unexpected situation") }
	self.leftoverSamples = (len(readBuffer) >> 2)
	if self.leftoverSamples < 0 { panic("unexpected situation") }

	// return
	return bytesServed, err
}

// You may use Seek to rewind and start playing after stoping, but not to loop
// or do seamless seeking with the resampled stream itself. Seamless seeking
// could only be done correctly if notifying the seek in advance of the
// interpolation window. That's not something anyone wants to deal with, so seeking
// will seek on the underlying buffer but reset the internal interpolation window
// of the speed shifter.
//
// This method panics if the underlying source doesn't implement [io.Seeker].
func (self *SpeedShifter) Seek(offset int64, whence int) (int64, error) {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	if whence == io.SeekCurrent {
		if offset == 0 { return 0, nil }
		panic("can't use relative seeks on a SpeedShifter (due to lookaheads)")
	}

	// seek underlying source
	seeker := self.source.(io.Seeker)
	position, err := seeker.Seek(offset, whence)

	// Resets interpolation window and related state.
	self.internalReset()

	// return seek results
	return position, err
}

// Resets interpolation window and related state.
func (self *SpeedShifter) internalReset() {
	self.leftoverSamples  = 0
	self.lookaheadSamples = 0
	self.leftWindow.Reset()
	self.rightWindow.Reset()
	for i := 0; i < self.windowSize/2 - 1; i++ {
		self.leftWindow.Push(0)
		self.rightWindow.Push(0)
	}
	
	buffer := []byte{0, 0, 0, 0}
	n, _ := self.source.Read(buffer)
	if n == 4 {
		left, right := GetSampleAsF64(buffer)
		self.leftWindow.Push(left)
		self.rightWindow.Push(right)
	} else {
		self.leftWindow.Push(0)
		self.rightWindow.Push(0)
	}
}

// --- helper circularWindow type for efficient interpolation ---

type circularWindow struct {
	winSize int
	buffer []float64
	startIndex int
	endIndex int
}

func (self *circularWindow) Reset() {
	self.startIndex = 0
	self.endIndex   = 0
}

func (self *circularWindow) Get() []float64 {
	return self.buffer[self.startIndex : self.endIndex]
}

func (self *circularWindow) Push(value float64) {
	if self.endIndex < len(self.buffer) {
		self.buffer[self.endIndex] = value
		if self.winSize == self.endIndex - self.startIndex { // window is full
			self.startIndex += 1
		}
		self.endIndex += 1
	} else {
		if self.winSize == self.endIndex - self.startIndex { // window is full
			copy(self.buffer, self.buffer[self.startIndex + 1 : self.endIndex])
			self.buffer[self.winSize - 1] = value
			self.startIndex = 0
			self.endIndex   = self.winSize
		} else {
			copy(self.buffer, self.buffer[self.startIndex : self.endIndex])
			newIndex := self.endIndex - self.startIndex
			self.buffer[newIndex] = value
			self.startIndex = 0
			self.endIndex   = newIndex + 1
		}
	}
}
