// Radio: a vintage-styled internet-radio player for one MP3 stream.
package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"

	webview "github.com/jchv/go-webview2"
)

const defaultURL = "https://ais-edge89-dal02.cdnstream.com/b92108_128mp3"

//go:embed ui.html
var uiHTML string

func main() {
	url := flag.String("url", defaultURL, "stream URL")
	startVol := flag.Int("volume", 80, "starting volume (0-100)")
	noAuto := flag.Bool("standby", false, "start powered off instead of playing")
	flag.Parse()

	initLog()
	engine := NewEngine(*url, float64(*startVol)/100)

	// WebView2 needs a writable folder for its cache; the exe's folder may not be.
	dataPath := filepath.Join(os.Getenv("LOCALAPPDATA"), "TrueOldiesChannel", "webview")
	_ = os.MkdirAll(dataPath, 0o755)

	w := webview.NewWithOptions(webview.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		DataPath:  dataPath,
		WindowOptions: webview.WindowOptions{
			Title:  "True Oldies Channel",
			Width:  700,
			Height: 480,
			Center: true,
		},
	})
	if w == nil {
		fatalBox("The Microsoft Edge WebView2 Runtime is required but was not found.\n\nInstall it from https://developer.microsoft.com/microsoft-edge/webview2/ and run True Oldies Channel again.")
		return
	}
	defer w.Destroy()
	w.SetSize(700, 480, webview.HintFixed)

	// Engine -> UI
	engine.OnEvent = func(ev Event) {
		b, _ := json.Marshal(ev)
		js := "window.onEvent && window.onEvent(" + string(b) + ")"
		w.Dispatch(func() {
			w.Eval(js)
			if ev.Kind == EvTitle {
				w.SetTitle("♪ " + ev.Text + "  —  True Oldies Channel")
			}
		})
	}

	// UI -> Engine
	w.Bind("radio_setVolume", func(v float64) { engine.SetVolume(v) })
	w.Bind("radio_power", func(on bool) {
		if on {
			engine.Start()
		} else {
			go engine.Stop() // Stop blocks briefly; keep the UI thread free
		}
	})
	w.Bind("radio_state", func() map[string]any {
		return map[string]any{"on": engine.IsRunning(), "volume": engine.Volume()}
	})
	// Namespace the bindings as window.radio.*
	w.Init(`window.radio = {
		setVolume: v => window.radio_setVolume(v),
		power: on => window.radio_power(on),
		state: () => window.radio_state(),
	};`)

	w.SetHtml(uiHTML)

	if !*noAuto {
		engine.Start()
	}
	w.Run()
	engine.Stop()
}
