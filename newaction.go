package main

import (
	"encoding/json"
	"fmt"
	"github.com/sourcegraph/go-diff/diff"
	"github.com/xanzy/go-gitlab"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var RETURNCODE_FILENAME = "./returncode.txt"

const TOKEN_NAME = "PAT"

func GetPRDiff() (body []byte, err error) {
	ci_commit_sha := os.Getenv("CI_COMMIT_SHA")
	_ = ci_commit_sha
	ci_mr_target_branch := os.Getenv("CI_MERGE_REQUEST_TARGET_BRANCH_NAME")
	lastArg := fmt.Sprintf("origin/%s", ci_mr_target_branch)
	diffCmd := exec.Command("git", "diff", lastArg, ci_commit_sha)
	//diffCmd := exec.Command("git","diff", ci_mr_target_branch)
	//fmt.Println("args", diffCmd.Args)
	diffOut, err := diffCmd.Output()
	if err != nil {
		fmt.Println("diffCmd failed")
		panic(err)
	}
	body = diffOut

	return body, nil
}

func DeterminePatchType(diffData []byte) (prType string, lang string, err error) {

	d, err := diff.ParseMultiFileDiff(diffData)
	if err != nil {
		//fmt.Println("[❌] DeterminePatchType couldn't parse diff output")
		err = fmt.Errorf("[❌] DeterminePatchType couldn't parse diff output")
		return "", "", err
	}
	for _, a_diff := range d {
		diffFileName := strings.TrimPrefix(a_diff.NewName, "b/")
		if prType != "" && diffFileName != prType {
			panic("pull request changes multiple package files")
		}
		switch diffFileName {
		case "requirements.txt":
			prType = "requirements.txt"
			lang = "py"
		case "package-lock.json":
			prType = "package-lock.json"
			lang = "js"
		case "yarn.lock":
			prType = "yarn.lock"
			lang = "js"
		case "Gemfile.lock":
			prType = "Gemfile.lock"
			lang = "rb"
		}
	}
	return prType, lang, err
}

func GetChanges(diffData []byte) *[]string {
	changes := make([]string, 0)
	d, err := diff.ParseMultiFileDiff(diffData)
	if err != nil {
		fmt.Println("[❌] GetChanges: couldn't parse diff output")
		panic(err)
	}
	for _, a_diff := range d {
		for _, hunk := range a_diff.Hunks {
			initial := string(hunk.Body)
			if strings.Contains(initial, "\n") {
				strs := strings.Split(initial, "\n")
				for _, str := range strs {
					if strings.HasPrefix(str, "+") && len(str) > 1 {
						changes = append(changes, str)
					}
				}
			}
		}
	}
	return &changes
}

func ParsePackageLock(changes *[]string) *[]pkgVerTuple {
	cur := 0
	pkgVer := make([]pkgVerTuple, 0)

	namePat := regexp.MustCompile(`\+.*?"(.*?)": {`)
	versionPat := regexp.MustCompile(`\+.*"version": "(.*?)"`)
	resolvedPat := regexp.MustCompile(`\+.*"resolved": "(.*?)"`)

	for cur < len(*changes)-2 {
		nameMatch := namePat.FindAllStringSubmatch((*changes)[cur], -1)
		if versionPat.MatchString((*changes)[cur+1]) {
			versionMatch := versionPat.FindAllStringSubmatch((*changes)[cur+1], -1)
			if resolvedPat.MatchString((*changes)[cur+2]) {
				if name := nameMatch[0][1]; !strings.Contains(name, "node_modules") {
					pv := pkgVerTuple{nameMatch[0][1], versionMatch[0][1]}
					pkgVer = append(pkgVer, pv)
				}
			}
		}
		cur += 1
	}
	return &pkgVer
}

func ParseYarnLock(changes *[]string) *[]pkgVerTuple {
	cur := 0
	pkgVer := make([]pkgVerTuple, 0)

	namePat := regexp.MustCompile(`\+(.*?)@.*:`)
	versionPat := regexp.MustCompile(`\+.*version "(.*?)"`)
	resolvedPat := regexp.MustCompile(`\+.*resolved "(.*?)"`)
	integrityPat := regexp.MustCompile(`\+.*integrity.*`)

	for cur < len(*changes)-3 {
		nameMatch := namePat.FindAllStringSubmatch((*changes)[cur], -1)
		if versionPat.MatchString((*changes)[cur+1]) {
			versionMatch := versionPat.FindAllStringSubmatch((*changes)[cur+1], -1)
			if resolvedPat.MatchString((*changes)[cur+2]) {
				if integrityPat.MatchString((*changes)[cur+3]) {
					pkgVer = append(pkgVer, pkgVerTuple{nameMatch[0][1], versionMatch[0][1]})
				}
			}
		}
		cur += 1
	}
	return &pkgVer
}

func ParseRequirementsDotTxt(changes *[]string) *[]pkgVerTuple {
	nameVerPat := regexp.MustCompile(`\+(.*?)==(.*)`)
	pkgVer := make([]pkgVerTuple, 0)
	for _, line := range *changes {
		if strings.Contains(line, "\n") {
			continue
		}
		if nameVerPat.MatchString(line) {
			nameVerMatch := nameVerPat.FindAllStringSubmatch(line, -1)
			pkgVer = append(pkgVer, pkgVerTuple{nameVerMatch[0][1], nameVerMatch[0][2]})
		}
	}
	return &pkgVer
}

func ParseGemfileLock(changes *[]string) *[]pkgVerTuple {
	nameVerPat := regexp.MustCompile(`\s{4}(.*?)\ \((.*?)\)`)
	pkgVer := make([]pkgVerTuple, 0)
	for _, line := range *changes {
		if nameVerPat.MatchString(line) {
			nameVerMatch := nameVerPat.FindAllStringSubmatch(line, -1)
			pkgVer = append(pkgVer, pkgVerTuple{nameVerMatch[0][1], nameVerMatch[0][2]})
		}
	}
	return &pkgVer
}

func GetChangedPackages(changes *[]string, prType string) *[]pkgVerTuple {
	var pkgVer *[]pkgVerTuple
	switch prType {
	case "package-lock.json":
		pkgVer = ParsePackageLock(changes)
	case "yarn.lock":
		pkgVer = ParseYarnLock(changes)
	case "requirements.txt":
		pkgVer = ParseRequirementsDotTxt(changes)
	case "Gemfile.lock":
		pkgVer = ParseGemfileLock(changes)
	}
	return pkgVer
}

func ReadPhylumAnalysis(fileName string) PhylumJson {
	homePath := os.Getenv("HOME")
	filePath := filepath.Join(homePath, fileName)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println("[❌] ReadPhylumAnalysis couldn't read file: ", fileName)
		panic(err)
	}
	var phylumJson PhylumJson
	if err := json.Unmarshal(data, &phylumJson); err != nil {
		fmt.Println("[❌] ReadPhylumAnalysis couldn't unmarshal JSON")
		panic(err)
	}
	return phylumJson
}

// ParsePhylumRiskData Join the changed packages with risk data from Phylum
func ParsePhylumRiskData(pkgVer *[]pkgVerTuple, phylumJson PhylumJson, ut UserThresholds) (retdata string, err error) {
	results := make([]string, 0)
	incompletes := make([]pkgVerTuple, 0)

	for _, pv := range *pkgVer {
		for _, pkg := range phylumJson.Packages {
			if pv.name == pkg.Name && pv.version == pkg.Version {
				switch pkg.Status {
				case "complete":
					results = append(results, CheckRiskScores(pkg, ut))
				case "incomplete":
					incompletes = append(incompletes, pkgVerTuple{pkg.Name, pkg.Version})
				}
			}
		}
	}
	if len(incompletes) > 0 {
		return "", fmt.Errorf("[❌ ERROR] Phylum status for %d packages was incomplete\n")
	}
	return strings.Join(results, ""), nil
}

func ToI(input float64) int {
	return int(input * 100)
}

func GenerateIssueRow(pkg Package, riskDomain string) string {
	var singleIssue strings.Builder
	if riskDomain == "vul" {
		for _, vuln := range pkg.Vulnerabilities {
			fmt.Fprintf(&singleIssue, "|%s|%s|%s\n", "Vulnerability", vuln.RiskLevel, vuln.Title)
		}
	} else {
		for _, issue := range pkg.Issues {
			riskDomain := strings.Replace(issue.RiskDomain, "_", " ", 1)
			fmt.Fprintf(&singleIssue, "|%s|%s|%s\n", riskDomain, issue.RiskLevel, issue.Title)
		}
	}
	return singleIssue.String()
}

func CheckRiskScores(pkg Package, ut UserThresholds) string {
	var headerString, failString, issueString strings.Builder
	issueMap := make(map[string]string, 0)
	rv := pkg.RiskVectors
	fmt.Fprintf(&headerString, "### Package: `%s@%s` failed\n", pkg.Name, pkg.Version)
	fmt.Fprintf(&headerString, "|Risk Domain|Identified Score|Requirement|\n")
	fmt.Fprintf(&headerString, "|-----------|----------------|-----------|\n")
	if rv.Vulnerability <= ut.Vul {
		fmt.Fprintf(&failString, "|Software Vulnerability|%d|%d|\n", ToI(rv.Vulnerability), ToI(ut.Vul))
		issueMap["vul"] = GenerateIssueRow(pkg, "vul")
	}
	if rv.MaliciousCode <= ut.Mal {
		fmt.Fprintf(&failString, "|Malicious Code|%d|%d|\n", ToI(rv.MaliciousCode), ToI(ut.Mal))
		issueMap["mal"] = GenerateIssueRow(pkg, "mal")
	}
	if rv.License <= ut.Lic {
		fmt.Fprintf(&failString, "|License|%d|%d|\n", ToI(rv.License), ToI(ut.Lic))
		issueMap["lic"] = GenerateIssueRow(pkg, "lic")
	}
	if rv.Engineering <= ut.Lic {
		fmt.Fprintf(&failString, "|Engineering|%d|%d|\n", ToI(rv.Engineering), ToI(ut.Eng))
		issueMap["eng"] = GenerateIssueRow(pkg, "eng")
	}
	if rv.Author <= ut.Aut {
		fmt.Fprintf(&failString, "|Author|%d|%d|\n", ToI(rv.Author), ToI(ut.Aut))
		issueMap["aut"] = GenerateIssueRow(pkg, "aut")
	}

	fmt.Fprintf(&issueString, "#### Issues Summary\n")
	fmt.Fprintf(&issueString, "|Risk Domain|Risk Level|Title|\n")
	fmt.Fprintf(&issueString, "|-----------|----------|-----|\n")

	for _, v := range issueMap {
		fmt.Fprintf(&issueString, v)
	}

	if failString.Len() > 0 {
		return headerString.String() + failString.String() + issueString.String()
	}
	return ""
}

func PRType() string {
	diffText, err := GetPRDiff()
	if err != nil {
		panic(err)
	}
	prType, _, err := DeterminePatchType(diffText)
	if err != nil {
		panic(err)
	}
	return prType
}

func Analyze(repo string, mrNum int, ut UserThresholds) {
	var returnCode int
	diffText, err := GetPRDiff()
	if err != nil {
		panic(err)
	}
	prType, lang, err := DeterminePatchType(diffText)
	_ = lang
	if err != nil {
		panic(err)
	}
	changes := GetChanges(diffText)
	pkgVer := GetChangedPackages(changes, prType)
	phylumJsonData := ReadPhylumAnalysis("phylum_analysis.json")
	phylumRiskData, err := ParsePhylumRiskData(pkgVer, phylumJsonData, ut)
	//TODO: can likely return just the exit value now - investigate
	if err != nil {
		returnCode = 5 //incomplete packages
		fmt.Printf("[*] Phylum analysis for %s MR#%d INCOMPLETE\n", repo, mrNum)
	}

	if len(phylumRiskData) > 0 {
		returnCode = 1

		//TODO: could use the go:embed directive here
		const header = `### Phylum OSS Supply Chain Risk Analysis
<details>
<summary>Background</summary>

<br />

This repository uses a GitHub Action to automatically analyze the risk of new dependencies added to requirements.txt via Pull Request. An administrator of this repository has set score requirements for Phylum's five risk domains.<br /><br />

If you see this comment, one or more dependencies added to the requirements.txt file in this Pull Request have failed Phylum's risk analysis.

</details>`
		projectFooter := fmt.Sprintf("\n[View this project in Phylum UI](https://app.phylum.io/projects/%s)", phylumJsonData.Project)
		finalStr := header + "\n\n" + phylumRiskData + projectFooter
		if err := CreateMRComment(repo, mrNum, finalStr); err != nil {
			panic(err)
		}

		fmt.Printf("[*] Phylum analysis for %s MR#%d FAILED\n", repo, mrNum)

	} else {
		returnCode = 0
		fmt.Printf("[*] Phylum analysis for %s MR#%d PASSED\n", repo, mrNum)
	}

	f2, err := os.Create(RETURNCODE_FILENAME)
	defer f2.Close()
	if err != nil {
		panic(fmt.Errorf("couldn't open %s for write()", RETURNCODE_FILENAME))
	}
	n2, err := f2.WriteString(strconv.Itoa(returnCode))
	if err != nil {
		panic(fmt.Errorf("couldn't write to %s ", RETURNCODE_FILENAME))
	}
	fmt.Printf("[*] wrote %d bytes to returncode.txt\n", n2)
}

func CreateMRComment(projectPath string, mrNum int, comment string) (err error) {
	apiKey := os.Getenv(TOKEN_NAME)
	if apiKey == "" {
		panic("could not read apiKey from os.environment")
	}

	git, err := gitlab.NewClient(apiKey)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
		return err

	}

	// Get a single project
	//projPath := "peterjmorgan/phylum-analyze-pr-gitlab-test"
	project, resp, err := git.Projects.GetProject(projectPath, &gitlab.GetProjectOptions{}, nil)
	if err != nil {
		log.Fatalf("Failed to get project %v", err)
		return err
	}
	_ = resp

	// Get a merge request
	mr, resp, err := git.MergeRequests.GetMergeRequest(project.ID, mrNum, &gitlab.GetMergeRequestsOptions{})
	if err != nil {
		log.Fatalf("Failed to get MR #%d - %v", mrNum, err)
		return err
	}
	_ = mr

	// Get diff versions - not really sure what the disambiguation is here
	mrDiffVersions, resp, err := git.MergeRequests.GetMergeRequestDiffVersions(
		project.ID,
		mrNum,
		&gitlab.GetMergeRequestDiffVersionsOptions{})
	if err != nil {
		log.Fatalf("Failed to get MR #%d Diff Versions - %v", mrNum, err)
		return err
	}
	_ = mrDiffVersions

	// Get the latest diff version - I think this makes sense. I should operate off of the most recent diff, i think?
	lastMrDiffVersion := mrDiffVersions[len(mrDiffVersions)-1]

	// The single merge request DIff version
	mrDiff, resp, err := git.MergeRequests.GetSingleMergeRequestDiffVersion(
		project.ID,
		mrNum,
		lastMrDiffVersion.ID)
	if err != nil {
		log.Fatalf("Failed to get MR #%d diff id:%d - %v", mrNum, lastMrDiffVersion.ID, err)
		return err
	}
	_ = mrDiff

	_, resp, err = git.Notes.CreateMergeRequestNote(
		project.ID,
		mrNum,
		&gitlab.CreateMergeRequestNoteOptions{
			Body: &comment,
		},
		nil)
	if err != nil {
		log.Fatalf("Failed to create note on MR#%d - %v", mrNum, err)
		return err
	}
	return nil
}
