package main

import (
	"github.com/zoroqi/skill_manager/internal/cmd"
	"os"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
