//go:generate go tool mockgen -destination ./usecase_mock_test.go -package cli_test -typed -write_command_comment=false github.com/aereal/register-github-secret/internal/cli RegisterRepositorySecretUsecase

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	set "github.com/hashicorp/go-set/v3"
	"golang.org/x/sync/errgroup"
)

type RegisterRepositorySecretUsecase interface {
	DoRegisterRepositorySecret(ctx context.Context, repoOwner string, repoName string, secretName string, plainMsg string) error
}

func NewApp(uc RegisterRepositorySecretUsecase) *App {
	return &App{uc: uc}
}

type App struct {
	uc RegisterRepositorySecretUsecase
}

func (a *App) Run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet(filepath.Base(args[0]), flag.ContinueOnError)
	var (
		secretName  string
		secretValue string
		repos       = set.New[qualifiedRepo](0)
	)
	fs.Func("repos", "repository name list", func(s string) error {
		qr := new(qualifiedRepo)
		if err := qr.Set(s); err != nil {
			return err
		}
		_ = repos.Insert(*qr)
		return nil
	})
	fs.StringVar(&secretName, "secret-name", "", "secret name")
	fs.StringVar(&secretValue, "secret-value", "", "secret value")
	err := fs.Parse(args[1:])
	switch {
	case errors.Is(err, flag.ErrHelp):
		return nil
	case err != nil:
		return err
	}
	if secretName == "" {
		return ErrSecretNameRequired
	}
	if secretValue == "" {
		return ErrSecretValueRequired
	}
	eg, ctx := errgroup.WithContext(ctx)
	for r := range repos.Items() {
		owner := r.Owner
		repoName := r.Repo
		eg.Go(func() error {
			return a.uc.DoRegisterRepositorySecret(ctx, owner, repoName, secretName, secretValue)
		})
	}
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("usecases.NewRegisterRepositorySecret.Do: %w", err)
	}
	return nil
}

type qualifiedRepo struct {
	Owner, Repo string
}

var _ flag.Value = (*qualifiedRepo)(nil)

func (r qualifiedRepo) String() string { return r.Owner + "/" + r.Repo }

func (r *qualifiedRepo) Set(v string) error {
	owner, repo, ok := strings.Cut(v, "/")
	if !ok {
		return &MalformedQualifiedRepoError{v}
	}
	*r = qualifiedRepo{Owner: owner, Repo: repo}
	return nil
}
