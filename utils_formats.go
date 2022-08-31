package edau

import "os"
import "io"
import "fmt"
import "strings"
import "errors"

import "github.com/hajimehoshi/ebiten/v2/audio"
import "github.com/hajimehoshi/ebiten/v2/audio/wav"
import "github.com/hajimehoshi/ebiten/v2/audio/mp3"
import "github.com/hajimehoshi/ebiten/v2/audio/vorbis"

// A common interface that all Ebitengine audio streams conform to.
type StdAudioStream interface {
	io.ReadSeeker
	Length() int64
}

// Returned by functions that require Ebitengine's audio.NewContext to have
// already been created, typically so the sample rate can be directly obtained
// from it with audio.CurrentContext().SampleRate().
var ErrAudioContextUninitialized = errors.New("Ebitengine's audio context not initialized")

// Loads an .ogg, .mp3 or .wav file as a [StdAudioStream]. Additionally,
// the returned interface also implements [io.Closer], which can be used
// to close the file associated to the stream, e.g.:
//    err := audioStream.(io.Closer).Close()
// The sample rate used is taken from Ebitengine's audio.CurrentContext().
// If no audio context has been initialized, [ErrAudioContextUninitialized]
// will be returned.
func LoadAudioFileAsStream(filename string) (StdAudioStream, error) {
	ctx := audio.CurrentContext()
	if ctx == nil { return nil, ErrAudioContextUninitialized }

	file, err := os.Open(filename)
	if err != nil { return nil, err }
	
	var stream StdAudioStream
	if strings.HasSuffix(filename, ".wav") {
		stream, err = wav.DecodeWithSampleRate(ctx.SampleRate(), file)
	} else if strings.HasSuffix(filename, ".ogg") {
		stream, err = vorbis.DecodeWithSampleRate(ctx.SampleRate(), file)
	} else if strings.HasSuffix(filename, ".mp3") {
		stream, err = mp3.DecodeWithSampleRate(ctx.SampleRate(), file)
	} else {
		return nil, fmt.Errorf("unexpected audio format for '%s'", filename)
	}

	return &streamWithClose{ stream, file }, err
}

type streamWithClose struct {
	stream StdAudioStream
	file *os.File
}

func (self *streamWithClose) Read(buffer []byte) (int, error) {
	return self.stream.Read(buffer)
}
func (self *streamWithClose) Seek(offset int64, whence int) (int64, error) {
	return self.stream.Seek(offset, whence)
}
func (self *streamWithClose) Length() int64 {
	return self.stream.Length()
}
func (self *streamWithClose) Close() error {
	return self.file.Close()
}

