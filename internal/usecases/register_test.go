package usecases_test

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/aereal/github-ops/internal/assertions"
	"github.com/aereal/github-ops/internal/usecases"
	"github.com/google/go-github/v69/github"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/nacl/box"
)

func TestRegisterRepositorySecret_Do(t *testing.T) {
	pubKey, err := getPublicKey()
	if err != nil {
		t.Fatal(err)
	}
	type input struct {
		repoOwner  string
		repoName   string
		secretName string
		plainMsg   string
	}
	testCases := []struct {
		wantErr error
		doMock  func(m *MockGHActionsService)
		input   input
		name    string
	}{
		{
			name: "ok",
			input: input{
				repoOwner:  "aereal",
				repoName:   "myrepo",
				secretName: "MY_SECRET",
				plainMsg:   "blah blah",
			},
			doMock: func(m *MockGHActionsService) {
				succeedsCreateOrUpdateRepoSecret(m).
					Times(1).
					After(succeedsGetRepoPublicKey(m, pubKey).Times(1))
			},
		},
		{
			name: "failed to GetRepoPublicKey",
			input: input{
				repoOwner:  "aereal",
				repoName:   "myrepo",
				secretName: "MY_SECRET",
				plainMsg:   "blah blah",
			},
			doMock: func(m *MockGHActionsService) {
				_ = failsGetRepoPublicKey(m)
			},
			wantErr: errGetRepoPublicKey,
		},
		{
			name: "failed to CreateOrUpdateRepoSecret",
			input: input{
				repoOwner:  "aereal",
				repoName:   "myrepo",
				secretName: "MY_SECRET",
				plainMsg:   "blah blah",
			},
			doMock: func(m *MockGHActionsService) {
				_ = failsCreateOrUpdateRepoSecret(m).
					Times(1).
					After(succeedsGetRepoPublicKey(m, pubKey).Times(1))
			},
			wantErr: errCreateOrUpdateRepoSecret,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockClient := NewMockGHActionsService(ctrl)
			if doMock := testCase.doMock; doMock != nil {
				doMock(mockClient)
			}
			ctx := t.Context()
			gotErr := usecases.
				NewRegisterRepositorySecret(mockClient).
				DoRegisterRepositorySecret(ctx, testCase.input.repoOwner, testCase.input.repoName, testCase.input.secretName, testCase.input.plainMsg)
			if diff := assertions.DiffErrorsConservatively(testCase.wantErr, gotErr); diff != "" {
				t.Errorf("error (-want, +got):\n%s", diff)
			}
		})
	}
}

func succeedsGetRepoPublicKey(m *MockGHActionsService, pubKey *github.PublicKey) *MockGHActionsServiceGetRepoPublicKeyCall {
	return m.EXPECT().
		GetRepoPublicKey(gomock.Any(), "aereal", "myrepo").
		Return(pubKey, &github.Response{}, nil)
}

var errGetRepoPublicKey = errors.New("fail: GetRepoPublicKey")

func failsGetRepoPublicKey(m *MockGHActionsService) *MockGHActionsServiceGetRepoPublicKeyCall {
	return m.EXPECT().
		GetRepoPublicKey(gomock.Any(), "aereal", "myrepo").
		Return(nil, &github.Response{}, errGetRepoPublicKey)
}

func succeedsCreateOrUpdateRepoSecret(m *MockGHActionsService) *MockGHActionsServiceCreateOrUpdateRepoSecretCall {
	return m.EXPECT().
		CreateOrUpdateRepoSecret(gomock.Any(), "aereal", "myrepo", &encryptedSecretMatcher{name: "MY_SECRET", keyID: "0xdeadbeaf"}).
		Return(&github.Response{}, nil)
}

var errCreateOrUpdateRepoSecret = errors.New("fail: CreateOrUpdateRepoSecret")

func failsCreateOrUpdateRepoSecret(m *MockGHActionsService) *MockGHActionsServiceCreateOrUpdateRepoSecretCall {
	return m.EXPECT().
		CreateOrUpdateRepoSecret(gomock.Any(), "aereal", "myrepo", &encryptedSecretMatcher{name: "MY_SECRET", keyID: "0xdeadbeaf"}).
		Return(nil, errCreateOrUpdateRepoSecret)
}

var (
	getPublicKey = sync.OnceValues(func() (*github.PublicKey, error) {
		pubKey, _, err := box.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		encodedKey := base64.StdEncoding.EncodeToString((*pubKey)[:])
		return &github.PublicKey{
			KeyID: ref("0xdeadbeaf"),
			Key:   ref(encodedKey),
		}, nil
	})
)

func ref[T any](t T) *T { return &t }

type encryptedSecretMatcher struct {
	name  string
	keyID string
}

var _ gomock.Matcher = (*encryptedSecretMatcher)(nil)

func (m *encryptedSecretMatcher) Matches(x any) bool {
	encryptedSecret, ok := x.(*github.EncryptedSecret)
	if !ok {
		return false
	}
	return encryptedSecret.Name == m.name && encryptedSecret.KeyID == m.keyID
}

func (m *encryptedSecretMatcher) String() string {
	return fmt.Sprintf("&github.EncryptedSecret{Name=%q; KeyID=%q}", m.name, m.keyID)
}
