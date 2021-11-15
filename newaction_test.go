package newaction

import "testing"

func Test_getPRDiff(t *testing.T) {
	type args struct {
		repo string
		prNum int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{"Test 1", args{"peterjmorgan/analyze-pr-action-test", 5}, "diff --git a/requirements.txt b/requirements.txt\nindex 53b13d5..45e36bc 100644\n--- a/requirements.txt\n+++ b/requirements.txt\n@@ -1 +1,5 @@\n requests==2.6.1\n+pillow==5.3.0\n+cffi==1.14.6\n+cherrypy==8.9.1\n+\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPRDiff(tt.args.repo, tt.args.prNum); got != tt.want {
				t.Errorf("getPRDiff() = %v, want %v", got, tt.want)
			}
		})
	}
}
