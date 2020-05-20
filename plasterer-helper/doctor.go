package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

const cmdDoctor = "doctor"

type doctorArgs struct {
	MinerConfig string   `json:"miner_config"`
	DbDirs      []string `json:"db_dirs"`
}

func parseDoctorArgs() (args *doctorArgs, err error) {
	doctorFlags := flag.NewFlagSet(cmdDoctor, flag.ExitOnError)

	minerConfigPtr := doctorFlags.String("miner_config", "config.json", "miner config file")
	dbDirsPtr := doctorFlags.String("db_dirs", "", "directories for massdb files, separated by comma")

	if err := doctorFlags.Parse(os.Args[2:]); err != nil {
		return nil, err
	}

	// parse dbDirs
	dbDirs, err := parseDbDirs(*dbDirsPtr)
	if err != nil {
		return nil, err
	}

	args = &doctorArgs{
		MinerConfig: *minerConfigPtr,
		DbDirs:      dbDirs,
	}
	return args, nil
}

func runDoctorCmd() error {
	// parse args
	args, err := parseDoctorArgs()
	if err != nil {
		return err
	}
	fmt.Printf("Running plasterer-helper doctor...\n\n")

	// load and check miner config
	cfg, err := loadMinerConfig(args.MinerConfig)
	if err != nil {
		fmt.Printf("config error: cannot parse miner_config(%s), %v\n\n", args.MinerConfig, err)
	} else if len(cfg.App.PubPassword) == 0 {
		fmt.Printf("config error: app.pub_password cannot be empty\n\n")
	} else if err = checkPassword(cfg.App.PubPassword); err != nil {
		fmt.Printf("config error: app.pub_password is invalid, %v\n\n", err)
	}

	// check db_dirs
	usages, errs := getDbDirUsages(args.DbDirs)
	for i, dir := range args.DbDirs {
		fmt.Printf("db_dir: %s\n", dir)
		infos, err := ioutil.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("error: db_dir is not exist, please create directory\n\n")
			} else {
				fmt.Printf("error: %v\n\n", err)
			}
			continue
		}
		if len(infos) > 0 {
			fmt.Printf("error: db_dir must be empty, please backup and remove current files\n\n")
			continue
		}
		if errs[i] != nil {
			fmt.Printf("error: cannot get disk usage, %v\n\n", errs[i])
			continue
		}
		fmt.Printf("available disk size: %d GiB\nmax db number: %d\n\n",
			usages[i].Free/GiB, fixDbDirNumber(usages[i], 0))
	}

	fmt.Println("This is the end of doctor report.")
	return nil
}
