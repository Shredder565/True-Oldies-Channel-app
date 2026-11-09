package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
)

// Event kinds sent to the UI.
const (
	EvState   = "state"   // connecting | playing | off | error
	EvTitle   = "title"   // now-playing text
	EvStation = "station" // station name from icy-name
	EvMessage = "message" // human-readable detail (errors etc.)
)

type Event struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// Engine streams one MP3 URL to the default audio device.
type Engine struct {
	URL     string
	OnEvent func(Event)

	vol *volumeReader

	mu      sync.Mutex
	running bool
	stop    chan struct{}
	done    chan struct{}

	ctx     *oto.Context
	ctxRate int
}

func NewEngine(url string, volume float64) *Engine {
	return &Engine{URL: url, vol: &volumeReader{vol: volume}}
}

func (e *Engine) emit(kind, text string) {
	if e.OnEvent != nil {
		e.OnEvent(Event{Kind: kind, Text: text})
	}
}

func (e *Engine) SetVolume(v float64) { e.vol.Set(v) }
func (e *Engine) Volume() float64     { return e.vol.Get() }

func (e *Engine) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

// Start begins streaming (no-op if already running).
func (e *Engine) Start() {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return
	}
	e.running = true
	e.stop = make(chan struct{})
	e.done = make(chan struct{})
	stop, done := e.stop, e.done
	e.mu.Unlock()
	go e.loop(stop, done)
}

// Stop halts streaming and waits for the audio to stop.
func (e *Engine) Stop() {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return
	}
	e.running = false
	close(e.stop)
	done := e.done
	e.mu.Unlock()
	<-done
	e.emit(EvState, "off")
}

var errStopped = errors.New("stopped")

func (e *Engine) loop(stop, done chan struct{}) {
	defer close(done)
	backoff := time.Second
	for {
		select {
		case <-stop:
			return
		default:
		}
		e.emit(EvState, "connecting")
		err := e.playOnce(stop)
		if errors.Is(err, errStopped) {
			return
		}
		e.emit(EvState, "error")
		e.emit(EvMessage, fmt.Sprintf("%v — retrying in %s", err, backoff))
		select {
		case <-stop:
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (e *Engine) playOnce(stop chan struct{}) error {
	req, err := http.NewRequest("GET", e.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Icy-MetaData", "1")
	req.Header.Set("User-Agent", "TrueOldiesChannel/1.0")
	// Force HTTP/1.1: Icecast/Shoutcast servers only inject ICY metadata on plain HTTP/1.x.
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	tr.ForceAttemptHTTP2 = false
	resp, err := (&http.Client{Transport: tr}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("server returned HTTP %d", resp.StatusCode)
	}
	if name := resp.Header.Get("icy-name"); name != "" {
		e.emit(EvStation, name)
	}
	metaInt, _ := strconv.Atoi(resp.Header.Get("icy-metaint"))
	logf("connected: HTTP %d, icy-name=%q icy-metaint=%d content-type=%q", resp.StatusCode, resp.Header.Get("icy-name"), metaInt, resp.Header.Get("Content-Type"))
	if metaInt == 0 {
		e.emit(EvMessage, "station is not sending song titles")
	}

	src := newICYReader(resp.Body, metaInt, func(t string) { e.emit(EvTitle, t) })
	dec, err := mp3.NewDecoder(src)
	if err != nil {
		return fmt.Errorf("mp3 decoder: %w", err)
	}

	if e.ctx == nil {
		op := &oto.NewContextOptions{
			SampleRate:   dec.SampleRate(),
			ChannelCount: 2,
			Format:       oto.FormatSignedInt16LE,
			BufferSize:   500 * time.Millisecond,
		}
		c, ready, err := oto.NewContext(op)
		if err != nil {
			return fmt.Errorf("audio device: %w", err)
		}
		<-ready
		e.ctx = c
		e.ctxRate = dec.SampleRate()
	}

	e.vol.src = dec
	p := e.ctx.NewPlayer(e.vol)
	p.SetBufferSize(2 * 1024 * 1024) // ~12 s of audio to absorb network hiccups
	p.Play()
	e.emit(EvState, "playing")

	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-stop:
			p.Close()
			return errStopped
		case <-t.C:
			if !p.IsPlaying() {
				perr := p.Err()
				p.Close()
				if perr != nil {
					return perr
				}
				return errors.New("stream ended")
			}
		}
	}
}
