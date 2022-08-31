package main

import "io"
import "os"
import "fmt"
import "log"
import "errors"
import "strconv"
import "runtime"
import "path/filepath"

import "image"
import "image/color"

import "github.com/hajimehoshi/ebiten/v2"
import "github.com/hajimehoshi/ebiten/v2/audio"
import "github.com/hajimehoshi/ebiten/v2/ebitenutil"
import "github.com/hajimehoshi/ebiten/v2/inpututil"

import "github.com/tinne26/edau"

// note: directX still has some issues that prevent this
//       example from performing reasonably
func init() {
	if runtime.GOOS == "windows" {
		os.Setenv("EBITEN_GRAPHICS_LIBRARY", "opengl")
	}
}

// This program is an interactive tool to find loop points to be used
// with the edau.Looper type. It expects the path to the audio file
// as an argument, which must be .ogg, .wav or .mp3 and have a sample rate
// of 44.1kHz (this can easily be changed in the code below, but I didn't
// set it as a program parameter).
const SampleRate = 44100 // only 44100 or 48000 expected
const PlaybackPreMillis = 2300 // must end with two zeros
const SampleSize = 4 // this must not be changed, it's for clarity in code
const BufferViewLen = 63 // must be odd

// each time the loop points are moved, we re-read the relevant
// portion of the stream to update the soundwave display. so, I
// keep this reusable buffer globally for that task.
var reuseBuffer = make([]byte, SampleRate*SampleSize)

type Game struct {
	looper *edau.Looper
	filename string
	optIndex int // for UI
	player *audio.Player

	bufferStart [BufferViewLen][2]float64 // refreshed every time we move loop points and
	bufferEnd   [BufferViewLen][2]float64 // used for displaying the soundwave around them
}

func (self *Game) Layout(width, height int) (int, int) {
	return 680, 480
}

func (self *Game) Update() error {
	loopStart := self.looper.GetLoopStart()
	loopEnd   := self.looper.GetLoopEnd()

	// handle space presses to start / stop the audio
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if self.player == nil {
			playbackStart := loopEnd - (PlaybackPreMillis*SampleRate*SampleSize)/1000
			if playbackStart < loopStart { playbackStart = loopStart }
			self.looper.Seek(playbackStart, io.SeekStart)

			var err error
			self.player, err = audio.CurrentContext().NewPlayer(self.looper)
			if err != nil { return err }
			self.player.Play()
		} else {
			self.player.Pause()
			self.player = nil
		}
		return nil // do not handle any other input
	}
	
	// handle key presses
	change := int64(1) // change magnitude for left/right arrows
	if ebiten.IsKeyPressed(ebiten.KeyShift)   { change = 10  }
	if ebiten.IsKeyPressed(ebiten.KeyControl) { change = 100 }
	if ebiten.IsKeyPressed(ebiten.KeyControl) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		change = 1000
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		if self.optIndex > 0 { self.optIndex -= 1 }
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		if self.optIndex < 3 { self.optIndex += 1 }
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		switch self.optIndex {
		case 0: // start seconds
			newLoopStart := loopStart - SampleRate*SampleSize*change
			if newLoopStart < 0 { newLoopStart = 0 }
			self.looper.AdjustLoop(newLoopStart, loopEnd)
		case 1: // start samples
			newLoopStart := loopStart - SampleSize*change
			if newLoopStart < 0 { newLoopStart = 0 }
			self.looper.AdjustLoop(newLoopStart, loopEnd)
		case 2: // end seconds
			newLoopEnd := loopEnd - SampleRate*SampleSize*change
			if newLoopEnd < loopStart + SampleRate*SampleSize {
				newLoopEnd = loopStart + SampleRate*SampleSize
			}
			if newLoopEnd < loopEnd {
				self.looper.AdjustLoop(loopStart, newLoopEnd)
			}
		case 3: // end samples
			newLoopEnd := loopEnd - SampleSize*change
			if newLoopEnd < loopStart + SampleSize {
				newLoopEnd = loopStart + SampleSize
			}
			if newLoopEnd < loopEnd {
				self.looper.AdjustLoop(loopStart, newLoopEnd)
			}
		default:
			panic("bad code, bad!")
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		switch self.optIndex {
		case 0: // start seconds
			newLoopStart := loopStart + SampleRate*SampleSize*change
			if newLoopStart > loopEnd - SampleRate*SampleSize {
				newLoopStart = loopEnd - SampleRate*SampleSize
			}
			if newLoopStart > loopStart {
				self.looper.AdjustLoop(newLoopStart, loopEnd)
			}
		case 1: // start samples
			newLoopStart := loopStart + SampleSize*change
			if newLoopStart > loopEnd - SampleSize {
				newLoopStart = loopEnd - SampleSize
			}
			if newLoopStart > loopStart {
				self.looper.AdjustLoop(newLoopStart, loopEnd)
			}
		case 2: // end seconds
			newLoopEnd := loopEnd + SampleRate*SampleSize*change
			looperStreamLen := self.looper.Length()
			if newLoopEnd > looperStreamLen { newLoopEnd = looperStreamLen }
			self.looper.AdjustLoop(loopStart, newLoopEnd)
		case 3: // end samples
			newLoopEnd := loopEnd + SampleSize*change
			looperStreamLen := self.looper.Length()
			if newLoopEnd > looperStreamLen { newLoopEnd = looperStreamLen }
			self.looper.AdjustLoop(loopStart, newLoopEnd)
		default:
			panic("bad code, bad!")
		}
	}

	newLoopStart := self.looper.GetLoopStart()
	newLoopEnd   := self.looper.GetLoopEnd()
	if loopStart != newLoopStart || loopEnd != newLoopEnd {
		if self.player != nil {
			self.player.Pause()
			self.player = nil
		}
		self.refreshViewBuffers()
	}

	return nil
}

func (self *Game) refreshViewBuffers() {
	// joint view
	start, end := self.looper.GetLoopPoints()
	leftLim, rightLim := start, end
	
	// read the start buffer (shown on the right)
	from := start
	self.readSection(&self.bufferStart, from, leftLim, rightLim)
	
	// read the end buffer (shown on the left)
	from = end - BufferViewLen*SampleSize
	self.readSection(&self.bufferEnd, from, leftLim, rightLim)
}

func (self *Game) readSection(buffer *[BufferViewLen][2]float64, start int64, leftLim, rightLim int64) {
	end := start + BufferViewLen*SampleSize
	missingBytes := int64(0)
	
	// apply left limit
	i := 0
	if start < leftLim {
		missingBytes = leftLim - start
		for missingBytes > 0 {
			buffer[i] = [2]float64{ 0, 0 }
			missingBytes -= 4
			i += 1
		}
		missingBytes = 0
	}
	
	// adjust based on right limit (will be applied at the end)
	if end > rightLim {
		missingBytes = rightLim - end
		end = rightLim
	}

	// seek and read
	nSeek, err := self.looper.Seek(start, io.SeekStart)
	if err != nil { panic(err) }
	if nSeek != start { panic("n != start") }
	bytesToRead := end - start
	readBuffer := reuseBuffer[0 : bytesToRead] 
	nRead, err := self.looper.Read(readBuffer)
	if err != nil {
		if errors.Is(err, io.EOF) && int64(nRead) != bytesToRead {
			panic(err)
		}
	}
	if int64(nRead) != bytesToRead { panic("looper short read") }
	for len(readBuffer) > 0 {
		left, right := edau.GetSampleAsF64(readBuffer)
		readBuffer = readBuffer[4:]
		buffer[i]  = [2]float64{ left, right }
		i += 1
	}

	// apply right limit
	for missingBytes > 0 {
		buffer[i] = [2]float64{ 0, 0 }
		missingBytes -= 4
		i += 1
	}
}

func (self *Game) Draw(screen *ebiten.Image) {
	const yStride = 16
	const yStrideExtra = yStride + 10
	const pad = 12

	self.drawSamples(screen)

	loopStart := self.looper.GetLoopStart()
	loopEnd   := self.looper.GetLoopEnd()
	loopStartSample := loopStart/SampleSize
	loopEndSample   := loopEnd/SampleSize
	loopStartSecond := loopStartSample/SampleRate
	loopEndSecond   := loopEndSample/SampleRate
	loopFractStartMs := (1000*(loopStartSample - (loopStartSecond*SampleRate)))/SampleRate
	loopFractEndMs   := (1000*(loopEndSample   - (loopEndSecond  *SampleRate)))/SampleRate
	
	loopStartSampleStr := strconv.FormatInt(loopStartSample, 10)
	loopEndSampleStr   := strconv.FormatInt(loopEndSample  , 10)
	loopFractStartMsStr := " (+" + strconv.FormatInt(loopFractStartMs, 10) + "ms)"
	loopFractEndMsStr   := " (+" + strconv.FormatInt(loopFractEndMs, 10) + "ms)"
	
	x, y := pad, pad
	ebitenutil.DebugPrintAt(screen, "File: " + self.filename, x, y) ; y += yStrideExtra
	ebitenutil.DebugPrintAt(screen, "Loop Start: ", x, y) ; y += yStride
	if self.optIndex == 0 { self.drawSelectorAt(x, y, screen) }
	ebitenutil.DebugPrintAt(screen, "> Second: " + fmtSecNice(loopStartSecond) + loopFractStartMsStr, x, y) ; y += yStride
	if self.optIndex == 1 { self.drawSelectorAt(x, y, screen) }
	ebitenutil.DebugPrintAt(screen, "> Sample: " + loopStartSampleStr, x, y) ; y += yStrideExtra
	ebitenutil.DebugPrintAt(screen, "Loop End: ", x, y) ; y += yStride
	if self.optIndex == 2 { self.drawSelectorAt(x, y, screen) }
	ebitenutil.DebugPrintAt(screen, "> Second: " + fmtSecNice(loopEndSecond) + loopFractEndMsStr, x, y) ; y += yStride
	if self.optIndex == 3 { self.drawSelectorAt(x, y, screen) }
	ebitenutil.DebugPrintAt(screen, "> Sample: " + loopEndSampleStr, x, y)
	
	spaceAction := "start" // or "stop"
	if self.player != nil { spaceAction = "stop" }
	ebitenutil.DebugPrintAt(screen, "(press SPACE to " + spaceAction + " playing) (hold SHIFT / CTRL for x10 / x100 / x1000 changes)", x, 480 - pad - yStride)
}

func (self *Game) drawSamples(screen *ebiten.Image) {
	x, y := 24, 300
	
	leftSampleColor  := color.RGBA{ 255,    0, 255, 255 }
	rightSampleColor := color.RGBA{ 128,   32,  64, 255 }
	for _, sample := range self.bufferEnd {
		self.drawSampleValue(screen, sample[0], leftSampleColor, x, y)
		x += 2
		self.drawSampleValue(screen, sample[1], rightSampleColor, x, y)
		x += 3
	}

	leftSampleColor  = color.RGBA{   0, 255, 255, 255 }
	rightSampleColor = color.RGBA{  64,  64, 255, 255 }
	for _, sample := range self.bufferStart {
		self.drawSampleValue(screen, sample[0], leftSampleColor, x, y)
		x += 2
		self.drawSampleValue(screen, sample[1], rightSampleColor, x, y)
		x += 3
	}
}

func (self *Game) drawSampleValue(screen *ebiten.Image, value float64, clr color.RGBA, x, y int) {
	if value > 1.0 || value < -1.0 { panic(value) }
	height := int(value*96.0)
	if height == 0 { height = 1 }
	screen.SubImage(image.Rect(x, y, x + 2, y - height)).(*ebiten.Image).Fill(clr)
}

func (self *Game) drawSelectorAt(x, y int, screen *ebiten.Image) {
	screen.SubImage(image.Rect(x - 2, y + 1, x + 256, y + 16)).(*ebiten.Image).Fill(color.RGBA{0, 128, 128, 255})
}

func fmtSecNice(seconds int64) string {
	var hours, mins int64
	if seconds > 3600 {
		hours = seconds/3600
		seconds = seconds % 3600
	}
	if seconds > 60 {
		mins = seconds/60
		seconds = seconds % 60
	}

	if hours != 0 {
		return fmt.Sprintf("%dh %02dm %02ds", hours, mins, seconds)
	} else if mins != 0 {
		return fmt.Sprintf("%dm %02ds", mins, seconds)
	} else {
		return fmt.Sprintf("%ds", seconds)
	}
}

func main() {
	// get argument (path to audio file to find looping points for)
	if len(os.Args) != 2 {
		fmt.Print("Usage: expects one argument pointing to the audio file to use.\n")
		os.Exit(1)
	}
	filename, err := filepath.Abs(os.Args[1])
	if err != nil { log.Fatal(err) }
	
	// open stream
	audio.NewContext(SampleRate) // retrieved later with audio.CurrentContext()
	stream, err := edau.LoadAudioFileAsStream(filename)
	if err != nil { log.Fatal(err) }
	looper := edau.NewLooper(stream, 0, stream.Length())

	// create "game" and start program
	game := &Game{ looper: looper, filename: filepath.Base(filename) }
	game.refreshViewBuffers()
	ebiten.SetWindowSize(680, 480)
	ebiten.SetWindowTitle("edau loop finder")
	err = ebiten.RunGame(game)
	if err != nil { log.Fatal(err) }
}