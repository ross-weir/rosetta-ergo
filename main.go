package main

import (
	"os"

	"github.com/ross-weir/rosetta-ergo/cmd"

	"github.com/fatih/color"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		color.Red(err.Error())
		os.Exit(1)
	}
}
