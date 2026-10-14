package main

import (
	"os"

	"awesomeProject/internal/cli"
	"awesomeProject/internal/service"
	"awesomeProject/internal/storage"
	"awesomeProject/pkg/logger"
)

func main() {
	log := logger.New()
	store := storage.NewFileStore("tasks.json")
	svc := service.New(store)
	app := cli.New(svc, os.Stdin, os.Stdout, os.Stderr)
	if err := app.Run(); err != nil {
		log.Error("ошибка: %v", err)
		os.Exit(1)
	}
}
