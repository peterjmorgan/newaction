package main


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
		{"peterjmorgan/analyze-pr-action-test", 7,"js"},
		{"peterjmorgan/analyze-pr-action-test", 5,"py"},
		{"peterjmorgan/analyze-pr-action-test", 10,"py"},
	}
	ut := UserThresholds{
		0.65,
		0.65,
		0.65,
		0.65,
		0.65,
	}

	for _, td := range testData {
		//diffText 		:= GetPRDiff(td.repo,td.prNum)
		//prType   		:= DeterminePatchType(diffText)
		//changes  		:= GetChanges(diffText)
		//pkgVer   		:= GetChangedPackages(changes,prType)
		//phylumJsonData 	:= ReadPhylumAnalysis(fmt.Sprintf("./phylum_analysis_%s.json",td.lang))
		//phylumRiskData 	:= ParsePhylumRiskData(pkgVer, phylumJsonData, ut)
		//_ = phylumRiskData
		RunActionOne(td.repo, td.prNum, ut)
	}
}
