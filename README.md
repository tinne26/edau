# EDAU
[![Go Reference](https://pkg.go.dev/badge/github.com/tinne26/edau.svg)](https://pkg.go.dev/github.com/tinne26/edau)

**WARNING**: some discussions on Ebitengine's discord lead me to publish this earlier than I intended. Don't expect much yet.

EDAU stands for *Ebitengine Digital Audio Utils*. As the name implies, it's a collection of types, interfaces and utilities to work with audio in the [Ebitengine game engine](https://github.com/hajimehoshi/ebiten).

This is currently a loosely scoped, exploratory project for personal use mostly. If any of its parts end up evolving into something bigger, I may separate them into a more focused repository.

Current utilities include:
- SpeedShifter, which can be used to play audio at different speeds, and a few different interpolator functions.
- A tight audio looper that unlike Ebitengine's default looper doesn't do any fading between end and start loop points, and a loop points finder utility.

## General advice for audio in Ebitengine
- Use the .ogg format for all your audio. The implementation used for Ebitengine is around twice as performant as .mp3, it doesn't have random padding at the start and end of the audio like mp3, and given the same size the quality is better than mp3 too.
- If you have few SFX's, load the audio as bytes from an Ebitengine audio stream with `io.ReadAll` and use [`audio.NewPlayerFromBytes`](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/audio#Context.NewPlayerFromBytes) to trigger them. Keeping multiple players reading from underlying compressed streams can be quite taxing, especially on WASM.
- Consider using [`audio#Player.SetBufferSize`](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/audio#Player.SetBufferSize) if you need responsive effects or real-time*-ish* audio. For desktops, you can typically go slightly below `time.Millisecond*20`. For browsers I don't recommend going below `time.Millisecond*50`.

## Pending work
Future plans include:
- Transitions for effects and the speed shifter. This is currently an open problem that I'm still trying to figure out, as applying transitions as early as possible leads to inconsistent transition timing, and trying to apply a consistent latency seems tricky (if possible at all).
- Better support for .ogg and a mono player, mostly intended for SFX's and saving some space.
- Effects and effects chains, with programmable and automatable parameters and a simple descriptive UI model to be able to integrate plugins into a DAW-like app.
- A few audio synths. Maybe even some light format to create SFX's programmatically.
- Maybe more sophisticate loopers for more complex song structures with repetitions and multiple sections, though this can already be built on top of the existing looper.

## See also...
- SolarLune's [resound](https://github.com/SolarLune/resound) library, which already includes multiple effects like volume, delay, low-pass filtering, panning, distortion, etc.
