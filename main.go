package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

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
	_ = testData
	//ut := UserThresholds{
	//	0.65,
	//	0.65,
	//	0.65,
	//	0.65,
	//	0.65,
	//}

	prTypeCmd := flag.NewFlagSet("pr_type",flag.ExitOnError)
	//prTypeCmdRepo := prTypeCmd.String("r","repo","Repository (google/subcommand)")
	//prTypeCmdPR := prTypeCmd.Int("n", 1,"Number of PR/MR")
	//prTypeCmdProvider := prTypeCmd.Int("p",0, "0 for GitHub, 1 for GitLab")

	analyzeCmd := flag.NewFlagSet("analyze", flag.ExitOnError)
	analyzeCmdRepo := analyzeCmd.String("r","repo","Repository (google/subcommand)")
	analyzeCmdPR := analyzeCmd.Int("n", 1,"Number of PR/MR")
	analyzeCmdThresholds := analyzeCmd.String("t", "thresholds", "Comma-separated float thresholds (VUL,MAL,LIC,ENG,AUT")

	if len(os.Args) < 2 {
		fmt.Println("expected 'pr_type' or 'analyze' subcommands")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "pr_type":
		prTypeCmd.Parse(os.Args[2:])
		//prType := PRType(*prTypeCmdRepo,*prTypeCmdPR, *prTypeCmdProvider)
		prType := PRType()
		fmt.Println(prType)
		os.Exit(0)
	case "analyze":
		//TODO: add argc checking here
		analyzeCmd.Parse(os.Args[2:])
		thresholds := strings.Split(*analyzeCmdThresholds,"," )
		someFloats := make([]float64,0)
		for _,t := range thresholds {
			aFloat, err := strconv.ParseFloat(t,64)
			if err != nil {
				fmt.Printf("couldn't parse float from %s",t)
				os.Exit(1)
			}
			someFloats = append(someFloats,aFloat)
		}
		ut := UserThresholds{
			someFloats[0],
			someFloats[1],
			someFloats[2],
			someFloats[3],
			someFloats[4],
		}
		Analyze(*analyzeCmdRepo, *analyzeCmdPR, ut)
		os.Exit(0)
	}

	//for _, td := range testData {
	//	//diffText 		:= GetPRDiff(td.repo,td.prNum)
	//	//prType   		:= DeterminePatchType(diffText)
	//	//changes  		:= GetChanges(diffText)
	//	//pkgVer   		:= GetChangedPackages(changes,prType)
	//	//phylumJsonData 	:= ReadPhylumAnalysis(fmt.Sprintf("./phylum_analysis_%s.json",td.lang))
	//	//phylumRiskData 	:= ParsePhylumRiskData(pkgVer, phylumJsonData, ut)
	//	//_ = phylumRiskData
	//	Analyze(td.repo, td.prNum, ut)
	//}
}
