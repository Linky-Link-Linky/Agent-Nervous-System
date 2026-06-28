package tuiengine

import (
    "github.com/hajimehoshi/ebiten/v2"
)

func KeyToANSI(key ebiten.Key) []byte {
    switch key {
    case ebiten.KeyF1:     return []byte("\x1b[11~")
    case ebiten.KeyF2:     return []byte("\x1b[12~")
    case ebiten.KeyF3:     return []byte("\x1b[13~")
    case ebiten.KeyF4:     return []byte("\x1b[14~")
    case ebiten.KeyF5:     return []byte("\x1b[15~")
    case ebiten.KeyF6:     return []byte("\x1b[17~")
    case ebiten.KeyF7:     return []byte("\x1b[18~")
    case ebiten.KeyF8:     return []byte("\x1b[19~")
    case ebiten.KeyF9:     return []byte("\x1b[20~")
    case ebiten.KeyF10:    return []byte("\x1b[21~")
    case ebiten.KeyF11:    return []byte("\x1b[23~")
    case ebiten.KeyF12:    return []byte("\x1b[24~")
    case ebiten.KeyTab:    return []byte("\t")
    case ebiten.KeyEnter:  return []byte("\r")
    case ebiten.KeyEscape: return []byte("\x1b")
    case ebiten.KeyBackspace: return []byte("\b")
    case ebiten.KeySpace:  return []byte(" ")
    case ebiten.KeyUp:     return []byte("\x1b[A")
    case ebiten.KeyDown:   return []byte("\x1b[B")
    case ebiten.KeyRight:  return []byte("\x1b[C")
    case ebiten.KeyLeft:   return []byte("\x1b[D")
    case ebiten.KeyHome:   return []byte("\x1b[H")
    case ebiten.KeyEnd:    return []byte("\x1b[F")
    case ebiten.KeyPageUp:   return []byte("\x1b[5~")
    case ebiten.KeyPageDown: return []byte("\x1b[6~")
    case ebiten.KeyDelete: return []byte("\x1b[3~")
    case ebiten.KeyInsert: return []byte("\x1b[2~")
    default:
        if key >= ebiten.KeyA && key <= ebiten.KeyZ {
            return []byte(key.String())
        }
        return nil
    }
}


