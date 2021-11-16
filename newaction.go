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
			errors.New("pull request changes multiple package files")
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

type pkgVerTuple struct {
	name string
	version string
}

func ParsePackageLock(changes []string) []pkgVerTuple {
	cur := 0
	pkgVer := make([]pkgVerTuple,0)

	namePat := regexp.MustCompile(`\+.*"(.*?)": {`)
	versionPat := regexp.MustCompile(`\+.*"version": "(.*?)"`)
	resolvedPat := regexp.MustCompile(`\+.*"resolved": "(.*?)"`)

	for cur < len(changes)-2 {
		nameMatch := namePat.FindAllStringSubmatch(changes[cur],-1)
		if versionPat.MatchString(changes[cur+1]) {
			versionMatch := versionPat.FindAllStringSubmatch(changes[cur+1],-1)
			if resolvedPat.MatchString(changes[cur+2]) {
				if name := nameMatch[0][1]; !strings.Contains(name,"node_modules") {
					pv := pkgVerTuple{nameMatch[0][1], versionMatch[0][1]}
					pkgVer = append(pkgVer, pv)
				}
			}
		}
		cur += 1
	}
	return pkgVer
}

func ParseYarnLock(changes []string) []pkgVerTuple {
	cur := 0
	pkgVer := make([]pkgVerTuple,0)

	namePat := regexp.MustCompile(`\+(.*?)@.*:`)
	versionPat := regexp.MustCompile(`\+.*version "(.*?)"`)
	resolvedPat := regexp.MustCompile(`\+.*resolved "(.*?)"`)
	integrityPat := regexp.MustCompile(`\+.*integrity.*`)


	for cur < len(changes)-3 {
		nameMatch := namePat.FindAllStringSubmatch(changes[cur],-1)
		if versionPat.MatchString(changes[cur+1]) {
			versionMatch := versionPat.FindAllStringSubmatch(changes[cur+1],-1)
			if resolvedPat.MatchString(changes[cur+2]) {
				if integrityPat.MatchString(changes[cur+3]) {
					pkgVer = append(pkgVer, pkgVerTuple{nameMatch[0][1], versionMatch[0][1]})
				}
			}
		}
		cur += 1
	}
	return pkgVer
}

func ParseRequirementsDotTxt(changes []string) []pkgVerTuple {
	nameVerPat := regexp.MustCompile(`\+(.*?)==(.*)`)
	pkgVer := make([]pkgVerTuple,0)
	for _,line := range changes {
		if strings.Contains(line,"\n") {
			continue
		}
		if nameVerPat.MatchString(line) {
			nameVerMatch := nameVerPat.FindAllStringSubmatch(line,-1)
			pkgVer = append(pkgVer, pkgVerTuple{nameVerMatch[0][1],nameVerMatch[0][2]})
		}
	}
	return pkgVer
}

