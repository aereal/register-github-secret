package usecases_test

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/aereal/register-github-secret/internal/usecases"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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
		name    string
		input   input
		doMock  func(m *MockGHActionsService)
		wantErr error
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
				Do(ctx, testCase.input.repoOwner, testCase.input.repoName, testCase.input.secretName, testCase.input.plainMsg)
			if diff := diffErrorsConservatively(testCase.wantErr, gotErr); diff != "" {
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

type literalError struct {
	msg string
}

func (e *literalError) Error() string { return e.msg }

func (e *literalError) Is(other error) bool {
	if e == nil {
		return other == nil
	}
	if other == nil {
		return false
	}
	return e.msg == other.Error()
}

func diffErrorsConservatively(want, got error) string {
	return cmp.Diff(want, got, cmpopts.EquateErrors())
}

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
