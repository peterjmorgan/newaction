package main

import "fmt"

type UserThresholds struct {
	Vul float64
	Mal float64
	Lic float64
	Eng float64
	Aut float64
}

func main() {
	testData := []struct{
		repo string
		prNum int
		lang string
	}{
		{"peterjmorgan/analyze-pr-action-test", 9,"js"},
		//{"peterjmorgan/analyze-pr-action-test", 7,"js"},
		//{"peterjmorgan/analyze-pr-action-test", 5,"py"},
	}
	ut := UserThresholds{0.25,0.25,0.25,0.25,0.25}

	for _, td := range testData {
		diffText := GetPRDiff(td.repo,td.prNum)
		prType   := DeterminePatchType(diffText)
		changes  := GetChanges(diffText)
		pkgVer   := GetChangedPackages(changes,prType)
		phylumJsonData := ReadPhylumAnalysis(fmt.Sprintf("./phylum_analysis_%s.json",td.lang))
		phylumRiskData := ParsePhylumRiskData(pkgVer, phylumJsonData, ut)
		_ = phylumRiskData


	}
	//// JavaScript - package-lock.json ----------------------------------------------------------------------------------
	//fmt.Println("Testing action - package-lock.json")
	//body := GetPRDiff("peterjmorgan/analyze-pr-action-test",9)
	//prType := DeterminePatchType(body)
	//fmt.Println("PR Type: ", prType)
	//changes := GetChanges(body)
	//pkgVer := ParsePackageLock(changes)
	//fmt.Println(pkgVer)
	//pkgVer = ParseYarnLock(changes)
	//fmt.Println(pkgVer)
	//
	//// JavaScript - Yarn.lock ------------------------------------------------------------------------------------------
	//fmt.Println("Testing action - yarn.lock")
	//body = GetPRDiff("peterjmorgan/analyze-pr-action-test",7)
	//prType = DeterminePatchType(body)
	//fmt.Println("PR Type: ", prType)
	//changes = GetChanges(body)
	//pkgVer = ParseYarnLock(changes)
	//fmt.Println(pkgVer)
	//
	//// Python - requirements.txt----------------------------------------------------------------------------------------
	//fmt.Println("Testing action - requirements.txt")
	//body = GetPRDiff("peterjmorgan/analyze-pr-action-test",5)
	//prType = DeterminePatchType(body)
	//fmt.Println("PR Type: ", prType)
	//changes = GetChanges(body)
	////pkgVer = ParseRequirementsDotTxt(changes)
	//pkgVer = GetChangedPackages(changes, prType)
	//fmt.Println(pkgVer)
	//phylumJsonData := ReadPhylumAnalysis("./phylum_analysis_py.json")
	//fmt.Println(phylumJsonData)
	//phylumRiskData := ParsePhylumRiskData(pkgVer, phylumJsonData)
	//fmt.Println(phylumRiskData)
	//
	//// Ruby - Gemfile.lock ---------------------------------------------------------------------------------------------
	//fmt.Println("Testing action - Gemfile.lock")
	//body = GetPRDiff("peterjmorgan/phylum-demo",54)
	//prType = DeterminePatchType(body)
	//fmt.Println("PR Type: ", prType)
	//changes = GetChanges(body)
	//pkgVer = ParseGemfileLock(changes)
	//fmt.Println(pkgVer)
}
