package foreground

import (
	"bytes"
	"encoding/binary"
)

// XNU aligns argv after executable_path= plus the executable string to the
// target process's pointer size. The stripped prefix is 16 bytes, aligned for
// both supported layouts. Skip exactly that padding, never empty argv slots:
// an erased argv-zero must not expose a later argument or environment string.
// Only complete, below-capacity native reads may be passed here.
func parseDarwinArgvZero(buf []byte, pointerSize int) string {
	if len(buf) < 4 || len(buf) >= maxProgramBytes || (pointerSize != 4 && pointerSize != 8) {
		return ""
	}
	argc := int(binary.NativeEndian.Uint32(buf[:4]))
	if argc < 1 || argc > 128 {
		return ""
	}
	buf = buf[4:]
	i := bytes.IndexByte(buf, 0) // executable path
	if i <= 0 || i > 1024 {
		return ""
	}
	start := (i + 1 + pointerSize - 1) & ^(pointerSize - 1)
	if start >= len(buf) {
		return ""
	}
	for _, padding := range buf[i+1 : start] {
		if padding != 0 {
			return ""
		}
	}
	end := bytes.IndexByte(buf[start:], 0)
	if end <= 0 {
		return ""
	}
	return string(buf[start : start+end])
}
