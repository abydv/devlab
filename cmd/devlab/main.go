// Command devlab is the entrypoint for the DevLab CLI and REST API server.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/abydv/devlab/api"
	"github.com/abydv/devlab/internal/config"
	"github.com/abydv/devlab/internal/engine"
	"github.com/abydv/devlab/internal/runtime/docker"
	"github.com/abydv/devlab/internal/runtime/k3d"
	"github.com/abydv/devlab/internal/runtime/shell"
	"github.com/abydv/devlab/internal/service/factory"
	"github.com/abydv/devlab/internal/storage"
	"github.com/abydv/devlab/internal/template"
	"github.com/abydv/devlab/internal/workspace"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "0.0.0-dev"

func main() {
	showVersion := flag.Bool("version", false, "print the devlab version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Fprintf(os.Stdout, "devlab %s\n", version)
		return
	}

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := storage.Open(cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	workspaces, err := workspace.NewManager(cfg.WorkspacesDir, db)
	if err != nil {
		return fmt.Errorf("init workspace manager: %w", err)
	}

	templates := template.NewRegistry(cfg.TemplatesDir)
	if err := templates.Load(); err != nil {
		return fmt.Errorf("load templates: %w", err)
	}

	sh := shell.New()
	services := factory.New(k3d.New(sh), docker.New(sh))

	e := engine.New(workspaces, templates, services)

	app := api.New(e)

	log.Printf("devlab %s listening on %s", version, cfg.ListenAddr)
	return app.Listen(cfg.ListenAddr)
}
