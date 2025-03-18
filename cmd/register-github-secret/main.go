package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aereal/github-ops/internal/cli"
	"github.com/aereal/github-ops/internal/log"
	"github.com/aereal/github-ops/internal/usecases"
	"github.com/google/go-github/v69/github"
)

func main() { os.Exit(run()) }

func run() int {
	log.Setup()
	ctx := context.Background()
	client := github.NewClient(nil).WithAuthToken(os.Getenv("GITHUB_TOKEN"))
	uc := usecases.NewRegisterRepositorySecret(client.Actions)
	if err := cli.NewApp(uc).Run(ctx, os.Args); err != nil {
		slog.ErrorContext(ctx, "Run failed", log.AttrError(err))
		return 1
	}
	return 0
}
