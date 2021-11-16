package main

import "fmt"

func main() {
	fmt.Println("hihi")
	body := GetPRDiff("peterjmorgan/analyze-pr-action-test",7)
	thing := DeterminePatchType(body)
	fmt.Println("PR Type: ", thing)
	changes := GetChanges(body)
	fmt.Println(changes)
	pkgVer := ParsePackageLock(changes)
	fmt.Println(pkgVer)
	pkgVer = ParseYarnLock(changes)
	fmt.Println(pkgVer)


}