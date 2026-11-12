package cmd

import "testing"

func TestRepositoryFromRemote(t *testing.T) {
	tests := []struct {
		name    string
		remote  string
		host    string
		want    string
		wantErr bool
	}{
		{
			name:   "HTTPS",
			remote: "https://codeberg.org/matcxz/codeberg-cli.git",
			host:   "https://codeberg.org",
			want:   "matcxz/codeberg-cli",
		},
		{
			name:   "SSH URL",
			remote: "ssh://git@codeberg.org/matcxz/codeberg-cli.git",
			host:   "https://codeberg.org",
			want:   "matcxz/codeberg-cli",
		},
		{
			name:   "SCP-style SSH",
			remote: "git@codeberg.org:matcxz/codeberg-cli.git",
			host:   "https://codeberg.org",
			want:   "matcxz/codeberg-cli",
		},
		{
			name:    "different host",
			remote:  "https://github.com/matcxz/codeberg-cli.git",
			host:    "https://codeberg.org",
			wantErr: true,
		},
		{
			name:    "invalid repository path",
			remote:  "https://codeberg.org/matcxz",
			host:    "https://codeberg.org",
			wantErr: true,
		},
		{
			name:   "instance hosted under a path",
			remote: "https://forge.example.org/forgejo/team/tool.git",
			host:   "https://forge.example.org/forgejo",
			want:   "team/tool",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := repositoryFromRemote(test.remote, test.host)
			if (err != nil) != test.wantErr {
				t.Fatalf("repositoryFromRemote() error = %v, wantErr %v", err, test.wantErr)
			}
			if err == nil && got != test.want {
				t.Fatalf("repositoryFromRemote() = %q, want %q", got, test.want)
			}
		})
	}
}
