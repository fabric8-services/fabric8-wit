package gemini

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/client"
	"github.com/fabric8-services/fabric8-wit/ptr"
)

func Test_keepScanningThisCodebase(t *testing.T) {
	type args struct {
		codebases *client.CodebaseList
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "no codebases at all",
			args: args{
				codebases: &client.CodebaseList{
					Data: []*client.Codebase{},
				},
			},
			want: false,
		},
		{
			name: "one codebase with cve-scan as false",
			args: args{
				codebases: &client.CodebaseList{
					Data: []*client.Codebase{
						{
							Attributes: &client.CodebaseAttributes{
								CveScan: ptr.Bool(false),
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "one codebase with cve-scan as true",
			args: args{
				codebases: &client.CodebaseList{
					Data: []*client.Codebase{
						{
							Attributes: &client.CodebaseAttributes{
								CveScan: ptr.Bool(true),
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "two codebases one with cve-scan as true and other false",
			args: args{
				codebases: &client.CodebaseList{
					Data: []*client.Codebase{
						{
							Attributes: &client.CodebaseAttributes{
								CveScan: ptr.Bool(false),
							},
						},
						{
							Attributes: &client.CodebaseAttributes{
								CveScan: ptr.Bool(true),
							},
						},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := keepScanningThisCodebase(tt.args.codebases); got != tt.want {
				t.Errorf("keepScanningThisCodebase() = %v, want %v", got, tt.want)
			}
		})
	}
}
