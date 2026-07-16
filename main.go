package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"unciv-mod-manager/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	backend := app.NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "Unciv Mod Manager",
		Width:     1400,
		Height:    900,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        backend.Startup,
		Bind: []interface{}{
			backend,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
