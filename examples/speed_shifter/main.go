package main

import "os"
import "io"
import "log"
import "fmt"
import "time"
import "strconv"

import "github.com/hajimehoshi/ebiten/v2"
import "github.com/hajimehoshi/ebiten/v2/audio"
import "github.com/hajimehoshi/ebiten/v2/ebitenutil"
import "github.com/hajimehoshi/ebiten/v2/inpututil"

import "github.com/tinne26/edau"

// TODO: add frequency spectrum for the visuals?

// ---- game implementation with up/down input to modify speed ----
type Game struct {
	audioSrc *edau.SpeedShifter
	speed float64
}
func (self *Game) Layout(w, h int) (int, int) { return w, h }
func (self *Game) Update() error {
	change := 0.1
	if ebiten.IsKeyPressed(ebiten.KeyShift) { change = 0.01 }

	speed := self.audioSrc.Speed()
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && speed < 1.9 {
		speed += change
		if speed >= 1.9 { speed = 1.9 }
		self.audioSrc.SetSpeed(speed)
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) && speed > 0.1 {
		speed -= change
		if speed <= 0.1 { speed = 0.1 }
		self.audioSrc.SetSpeed(speed)
	}
	return nil
}
func (self *Game) Draw(screen *ebiten.Image) {
	speedStr := strconv.FormatFloat(self.audioSrc.Speed(), 'f', 2, 64)
	fpsStr := strconv.FormatFloat(ebiten.CurrentFPS(), 'f', 2, 64)
	str := fpsStr + "fps\nup/down to change audio speed\n"
	str += "hold SHIFT for smaller changes\n"
	str += "current speed factor: x" + speedStr
	ebitenutil.DebugPrint(screen, str)
}

// ---- main function that sets up the speed shifter and starts game ----

func main() {
	const ErrStatus = 1
	
	// usage
	if len(os.Args) != 2 {
		fmt.Print("Usage: go run examples/speed_shifter/main.go path/to/audio_file.{ogg|wav|mp3}\n")
		os.Exit(1)
	}

	// init audio context and load file
	ctx := audio.NewContext(44100)
	stream, err := edau.LoadAudioFileAsStream(os.Args[1])
	if err != nil { log.Fatal(err) }

	// create speed shifter and start playing audio
	shifter := edau.NewDefaultSpeedShifter(stream)
	ebiten.SetWindowTitle("speed shifter")
	player, err := ctx.NewPlayer(shifter)
	if err != nil { log.Fatal(err) }
	player.SetBufferSize(time.Millisecond*100)
	player.Play()

	// start the game, which allows modifying the playback speed
	err = ebiten.RunGame(&Game{ audioSrc: shifter, speed: 1.0 })
	if err != nil { log.Fatal(err) }

	err = stream.(io.Closer).Close()
	if err != nil { log.Fatal(err) }
}
