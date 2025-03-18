package cli_test

import (
	"testing"

	"github.com/aereal/github-ops/internal/assertions"
	"github.com/aereal/github-ops/internal/cli"
	"go.uber.org/mock/gomock"
)

func TestApp_Run(t *testing.T) {
	testCases := []struct {
		wantErr error
		doMock  func(m *MockRegisterRepositorySecretUsecase)
		name    string
		args    []string
	}{
		{
			name: "some repos specified",
			args: []string{"app", "-secret-name", "MY_SECRET", "-secret-value", "blah blah", "-repos", "aereal/repo1", "-repos", "aereal/repo2"},
			doMock: func(m *MockRegisterRepositorySecretUsecase) {
				m.EXPECT().DoRegisterRepositorySecret(gomock.Any(), "aereal", "repo1", "MY_SECRET", "blah blah").Return(nil).Times(1)
				m.EXPECT().DoRegisterRepositorySecret(gomock.Any(), "aereal", "repo2", "MY_SECRET", "blah blah").Return(nil).Times(1)
			},
			wantErr: nil,
		},
		{
			name: "failed to register",
			args: []string{"app", "-secret-name", "MY_SECRET", "-secret-value", "blah blah", "-repos", "aereal/repo1", "-repos", "aereal/repo2"},
			doMock: func(m *MockRegisterRepositorySecretUsecase) {
				m.EXPECT().DoRegisterRepositorySecret(gomock.Any(), "aereal", "repo1", "MY_SECRET", "blah blah").Return(nil).Times(1)
				m.EXPECT().DoRegisterRepositorySecret(gomock.Any(), "aereal", "repo2", "MY_SECRET", "blah blah").Return(errFailed).Times(1)
			},
			wantErr: errFailed,
		},
		{
			name: "same repos repeated",
			args: []string{"app", "-secret-name", "MY_SECRET", "-secret-value", "blah blah", "-repos", "aereal/repo1", "-repos", "aereal/repo1"},
			doMock: func(m *MockRegisterRepositorySecretUsecase) {
				m.EXPECT().DoRegisterRepositorySecret(gomock.Any(), "aereal", "repo1", "MY_SECRET", "blah blah").Return(nil).Times(1)
			},
			wantErr: nil,
		},
		{
			name:    "help wanted",
			args:    []string{"app", "-help"},
			wantErr: nil,
		},
		{
			name:    "invalid repo",
			args:    []string{"app", "-secret-name", "MY_SECRET", "-secret-value", "blah blah", "-repos", "repo1"},
			wantErr: assertions.LiteralError(`invalid value "repo1" for flag -repos: malformed qualified repository name: "repo1"`),
		},
		{
			name:    "no repos specified",
			args:    []string{"app", "-secret-name", "MY_SECRET", "-secret-value", "blah blah"},
			wantErr: nil,
		},
		{
			name:    "no secret name",
			args:    []string{"app"},
			wantErr: cli.ErrSecretNameRequired,
		},
		{
			name:    "no secret value",
			args:    []string{"app", "-secret-name", "MY_SECRET"},
			wantErr: cli.ErrSecretValueRequired,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockUsecase := NewMockRegisterRepositorySecretUsecase(ctrl)
			if tc.doMock != nil {
				tc.doMock(mockUsecase)
			}
			app := cli.NewApp(mockUsecase)
			ctx := t.Context()
			gotErr := app.Run(ctx, tc.args)
			if diff := assertions.DiffErrorsConservatively(tc.wantErr, gotErr); diff != "" {
				t.Errorf("error (-want, +got):\n%s", diff)
			}
		})
	}
}

var errFailed = assertions.LiteralError("failure")
