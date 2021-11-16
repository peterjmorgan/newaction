package main

import "fmt"

func main() {
	fmt.Println("Testing action - package-lock.json")
	body := GetPRDiff("peterjmorgan/analyze-pr-action-test",9)
	thing := DeterminePatchType(body)
	fmt.Println("PR Type: ", thing)
	changes := GetChanges(body)
	pkgVer := ParsePackageLock(changes)
	fmt.Println(pkgVer)
	pkgVer = ParseYarnLock(changes)
	fmt.Println(pkgVer)

	fmt.Println("Testing action - yarn.lock")
	body = GetPRDiff("peterjmorgan/analyze-pr-action-test",7)
	thing = DeterminePatchType(body)
	fmt.Println("PR Type: ", thing)
	changes = GetChanges(body)
	pkgVer = ParseYarnLock(changes)
	fmt.Println(pkgVer)

	fmt.Println("Testing action - requirements.txt")
	body = GetPRDiff("peterjmorgan/analyze-pr-action-test",5)
	thing = DeterminePatchType(body)
	fmt.Println("PR Type: ", thing)
	changes = GetChanges(body)
	pkgVer = ParseRequirementsDotTxt(changes)
	fmt.Println(pkgVer)

	fmt.Println("Testing action - Gemfile.lock")
	body = GetPRDiff("peterjmorgan/phylum-demo",54)
	thing = DeterminePatchType(body)
	fmt.Println("PR Type: ", thing)
	changes = GetChanges(body)
	pkgVer = ParseGemfileLock(changes)
	fmt.Println(pkgVer)
}
