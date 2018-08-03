package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_convertGithubURL(t *testing.T) {

	tests := []struct {
		name    string
		arg     string
		want    string
		wantErr bool
	}{
		{
			name:    "valid url with git",
			arg:     "git@github.com:USERNAME/REPOSITORY.git",
			want:    "https://github.com/USERNAME/REPOSITORY.git",
			wantErr: false,
		},
		{
			name:    "valid url with https",
			arg:     "https://github.com/USERNAME/REPOSITORY.git",
			want:    "https://github.com/USERNAME/REPOSITORY.git",
			wantErr: false,
		},
		{
			name:    "invalid url without git at the end",
			arg:     "git@github.com:USERNAME/REPOSITORY",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertGithubURL(tt.arg)
			if tt.wantErr {
				require.Error(t, err)
				t.Logf("convertGithubURL() failed with error = %v", err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
