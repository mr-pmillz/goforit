/*
Package main

Copyright © 2023 MrPMillz
*/
package main

import (
	"github.com/mr-pmillz/goforit/cmd"
	"log"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		log.Fatalf("error running root command:\n%+v\n", err)
	}
}
