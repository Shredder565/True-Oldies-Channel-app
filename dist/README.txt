True Oldies Channel
===================

Double-click TrueOldiesChannel.exe. The radio warms up and tunes in automatically.

Default stream: https://ais-edge89-dal02.cdnstream.com/b92108_128mp3

Controls:
  POWER knob    click to switch on/off (or press Space)
  VOLUME knob   drag up/down, mouse wheel anywhere, or arrow keys
  Dial display  shows the song now playing (also in the window title)
  Pilot lamp    amber = tuning in, green = playing, red = connection problem

If the connection drops it reconnects on its own.

Command-line options (optional):
  TrueOldiesChannel.exe -volume 50
  TrueOldiesChannel.exe -standby            start switched off
  TrueOldiesChannel.exe -url https://some-other-station/stream.mp3

Notes:
  * Single file, no install. Windows 10/11, 64-bit.
  * Uses the Microsoft Edge WebView2 Runtime for its window, which is already
    on virtually every Windows 10/11 PC. If it's missing you'll get a message
    with the download link.
  * Windows SmartScreen may warn on first launch because the exe isn't
    code-signed: "More info" -> "Run anyway".

Rebuilding from source (Go 1.24+):
  GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w -H windowsgui" -o TrueOldiesChannel.exe .

Troubleshooting song titles:
  If the display says the station isn't providing song info, a diagnostic log
  is written to  %LOCALAPPDATA%\TrueOldiesChannel\radio.log  showing exactly
  what the station sent.
