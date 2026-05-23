package main

import (
	"flag"
	"log"

	"filestore/internal/config"
	"filestore/internal/handler"
	"filestore/internal/service"
	"filestore/pkg/database"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db := database.MustNewMySQL(cfg.MySQL)
	defer db.Close()

	fileSvc := service.NewFileService(db)

	r := handler.SetupRouter(cfg, fileSvc)

	log.Printf("filestore server (version: %s) starting on %s", config.Version, cfg.Server.Address)
	if err := r.Run(cfg.Server.Address); err != nil {
		log.Fatalf("server start failed: %v", err)
	}
}
