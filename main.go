package main

import "fmt"

func main() {
	fmt.Println("hihi")
	body := GetPRDiff("peterjmorgan/phylum-demo",53)
	thing := DeterminePatchType(body)
	fmt.Println("PR Type: ", thing)
	changes := GetChanges(body)
	fmt.Println(changes)
	pkgVer := ParsePackageLock(changes)
	fmt.Println(pkgVer)


}