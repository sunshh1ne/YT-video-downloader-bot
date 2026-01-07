package main

import (
	"fmt"
)

func main() {
	var a, b int
	fmt.Print("Enter a: ")
	fmt.Scan(&a)
	fmt.Print("Enter b: ")
	fmt.Scan(&b)
	fmt.Printf("The sum of %d and %d is %d\n", a, b, a+b)
}
