package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sourcegraph/go-diff/diff"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

func GetPRDiff(repo string, prNum int) []byte {
	//if strings.ContainsRune(repo,'-') {
	//	repo = strings.ReplaceAll(repo,"-","_")
	//}

	url := fmt.Sprintf("https://patch-diff.githubusercontent.com/raw/%s/pull/%d.diff", repo, prNum)

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

func ParseGemfileLock(changes []string) []pkgVerTuple {
	nameVerPat := regexp.MustCompile(`\s{4}(.*?)\ \((.*?)\)`)
	pkgVer := make([]pkgVerTuple,0)
	for _,line := range changes {
		if nameVerPat.MatchString(line) {
			nameVerMatch := nameVerPat.FindAllStringSubmatch(line, -1)
			pkgVer = append(pkgVer, pkgVerTuple{nameVerMatch[0][1], nameVerMatch[0][2]})
		}
	}
	return pkgVer
}

func GetChangedPackages(changes []string, prType string) []pkgVerTuple {
	var pkgVer []pkgVerTuple
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
	Issues       []interface{} `json:"issues"`
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
func ParsePhylumRiskData(pkgVer []pkgVerTuple, phylumJson PhylumJson, ut UserThresholds) string {
	//resultPackages := make([]Package,0)
	results := make([]string,0)
	incompletes := make([]pkgVerTuple,0)

	for _, pv := range pkgVer {
		for _, pkg := range phylumJson.Packages {
			if pv.name == pkg.Name && pv.version == pkg.Version {
				switch pkg.Status {
				case "complete":
					fmt.Println("[✅ COMPLETE] ", pkg.Name)
					results = append(results, CheckRiskScores(pkg, ut))
				case "incomplete":
					fmt.Println("[❌ INCOMPLETE] ", pkg.Name)
					incompletes = append(incompletes,pkgVerTuple{pkg.Name, pkg.Version})
				}
			}
		}
	}
	if len(incompletes) > 0 {
		fmt.Printf("[❌ ERROR] Phylum status for %d packages was incomplete\n", len(incompletes))
		panic(errors.New("baaad"))
	}
	return strings.Join(results,"")
}

func CheckRiskScores(pkg Package, ut UserThresholds) string {
	var headerString, failString, issueString strings.Builder
	issueFlags := make([]string,0)
	issueMap := make(map[string]string,0)
	rv := pkg.RiskVectors
	fmt.Fprintf(&headerString, "### Package: `%s@%s` failed\n", pkg.Name, pkg.Version)
	fmt.Fprintf(&headerString, "|Risk Domain|Identified Score|Requirement|\n")
	fmt.Fprintf(&headerString, "|-----------|----------------|-----------|\n")
	if rv.Vulnerability <= ut.Vul {
		fmt.Fprintf(&failString, "|Software Vulnerability|%d|%d|\n", rv.Vulnerability, ut.Vul)
		issueFlags = append(issueFlags,"vul")
		var singleIssue strings.Builder
		for _, vuln := range pkg.Vulnerabilities {
			fmt.Fprintf(&singleIssue,"|%s|%s|%s\n", "Vuln",vuln.RiskLevel, vuln.Title)
		}
		issueMap["vul"] = singleIssue.String()
	}
	if rv.MaliciousCode <= ut.Mal {
		fmt.Fprintf(&failString, "|Malicious Code|%d|%d|\n", rv.MaliciousCode, ut.Mal)
		issueFlags = append(issueFlags,"mal")
	}
	if rv.License <= ut.Lic {
		fmt.Fprintf(&failString, "|License|%d|%d|\n", rv.License, ut.Lic)
		issueFlags = append(issueFlags,"lic")
	}
	if rv.Engineering <= ut.Lic {
		fmt.Fprintf(&failString, "|Engineering|%d|%d|\n", rv.Engineering, ut.Eng)
		issueFlags = append(issueFlags,"eng")
	}
	if rv.Author <= ut.Aut {
		fmt.Fprintf(&failString, "|Author|%d|%d|\n", rv.Author, ut.Aut)
		issueFlags = append(issueFlags,"aut")
	}

	fmt.Fprintf(&issueString, "\n")
	fmt.Fprintf(&issueString, "#### Issues Summary\n")
	fmt.Fprintf(&issueString, "|Risk Domain|Risk Level|Title|\n")
	fmt.Fprintf(&issueString, "|-----------|----------|-----|\n")

	for _,v := range issueMap {
		fmt.Fprintf(&issueString, v)
	}

	if failString.Len() > 0 {
		return headerString.String() + failString.String() + issueString.String()
	}
	return ""
}