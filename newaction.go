package newaction

import (
	"fmt"
	"io"
	"net/http"
)

func getPRDiff(repo string, prNum int) string {
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
	return string(body)
}

//func main() {
//	repo := "peterjmorgan/analyze-pr-action-test"
//	pr := 5
//	result := getPRDiff(repo, pr)
//
//	fmt.Println(string(result))
//
//}