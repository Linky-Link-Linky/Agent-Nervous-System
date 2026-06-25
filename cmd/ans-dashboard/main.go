package main

import (
	"log"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/dashboard"
)

func main() {
	app := dashboard.NewApp()
	if err := app.Run(); err != nil {
		log.Fatalf("dashboard error: %v", err)
	}
}
