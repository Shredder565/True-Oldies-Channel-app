package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var logger *log.Logger

// initLog writes a small diagnostic log to %LOCALAPPDATA%\TrueOldiesChannel\radio.log.
func initLog() {
	dir := filepath.Join(os.Getenv("LOCALAPPDATA"), "TrueOldiesChannel")
	_ = os.MkdirAll(dir, 0o755)
	f, err := os.OpenFile(filepath.Join(dir, "radio.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return
	}
	logger = log.New(f, "", log.LstdFlags)
}

func logf(format string, a ...any) {
	if logger != nil {
		logger.Output(2, fmt.Sprintf(format, a...))
	}
}
