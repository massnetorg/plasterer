package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("subcommand not found")
		os.Exit(1)
	}

	var cmd = os.Args[1]
	var err error
	switch cmd {
	case cmdInit:
		err = runInitCmd()
	case cmdDoctor:
		err = runDoctorCmd()
	default:
		err = fmt.Errorf("unknown subcommand: %s", cmd)
	}

	if err != nil {
		fmt.Printf("%s error: %v\n", cmd, err)
		os.Exit(3)
	}
}
