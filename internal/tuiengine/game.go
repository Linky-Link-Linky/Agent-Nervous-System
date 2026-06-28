package tuiengine

import (
    "bytes"
    "image/color"
    "io"
    "runtime"
    "time"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/audio"
    "github.com/hajimehoshi/ebiten/v2/audio/wav"
    "github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Game struct {
    bridge    *Bridge
    inputW    *io.PipeWriter
    shader    *ebiten.Shader
    offscreen *ebiten.Image
    bootDone  bool
    bootTick  int
    audioCtx  *audio.Context
    shaderOn  bool
    tintRGB   [3]float64
}

func NewGame(bridge *Bridge, inputW *io.PipeWriter, shaderData []byte) (*Game, error) {
    g := &Game{bridge: bridge, inputW: inputW, tintRGB: [3]float64{0.96, 1.0, 0.94}, shaderOn: true}
    if len(shaderData) > 0 {
        s, err := ebiten.NewShader(shaderData)
        if err == nil {
            g.shader = s
        }
    }
    g.offscreen = ebiten.NewImage(1280, 800)
    g.audioCtx = audio.NewContext(48000)
    return g, nil
}

func (g *Game) Update() error {
	if !g.bootDone {
		g.bootTick++
		if g.bootTick > 90 {
			g.bootDone = true
		}
	}
	g.bridge.Tick()
	for _, k := range allKeys {
		if ebiten.IsKeyPressed(k) {
			if seq := KeyToANSI(k); seq != nil {
				g.inputW.Write(seq)
			}
		}
	}
	return nil
}

var allKeys = []ebiten.Key{
	ebiten.KeyA, ebiten.KeyB, ebiten.KeyC, ebiten.KeyD, ebiten.KeyE,
	ebiten.KeyF, ebiten.KeyG, ebiten.KeyH, ebiten.KeyI, ebiten.KeyJ,
	ebiten.KeyK, ebiten.KeyL, ebiten.KeyM, ebiten.KeyN, ebiten.KeyO,
	ebiten.KeyP, ebiten.KeyQ, ebiten.KeyR, ebiten.KeyS, ebiten.KeyT,
	ebiten.KeyU, ebiten.KeyV, ebiten.KeyW, ebiten.KeyX, ebiten.KeyY, ebiten.KeyZ,
	ebiten.Key0, ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4,
	ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9,
	ebiten.KeyF1, ebiten.KeyF2, ebiten.KeyF3, ebiten.KeyF4, ebiten.KeyF5,
	ebiten.KeyF6, ebiten.KeyF7, ebiten.KeyF8, ebiten.KeyF9, ebiten.KeyF10,
	ebiten.KeyF11, ebiten.KeyF12,
	ebiten.KeyEnter, ebiten.KeyBackspace, ebiten.KeySpace,
	ebiten.KeyUp, ebiten.KeyDown, ebiten.KeyLeft, ebiten.KeyRight,
	ebiten.KeyTab, ebiten.KeyEscape,
	ebiten.KeyBackquote, ebiten.KeyComma, ebiten.KeyPeriod, ebiten.KeySlash,
	ebiten.KeySemicolon, ebiten.KeyQuote,
	ebiten.KeyLeftBracket, ebiten.KeyRightBracket, ebiten.KeyBackslash,
	ebiten.KeyMinus, ebiten.KeyEqual,
	ebiten.KeyDelete, ebiten.KeyHome, ebiten.KeyEnd,
	ebiten.KeyPageUp, ebiten.KeyPageDown, ebiten.KeyInsert,
}

func (g *Game) Draw(screen *ebiten.Image) {
    w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
    g.offscreen.Clear()
    g.offscreen.DrawImage(g.bridge.Frame(), nil)

    if g.shader != nil && g.shaderOn {
        opts := &ebiten.DrawRectShaderOptions{}
        opts.Uniforms = map[string]any{
            "Resolution": []float32{float32(w), float32(h)},
            "Time":       float32(g.bootTick),
        }
        opts.Images[0] = g.offscreen
        screen.DrawRectShader(w, h, g.shader, opts)
    } else {
        screen.DrawImage(g.offscreen, nil)
    }

    if !g.bootDone {
        // scanline wipe
        progress := float64(g.bootTick-20) / 30.0
        if progress < 0 { progress = 0 }
        if progress > 1 { progress = 1 }
        revealH := int(float64(h) * progress)
        ebitenutil.DrawRect(screen, 0, float64(revealH), float64(w), float64(h-revealH), color.RGBA{0, 0, 0, 255})
    }
}

func (g *Game) Layout(w, h int) (int, int) {
    g.bridge.Resize(w, h)
    if g.offscreen == nil || g.offscreen.Bounds().Dx() != w || g.offscreen.Bounds().Dy() != h {
        g.offscreen = ebiten.NewImage(w, h)
    }
    return w, h
}

func (g *Game) SetShaderOn(on bool) { g.shaderOn = on }
func (g *Game) SetTint(r, gr, b float64) { g.tintRGB = [3]float64{r, gr, b} }

func PlayBootSoundData(data []byte) {
    if len(data) == 0 || runtime.GOARCH == "wasm" { return }
    ctx := audio.NewContext(48000)
    s, err := wav.Decode(ctx, bytes.NewReader(data))
    if err != nil { return }
    p, err := ctx.NewPlayer(s)
    if err != nil { return }
    p.Play()
    go func() {
        time.Sleep(100 * time.Millisecond)
        for p.IsPlaying() { time.Sleep(50 * time.Millisecond) }
        p.Close()
    }()
}
