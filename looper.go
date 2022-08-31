package edau

import "io"
import "sync"
import "bytes"

// A tight audio looper. Unlike Ebitengine's [infinite looper], this looper doesn't require padding
// after the end point because it doesn't perform any blending during the transition. Additionally,
// the start and end points can be changed at any time with [Looper.AdjustLoop].
//
// [infinite looper]: https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/audio#InfiniteLoop
type Looper struct {
	stream io.ReadSeeker
	mutex sync.Mutex
	position int64
	loopStart int64
	activeLoopEnd int64 // relevant when modifying loop points. if we are going towards an
	                    // end loop point but we change it for another that comes earlier
							  // but we have already past it, we still have to continue towards
							  // the previous loop end point
	loopEnd int64
}

// Creates a new tight [Looper].
//
// The stream must be a L16 little-endian stream with two channels (Ebitengine's
// default audio format). loopStart and loopEnd must be multiples of 4, and loopStart
// must be strictly smaller than loopEnd. This method will panic if any of those are
// not respected.
//
// The loopEnd point is not itself included in the loop. For example, to play a full
// audio stream in a loop, you would use NewLooper(stream, 0, stream.Length()).
//
// If you need help to determine the loop start and end points, see [apps/loop_finder].
//
// [apps/loop_finder]: https://github.com/tinne26/edau/tree/main/apps
func NewLooper(stream io.ReadSeeker, loopStart int64, loopEnd int64) *Looper {
	assertLoopValuesValidity(loopStart, loopEnd)
	return &Looper {
		stream: stream,
		loopStart: loopStart,
		loopEnd: loopEnd,
		activeLoopEnd: loopEnd,
	}
}

// Implements [io.Reader].
func (self *Looper) Read(buffer []byte) (int, error) {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	var bytesRead int
	for len(buffer) > 0 {
		untilNextLoop := self.activeLoopEnd - self.position
		if untilNextLoop < 0 { untilNextLoop = 0 }
		
		// simple case: not reaching next loop point yet
		if int64(len(buffer)) <= untilNextLoop {
			n, err := self.readAll(buffer)
			bytesRead += n
			return bytesRead, err
		}
	
		// complex case: one or more loop points reached
		if untilNextLoop > 0 {
			n, err := self.readAll(buffer[0 : untilNextLoop])
			bytesRead += n
			if err != nil { return bytesRead, err }
		}
	
		var err error
		self.activeLoopEnd = self.loopEnd
		self.position, err = self.stream.Seek(self.loopStart, io.SeekStart)
		if err != nil { return bytesRead, err }
		buffer = buffer[untilNextLoop : ]
	}

	return bytesRead, nil
}

func (self *Looper) readAll(buffer []byte) (int, error) {
	bytesRead := 0
	defer func(){ self.position += int64(bytesRead) }()
	
	for {
		// read and return if we are done or got an error
		n, err := self.stream.Read(buffer)
		bytesRead += n
		if n == len(buffer) || err != nil {
			return bytesRead, err
		}

		// we didn't read enough, try again
		buffer = buffer[n : ]
	}
}

// Seek seeks directly on the underlying stream. It is the caller's 
// responsibility to make sure the seek falls inside the current loop
// (if that's desired).
func (self *Looper) Seek(offset int64, whence int) (int64, error) {
	self.mutex.Lock()
	n, err := self.stream.Seek(offset, whence)
	self.position = n
	if self.position <= self.loopEnd {
		self.activeLoopEnd = self.loopEnd
	}
	self.mutex.Unlock()
	return n, err
}

// Returns the current playback position. The value will always be multiple
// of 4, as in Ebitengine each sample is composed of 4 bytes.
func (self *Looper) GetPosition() int64 {
	self.mutex.Lock()
	position := self.position
	self.mutex.Unlock()
	return position
}

// Returns the currently configured loop starting point (in bytes, not samples).
func (self *Looper) GetLoopStart() int64 {
	self.mutex.Lock()
	loopStart := self.loopStart
	self.mutex.Unlock()
	return loopStart
}

// Returns the currently configured loop ending point (in bytes, not samples).
func (self *Looper) GetLoopEnd() int64 {
	self.mutex.Lock()
	loopEnd := self.loopEnd
	defer self.mutex.Unlock()
	return loopEnd
}

// Like [Looper.GetLoopStart] and [Looper.GetLoopEnd], but both at once.
func (self *Looper) GetLoopPoints() (int64, int64) {
	self.mutex.Lock()
	loopStart := self.loopStart
	loopEnd   := self.loopEnd
	self.mutex.Unlock()
	return loopStart, loopEnd
}

// Sets new values for the loop starting and ending points. The values are
// []byte indices. Therefore, since Ebitengine audio samples require 4 bytes
// each, the passed start and end points must also be multiples of 4.
//
// If the new loop end is set before the current playback position, the loop
// will continue playing until the previously configured end point before
// the new loop comes into effect.
func (self *Looper) AdjustLoop(loopStart, loopEnd int64) {
	assertLoopValuesValidity(loopStart, loopEnd)
	self.mutex.Lock()
	self.loopStart = loopStart
	self.loopEnd = loopEnd
	if loopEnd >= self.position {
		self.activeLoopEnd = loopEnd
	}
	self.mutex.Unlock()
}

// Returns the underlying stream's length. The underlying stream must
// have a Length() int64 method or be a [bytes.Reader]. This method
// will panic otherwise.
func (self *Looper) Length() int64 {
	self.mutex.Lock()
	var length int64
	switch streamWithLen := self.stream.(type) {
	case *bytes.Reader:
		length = int64(streamWithLen.Len())
	case StdAudioStream:
		length = streamWithLen.Length()
	default:
		panic("Looper underlying stream doesn't implement Length() int64 and is not a *bytes.Reader either")
	}
	self.mutex.Unlock()
	return length
}

func assertLoopValuesValidity(loopStart, loopEnd int64) {
	if loopStart & 0b11 != 0 { panic("loopStart must be multiple of 4") }
	if loopEnd   & 0b11 != 0 { panic("loopEnd must be multiple of 4") }
	if loopStart >= loopEnd { panic("loopStart must be strictly smaller than loopEnd") }
	if loopStart < 0 { panic("loopStart must be >= 0") }
	// Note: technically loopStart can be loopEnd - 4 or similar extremely short distances.
	//       This is allowed but it's not really correct. Nothing will sound and the looper
	//       is likely to start lagging unless absurd sample rates are used. Other small
	//       loop lengths are equally likely to cause trouble, but that's on the user.
}