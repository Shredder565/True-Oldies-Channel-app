package main

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// ICY metadata stripping: Icecast/Shoutcast servers interleave a small
// metadata block every N bytes when the client asks for it. We request it so
// we can show the song title, then remove it so the MP3 decoder never sees it.
// ---------------------------------------------------------------------------

type icyReader struct {
	r        *bufio.Reader
	metaInt  int
	toMeta   int
	onTitle  func(string)
	lastTitl string
}

func newICYReader(r io.Reader, metaInt int, onTitle func(string)) *icyReader {
	return &icyReader{r: bufio.NewReaderSize(r, 64*1024), metaInt: metaInt, toMeta: metaInt, onTitle: onTitle}
}

func (c *icyReader) Read(p []byte) (int, error) {
	if c.metaInt <= 0 {
		return c.r.Read(p)
	}
	if c.toMeta == 0 {
		if err := c.readMeta(); err != nil {
			return 0, err
		}
		c.toMeta = c.metaInt
	}
	if len(p) > c.toMeta {
		p = p[:c.toMeta]
	}
	n, err := c.r.Read(p)
	c.toMeta -= n
	return n, err
}

func (c *icyReader) readMeta() error {
	lenByte, err := c.r.ReadByte()
	if err != nil {
		return err
	}
	size := int(lenByte) * 16
	if size == 0 {
		return nil
	}
	buf := make([]byte, size)
	if _, err := io.ReadFull(c.r, buf); err != nil {
		return err
	}
	meta := string(bytes.TrimRight(buf, "\x00"))
	logf("metadata: %s", meta)
	if title, ok := parseStreamTitle(meta); ok {
		if title != c.lastTitl && c.onTitle != nil {
			c.lastTitl = title
			c.onTitle(title)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Volume control: scale 16-bit little-endian stereo PCM coming out of the decoder.
// ---------------------------------------------------------------------------

type volumeReader struct {
	src io.Reader
	mu  sync.RWMutex
	vol float64 // 0.0 .. 1.0
}

func (v *volumeReader) Set(vol float64) {
	if vol < 0 {
		vol = 0
	}
	if vol > 1 {
		vol = 1
	}
	v.mu.Lock()
	v.vol = vol
	v.mu.Unlock()
}

func (v *volumeReader) Get() float64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.vol
}

func (v *volumeReader) Read(p []byte) (int, error) {
	n, err := v.src.Read(p)
	vol := v.Get()
	if vol != 1.0 {
		for i := 0; i+1 < n; i += 2 {
			s := int16(uint16(p[i]) | uint16(p[i+1])<<8)
			f := float64(s) * vol
			if f > 32767 {
				f = 32767
			} else if f < -32768 {
				f = -32768
			}
			s = int16(f)
			p[i] = byte(uint16(s))
			p[i+1] = byte(uint16(s) >> 8)
		}
	}
	return n, err
}


// parseStreamTitle extracts StreamTitle='...' (or "...") from an ICY metadata block.
func parseStreamTitle(meta string) (string, bool) {
	i := strings.Index(meta, "StreamTitle=")
	if i < 0 {
		return "", false
	}
	rest := meta[i+len("StreamTitle="):]
	if rest == "" {
		return "", false
	}
	q := rest[0]
	if q == '\'' || q == '"' {
		rest = rest[1:]
		if j := strings.Index(rest, string(q)+";"); j >= 0 {
			rest = rest[:j]
		} else if j := strings.LastIndex(rest, string(q)); j >= 0 {
			rest = rest[:j]
		}
	} else if j := strings.Index(rest, ";"); j >= 0 {
		rest = rest[:j]
	}
	title := strings.TrimSpace(rest)
	return title, title != ""
}
