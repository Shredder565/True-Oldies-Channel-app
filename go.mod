module radio

go 1.24

require (
	github.com/ebitengine/oto/v3 v3.3.3
	github.com/hajimehoshi/go-mp3 v0.3.4
	github.com/jchv/go-webview2 v0.0.0-20260205173254-56598839c808
)

require (
	github.com/ebitengine/purego v0.8.4 // indirect
	github.com/jchv/go-winloader v0.0.0-20250406163304-c1995be93bd1 // indirect
	golang.org/x/sys v0.25.0 // indirect
)

replace golang.org/x/sys => github.com/golang/sys v0.33.0

replace github.com/ebitengine/purego => github.com/ebitengine/purego v0.8.4
