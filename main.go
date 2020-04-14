package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Println(os.Args[1])
	fmt.Println(os.Args[2:])

	for i := 0; i < len(os.Args); i++ {
		fmt.Printf("index %d", i)
		fmt.Printf(" element %s\n", os.Args[i])
	}

	var args = strings.Join(os.Args, " ")
	fmt.Printf(args)
	var s = strings.ToUpper(args)
	fmt.Printf(s)

}
