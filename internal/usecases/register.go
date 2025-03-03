package usecases

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/go-github/v69/github"
	"golang.org/x/crypto/nacl/box"
)

type GHActionsService interface {
	GetRepoPublicKey(ctx context.Context, owner, repo string) (*github.PublicKey, *github.Response, error)
	CreateOrUpdateRepoSecret(ctx context.Context, owner, repo string, eSecret *github.EncryptedSecret) (*github.Response, error)
}

func NewRegisterRepositorySecret(client GHActionsService) *RegisterRepositorySecret {
	return &RegisterRepositorySecret{client: client}
}

type RegisterRepositorySecret struct {
	client GHActionsService
}

func (u *RegisterRepositorySecret) DoRegisterRepositorySecret(ctx context.Context, repoOwner string, repoName string, secretName string, plainMsg string) error {
	pubKey, _, err := u.client.GetRepoPublicKey(ctx, repoOwner, repoName)
	if err != nil {
		return fmt.Errorf("GetRepoPublicKey: %w", err)
	}
	serverPubKey, err := getRawPublicKey(pubKey)
	if err != nil {
		return err
	}
	encrypted, err := encryptAndEncode([]byte(plainMsg), serverPubKey)
	if err != nil {
		return err
	}
	secret := &github.EncryptedSecret{
		Name:           secretName,
		KeyID:          pubKey.GetKeyID(),
		EncryptedValue: encrypted,
	}
	slog.InfoContext(ctx, "set repository secret",
		slog.String("repo.owner", repoOwner),
		slog.String("repo.name", repoName),
		slog.String("secret.name", secretName),
	)
	if _, err := u.client.CreateOrUpdateRepoSecret(ctx, repoOwner, repoName, secret); err != nil {
		return fmt.Errorf("CreateOrUpdateRepoSecret: %w", err)
	}
	return nil
}

func encryptAndEncode(msg []byte, pubKey *[32]byte) (string, error) {
	var out []byte
	got, err := box.SealAnonymous(out, msg, pubKey, rand.Reader)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(got), nil
}

func getRawPublicKey(pubKey *github.PublicKey) (*[32]byte, error) {
	rawPubKey, err := base64.StdEncoding.DecodeString(pubKey.GetKey())
	if err != nil {
		return nil, fmt.Errorf("base64.Encoding.DecodeString: %w", err)
	}
	serverPubKey := new([32]byte)
	if _, err := io.ReadFull(bytes.NewReader(rawPubKey), serverPubKey[:]); err != nil {
		return nil, fmt.Errorf("io.ReadFull: %w", err)
	}
	return serverPubKey, nil
}
