package utils

import (
	"fmt"
	"os"
	"testing"
)

func TestResolveAbsPath(t *testing.T) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		t.Errorf("cant get homedir: %v", err)
	}

	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"Tilda Test", args{path: "~/.bash_history"}, fmt.Sprintf("%s/.bash_history", homedir), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveAbsPath(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveAbsPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveAbsPath() got = %v, want %v", got, tt.want)
			}
		})
	}
}
