package main

import (
	"fmt"
)

func doSomething(i int) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("[WARN] Recovered from do something %d: %v\n", i, err)
		}
	}()

	if i == 1 || i == 3 {
		fmt.Println(i / (i-i))
	}
	fmt.Println("done something", i)
}

func myfun() {
	for i := 0; i < 6; i++ {
		doSomething(i)
	}
}

func main() {
	myfun()
}
