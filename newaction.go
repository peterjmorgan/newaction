package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sourcegraph/go-diff/diff"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var gbl_didFail = false
var PR_COMMENT_FILENAME = "./pr_comment.txt"
var RETURNCODE_FILENAME = "./returncode.txt"

func GetPRDiff(repo string, prNum int, provider int) (body *[]byte, err error) {
	var url string

	if provider == 1 {
		url = fmt.Sprintf("https://gitlab.com/%s/-/merge_requests/%d.diff", repo, prNum)
	} else if provider == 0 {
		url = fmt.Sprintf("https://patch-diff.githubusercontent.com/raw/%s/pull/%d.diff", repo, prNum)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.New("couldn't GET the diff from GitHub")
	}
	defer resp.Body.Close()

	body = &[]byte{}
	*body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("couldn't read data from body")
	}
	return body,nil
}

func DeterminePatchType(diffData *[]byte) (prType string, lang string, err error) {

	d, err := diff.ParseMultiFileDiff(*diffData)
	if err != nil {
		//fmt.Println("[❌] DeterminePatchType couldn't parse diff output")
		err = fmt.Errorf("[❌] DeterminePatchType couldn't parse diff output")
		return "", "", err
	}
	for _, diff := range d {
		diffFileName := strings.TrimPrefix(diff.NewName, "b/")
		if prType != "" && diffFileName != prType {
			errors.New("pull request changes multiple package files")
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
	return prType,lang,err
}

func GetChanges(diffData *[]byte) *[]string {
	changes := make([]string,0)
	d, err := diff.ParseMultiFileDiff(*diffData)
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
	return &changes
}

type pkgVerTuple struct {
	name string
	version string
}

func ParsePackageLock(changes *[]string) *[]pkgVerTuple {
	cur := 0
	pkgVer := make([]pkgVerTuple,0)

	namePat := regexp.MustCompile(`\+.*"(.*?)": {`)
	versionPat := regexp.MustCompile(`\+.*"version": "(.*?)"`)
	resolvedPat := regexp.MustCompile(`\+.*"resolved": "(.*?)"`)

	for cur < len(*changes)-2 {
		nameMatch := namePat.FindAllStringSubmatch((*changes)[cur],-1)
		if versionPat.MatchString((*changes)[cur+1]) {
			versionMatch := versionPat.FindAllStringSubmatch((*changes)[cur+1],-1)
			if resolvedPat.MatchString((*changes)[cur+2]) {
				if name := nameMatch[0][1]; !strings.Contains(name,"node_modules") {
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
	pkgVer := make([]pkgVerTuple,0)

	namePat := regexp.MustCompile(`\+(.*?)@.*:`)
	versionPat := regexp.MustCompile(`\+.*version "(.*?)"`)
	resolvedPat := regexp.MustCompile(`\+.*resolved "(.*?)"`)
	integrityPat := regexp.MustCompile(`\+.*integrity.*`)

	for cur < len(*changes)-3 {
		nameMatch := namePat.FindAllStringSubmatch((*changes)[cur],-1)
		if versionPat.MatchString((*changes)[cur+1]) {
			versionMatch := versionPat.FindAllStringSubmatch((*changes)[cur+1],-1)
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
	pkgVer := make([]pkgVerTuple,0)
	for _,line := range *changes {
		if strings.Contains(line,"\n") {
			continue
		}
		if nameVerPat.MatchString(line) {
			nameVerMatch := nameVerPat.FindAllStringSubmatch(line,-1)
			pkgVer = append(pkgVer, pkgVerTuple{nameVerMatch[0][1],nameVerMatch[0][2]})
		}
	}
	return &pkgVer
}

func ParseGemfileLock(changes *[]string) *[]pkgVerTuple {
	nameVerPat := regexp.MustCompile(`\s{4}(.*?)\ \((.*?)\)`)
	pkgVer := make([]pkgVerTuple,0)
	for _,line := range *changes {
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

type Package struct {
	Name               string  `json:"name"`
	Version            string  `json:"version"`
	Status             string  `json:"status"`
	LastUpdated        int64   `json:"last_updated"`
	License            string  `json:"license"`
	PackageScore       float64 `json:"package_score"`
	NumDependencies    int     `json:"num_dependencies"`
	NumVulnerabilities int     `json:"num_vulnerabilities"`
	Type               string  `json:"type"`
	RiskVectors        struct {
		Engineering   float64 `json:"engineering"`
		Vulnerability float64 `json:"vulnerability"`
		Author        float64 `json:"author"`
		MaliciousCode float64 `json:"malicious_code"`
		License       float64 `json:"license"`
	} `json:"riskVectors"`
	Dependencies interface{}  `json:"dependencies"`
	Vulnerabilities []struct {
		Cve         []string `json:"cve"`
		Severity    float64  `json:"severity"`
		RiskLevel   string   `json:"risk_level"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Remediation string   `json:"remediation"`
	} `json:"vulnerabilities"`
	Issues       []struct {
		Title	string `json:"title"`
		Description string `json:"description"`
		RiskLevel string `json:"risk_level"`
		RiskDomain string `json:"risk_domain"`
	}`json:"issues"`
}

type PhylumJson struct {
	JobID         string  `json:"job_id"`
	Ecosystem     string  `json:"ecosystem"`
	UserID        string  `json:"user_id"`
	UserEmail     string  `json:"user_email"`
	CreatedAt     int64   `json:"created_at"`
	Status        string  `json:"status"`
	Score         float64 `json:"score"`
	Pass          bool    `json:"pass"`
	Msg           string  `json:"msg"`
	Action        string  `json:"action"`
	NumIncomplete int     `json:"num_incomplete"`
	LastUpdated   int64   `json:"last_updated"`
	Project       string  `json:"project"`
	ProjectName   string  `json:"project_name"`
	Label         string  `json:"label"`
	Thresholds    struct {
		Author        float64 `json:"author"`
		Engineering   float64 `json:"engineering"`
		License       float64 `json:"license"`
		Malicious     float64 `json:"malicious"`
		Total         float64 `json:"total"`
		Vulnerability float64 `json:"vulnerability"`
	} `json:"thresholds"`
	Packages []Package `json:"packages"`
}

func ReadPhylumAnalysis(filePath string) PhylumJson {
	data,err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println("[❌] ReadPhylumAnalysis couldn't read file: ", filePath)
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
func ParsePhylumRiskData(pkgVer *[]pkgVerTuple, phylumJson PhylumJson, ut UserThresholds) (retdata string,err error) {
	results := make([]string,0)
	incompletes := make([]pkgVerTuple,0)

	for _, pv := range *pkgVer {
		for _, pkg := range phylumJson.Packages {
			if pv.name == pkg.Name && pv.version == pkg.Version {
				switch pkg.Status {
				case "complete":
					results = append(results, CheckRiskScores(pkg, ut))
				case "incomplete":
					incompletes = append(incompletes,pkgVerTuple{pkg.Name, pkg.Version})
				}
			}
		}
	}
	if len(incompletes) > 0 {
		return "", fmt.Errorf("[❌ ERROR] Phylum status for %d packages was incomplete\n")
	}
	return strings.Join(results,""), nil
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
			riskDomain := strings.Replace(issue.RiskDomain, "_", " ",1)
			fmt.Fprintf(&singleIssue,"|%s|%s|%s\n", riskDomain,issue.RiskLevel, issue.Title)
		}
	}
	return singleIssue.String()
}

func CheckRiskScores(pkg Package, ut UserThresholds) string {
	var headerString, failString, issueString strings.Builder
	issueMap := make(map[string]string,0)
	rv := pkg.RiskVectors
	fmt.Fprintf(&headerString, "### Package: `%s@%s` failed\n", pkg.Name, pkg.Version)
	fmt.Fprintf(&headerString, "|Risk Domain|Identified Score|Requirement|\n")
	fmt.Fprintf(&headerString, "|-----------|----------------|-----------|\n")
	if rv.Vulnerability <= ut.Vul {
		fmt.Fprintf(&failString, "|Software Vulnerability|%d|%d|\n", ToI(rv.Vulnerability * 100), ToI(ut.Vul * 100))
		issueMap["vul"] = GenerateIssueRow(pkg,"vul")
	}
	if rv.MaliciousCode <= ut.Mal {
		fmt.Fprintf(&failString, "|Malicious Code|%d|%d|\n", ToI(rv.MaliciousCode), ToI(ut.Mal))
		issueMap["mal"] = GenerateIssueRow(pkg,"mal")
	}
	if rv.License <= ut.Lic {
		fmt.Fprintf(&failString, "|License|%d|%d|\n", ToI(rv.License), ToI(ut.Lic))
		issueMap["lic"] = GenerateIssueRow(pkg,"lic")
	}
	if rv.Engineering <= ut.Lic {
		fmt.Fprintf(&failString, "|Engineering|%d|%d|\n", ToI(rv.Engineering), ToI(ut.Eng))
		issueMap["eng"] = GenerateIssueRow(pkg,"eng")
	}
	if rv.Author <= ut.Aut {
		fmt.Fprintf(&failString, "|Author|%d|%d|\n", ToI(rv.Author), ToI(ut.Aut))
		issueMap["aut"] = GenerateIssueRow(pkg,"aut")
	}

	fmt.Fprintf(&issueString, "#### Issues Summary\n")
	fmt.Fprintf(&issueString, "|Risk Domain|Risk Level|Title|\n")
	fmt.Fprintf(&issueString, "|-----------|----------|-----|\n")

	for _,v := range issueMap {
		fmt.Fprintf(&issueString, v)
	}

	if failString.Len() > 0 {
		gbl_didFail = true
		return headerString.String() + failString.String() + issueString.String()
	}
	return ""
}

func PRType(repo string, prNum int, provider int) string {
	diffText,err := GetPRDiff(repo, prNum, provider)
	if err != nil {
		panic(err)
	}
	prType,_,err   := DeterminePatchType(diffText)
	if err != nil {
		panic(err)
	}
	return prType
}

func Analyze(repo string, prNum int, ut UserThresholds) {
	var returnCode int
	provider := 0
	diffText,err := GetPRDiff(repo, prNum, provider)
	if err != nil {
		panic(err)
	}
	prType,lang,err   := DeterminePatchType(diffText)
	if err != nil {
		panic(err)
	}
	changes  := GetChanges(diffText)
	pkgVer   := GetChangedPackages(changes,prType)
	phylumJsonData := ReadPhylumAnalysis(fmt.Sprintf("./phylum_analysis_%s.json",lang))
	phylumRiskData, err := ParsePhylumRiskData(pkgVer, phylumJsonData, ut)
	if err != nil {
		returnCode = 5	 //incomplete packages
		fmt.Printf("[*] Phylum analysis for %s PR#%d INCOMPLETE\n", repo, prNum)
	}

	if len(phylumRiskData) > 0 {
		returnCode = 1

		const header = `### Phylum OSS Supply Chain Risk Analysis
		<details>
		<summary>Background</summary>

		<br />

		This repository uses a GitHub Action to automatically analyze the risk of new dependencies added to requirements.txt via Pull Request. An administrator of this repository has set score requirements for Phylum's five risk domains.<br /><br />

		If you see this comment, one or more dependencies added to the requirements.txt file in this Pull Request have failed Phylum's risk analysis.

		</details>

		`
		projectFooter := fmt.Sprintf("\n[View this project in Phylum UI](https://app.phylum.io/projects/%s)", phylumJsonData.Project)
		f1, err := os.Create(PR_COMMENT_FILENAME)
		defer f1.Close()
		if err != nil { // change to /home/runner for github
			panic(fmt.Errorf("couldn't open %s for write()", PR_COMMENT_FILENAME))
		}
		f1.WriteString(header)
		f1.WriteString(phylumRiskData)
		f1.WriteString(projectFooter)
		fmt.Printf("[*] wrote %d bytes to pr_comment.txt\n", len(header + phylumRiskData + projectFooter ))
		fmt.Printf("[*] Phylum analysis for %s PR#%d FAILED\n", repo, prNum)

	} else {
		returnCode = 0
		fmt.Printf("[*] Phylum analysis for %s PR#%d PASSED\n", repo, prNum)
	}

	f2, err := os.Create(RETURNCODE_FILENAME)
	defer f2.Close()
	if err != nil {
		panic(fmt.Errorf("couldn't open %s for write()", RETURNCODE_FILENAME))
	}
	n2,err := f2.WriteString(strconv.Itoa(returnCode))
	if err != nil {
		panic(fmt.Errorf("couldn't write to %s ", RETURNCODE_FILENAME))
	}
	fmt.Printf("[*] wrote %d bytes to returncode.txt\n", n2)
}