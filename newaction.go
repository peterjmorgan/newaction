package main

import (
	"errors"
	"fmt"
	"github.com/sourcegraph/go-diff/diff"
	"io"
	"net/http"
	"regexp"
	"strings"
)

func GetPRDiff(repo string, prNum int) []byte {
	//if strings.ContainsRune(repo,'-') {
	//	repo = strings.ReplaceAll(repo,"-","_")
	//}

	url := fmt.Sprintf("https://patch-diff.githubusercontent.com/raw/%s/pull/%d.diff", repo, prNum)
	fmt.Println(url)

	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return body
}

func DeterminePatchType(diffData []byte) string {
	var prType string

	d, err := diff.ParseMultiFileDiff(diffData)
	if err != nil {
		fmt.Println("[❌] DeterminePatchType couldn't parse diff output")
		panic(err)
	}
	for _, diff := range d {
		diffFileName := strings.TrimPrefix(diff.NewName, "b/")
		if prType != "" && diffFileName != prType {
			errors.New("Pull request changes multiple package files")
		}
		switch diffFileName {
		case "requirements.txt":
			prType = "requirements.txt"
		case "package-lock.json":
			prType = "package-lock.json"
		case "yarn.lock":
			prType = "yarn.lock"
		case "Gemfile.lock":
			prType = "Gemfile.lock"
		}
	}
	return prType
}

func GetChanges(diffData []byte) []string {
	changes := make([]string,0)
	d, err := diff.ParseMultiFileDiff(diffData)
	if err != nil {
		fmt.Println("[❌] GetChanges: couldn't parse diff output")
		panic(err)
	}
	for _, diff := range d {
		for _, hunk := range diff.Hunks {
			initial := string(hunk.Body)
			if strings.Contains(initial,"\n") {
				strs := strings.Split(initial,"\n")
				for _, str := range strs {
					if strings.HasPrefix(str,"+") && len(str) > 1 {
						changes = append(changes, str)
					}
				}
			}
		}
	}
	return changes
}

func ParsePackageLock(changes []string) []map[string]int {
	cur := 0
	pkgVer := make(map[string]int,0)
	name_pat := regexp.Compile(".*\"(.*?)\": \{")
	version_pat := regexp.Compile(".*\"version\": \"(.*?)\"")
	resolved_pat := regexp.Compile(".*\"resolved\": \"(.*?)\"")

	for cur < len(changes)-2 {

		cur += 1
	}



}
