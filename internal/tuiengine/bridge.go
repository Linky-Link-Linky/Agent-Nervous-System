package tuiengine

import (
    "bytes"
    "image/color"
    "io"
    "sync"
    "unicode/utf8"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/text"
    "golang.org/x/image/font"
    "golang.org/x/image/font/opentype"
)

type Cell struct {
    R    rune
    Fg   color.RGBA
    Bg   color.RGBA
    Bold bool
}

type Bridge struct {
    grid     [][]Cell
    rows, cols int
    cellW, cellH int
    font     font.Face
    frame    *ebiten.Image
    src      *io.PipeReader
    mu       sync.Mutex
    buf      bytes.Buffer
}

func NewBridge(src *io.PipeReader, fontData []byte) (*Bridge, error) {
    tt, err := opentype.Parse(fontData)
    if err != nil {
        return nil, err
    }
    face, err := opentype.NewFace(tt, &opentype.FaceOptions{
        Size: 14, DPI: 72, Hinting: font.HintingNone,
    })
    if err != nil {
        return nil, err
    }
    b := &Bridge{
        font: face, src: src,
        cellW: 9, cellH: 18,
        cols: 140, rows: 44,
    }
    b.cellW = 9
    if b.cellW < 1 { b.cellW = 9 }
    b.cellH = face.Metrics().Height.Ceil()
    if b.cellH < 1 { b.cellH = 18 }
    b.resizeGrid()
    return b, nil
}

func (b *Bridge) resizeGrid() {
    b.grid = make([][]Cell, b.rows)
    for r := range b.grid {
        b.grid[r] = make([]Cell, b.cols)
    }
    b.frame = ebiten.NewImage(b.cols*b.cellW, b.rows*b.cellH)
}

func (b *Bridge) Resize(w, h int) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.cols = w / b.cellW
    b.rows = h / b.cellH
    if b.cols < 20 { b.cols = 20 }
    if b.rows < 10 { b.rows = 10 }
    b.resizeGrid()
}

func (b *Bridge) Tick() {
    tmp := make([]byte, 4096)
    n, err := b.src.Read(tmp)
    if err != nil || n == 0 {
        return
    }
    b.mu.Lock()
    b.buf.Write(tmp[:n])
    // Simple ANSI parser: split by newlines, accumulate chars per row
    for b.buf.Len() > 0 {
        line, err := b.buf.ReadBytes('\n')
        if err != nil {
            break
        }
        b.parseLine(line)
    }
    b.renderFrame()
    b.mu.Unlock()
}

func (b *Bridge) parseLine(line []byte) {
    // Minimal ANSI parser: strip escape sequences, collect runes
    var fg, bg color.RGBA
    fg = color.RGBA{0x39, 0xff, 0x14, 0xff} // default phosphor green
    bg = color.RGBA{0x0a, 0x0f, 0x05, 0xff}
    col := 0
    row := 0
    if len(b.grid) > 0 && len(b.grid[0]) > 0 {
        // shift rows up
        for r := 0; r < len(b.grid)-1; r++ {
            copy(b.grid[r], b.grid[r+1])
        }
        // clear last row
        for c := range b.grid[len(b.grid)-1] {
            b.grid[len(b.grid)-1][c] = Cell{}
        }
        row = len(b.grid) - 1
    }
    for i := 0; i < len(line); i++ {
        if line[i] == 0x1b && i+1 < len(line) && line[i+1] == '[' {
            i += 2
            for i < len(line) && (line[i] < 'A' || line[i] > 'z') {
                i++
            }
            continue
        }
        r, size := utf8.DecodeRune(line[i:])
        if r == utf8.RuneError { continue }
        if row < len(b.grid) && col < len(b.grid[row]) {
            b.grid[row][col] = Cell{R: r, Fg: fg, Bg: bg}
        }
        col++
        i += size - 1
    }
}

func (b *Bridge) renderFrame() {
    b.frame.Clear()
    for r := 0; r < len(b.grid) && r < b.rows; r++ {
        for c := 0; c < len(b.grid[r]) && c < b.cols; c++ {
            cell := b.grid[r][c]
            if cell.R == 0 {
                continue
            }
            x := c * b.cellW
            y := r * b.cellH
            text.Draw(b.frame, string(cell.R), b.font, x, y+b.cellH-4, cell.Fg)
        }
    }
}

func (b *Bridge) Frame() *ebiten.Image {
    b.mu.Lock()
    defer b.mu.Unlock()
    return b.frame
}
