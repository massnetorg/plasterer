package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"massnet.org/mass/config"
	"massnet.org/mass/poc/pocutil"
	"massnet.org/mass/poc/wallet"
	_ "massnet.org/mass/poc/wallet/db/ldb"
)

const cmdInit = "init"

type initArgs struct {
	MinerConfig   string
	MinerPrivPass string
	DbDirs        []string
	DbNumbers     []int
}

func parseInitArgs() (args *initArgs, err error) {
	initFlags := flag.NewFlagSet(cmdInit, flag.ExitOnError)

	minerConfigPtr := initFlags.String("miner_config", "config.json", "miner config file")
	minerPrivPassPtr := initFlags.String("miner_priv_pass", "", "miner private password")
	dbDirsPtr := initFlags.String("db_dirs", "", "directories for massdb files, separated by comma")
	dbNumbersPtr := initFlags.String("db_numbers", "", "number of massdb files for each directory, separated by comma")

	if err := initFlags.Parse(os.Args[2:]); err != nil {
		return nil, err
	}

	// check miner private pass
	if len(*minerPrivPassPtr) == 0 {
		return nil, errors.New("missing miner_priv_pass")
	}

	// parse dbDirs
	dbDirs, err := parseDbDirs(*dbDirsPtr)
	if err != nil {
		return nil, err
	}
	// parse dbNumbers
	dbNumbers, err := parseDbNumbers(*dbNumbersPtr, len(dbDirs))
	if err != nil {
		return nil, err
	}

	args = &initArgs{
		MinerConfig:   *minerConfigPtr,
		MinerPrivPass: *minerPrivPassPtr,
		DbDirs:        dbDirs,
		DbNumbers:     dbNumbers,
	}
	return args, nil
}

func initPoCWallet(cfg *config.Config) (*wallet.PoCWallet, error) {
	pub := []byte(cfg.App.PubPassword)
	if len(pub) == 0 {
		return nil, errors.New("missing config.pub_pass")
	}

	manager, err := wallet.NewPoCWallet(cfg.Config, pub)
	if err != nil {
		return nil, err
	}

	accountIDs := manager.ListKeystoreNames()
	if len(accountIDs) == 0 {
		priv := []byte(cfg.PrivatePass)
		if len(priv) == 0 {
			return nil, errors.New("missing miner_priv_pass")
		}
		fmt.Println("initializing poc wallet:", cfg.Miner.MinerDir)
		accountID, err := manager.NewKeystore(priv, nil, "", &config.ChainParams, nil)
		if err != nil {
			return nil, fmt.Errorf("fail to create new keystore: %s, %w", cfg.Miner.MinerDir, err)
		}
		fmt.Println("poc wallet initialized:", accountID)
	} else {
		return nil, errors.New("cannot initialize existed poc wallet, please backup and remove current poc wallet")
	}
	return manager, nil
}

func generatePlastererConfig(pocWallet *wallet.PoCWallet, dbDirs []string, dbNumbers []int) (err error) {
	generatedDirs := make([]string, 0, len(dbDirs))
	defer func() {
		if err != nil {
			for _, dir := range generatedDirs {
				err := os.Remove(filepath.Join(dir, "pks.conf"))
				if err != nil {
					fmt.Println("warning: error removing generated plasterer config", dir, err)
				}
			}
		}
	}()
	for i, dir := range dbDirs {
		lines := make([]string, dbNumbers[i])
		for n := 0; n < dbNumbers[i]; n++ {
			pk, ordinal, err := pocWallet.GenerateNewPublicKey()
			if err != nil {
				return err
			}
			line := fmt.Sprintf("%d|%s|%s|%d",
				ordinal, pocutil.PubKeyHash(pk), hex.EncodeToString(pk.SerializeCompressed()), BitLength32)
			lines[n] = line
		}
		f, err := os.Create(filepath.Join(dir, "pks.conf"))
		if err != nil {
			return err
		}
		if _, err = f.WriteString(strings.Join(lines, "\n") + "\n"); err != nil {
			f.Close()
			return err
		}
		f.Close()
		generatedDirs = append(generatedDirs, dir)
	}
	return err
}

func printInitResults(configFile, minerPrivPass string, dbDirs []string, dbNumbers []int, uselessDbDirs []string) {
	dbResults := make([]string, len(dbDirs))
	proofDirs := make([]string, len(dbDirs))
	for i := range dbResults {
		dbResults[i] = fmt.Sprintf(`"directory": %s, "number": %d`, dbDirs[i], dbNumbers[i])
		proofDirs[i] = fmt.Sprintf(`      "%s"`, dbDirs[i])
	}
	dbResultsStr := strings.Join(dbResults, "\n")
	proofDirsStr := strings.Join(proofDirs, ",\n")

	fmt.Printf(`
Successfully initialized massminerd by plasterer-helper!

Summary for generation:
%s

Please manually modify the following items in your miner config file (%s):
{
  "miner": {
    "spacekeeper_backend": "spacekeeper.plasterer",
    "proof_dir": [
%s
    ],
    "private_password": "%s"
  }
}

Attention: DO NOT run massminerd while plasterer is running, or massdb files may be corrupted.

`, dbResultsStr, configFile, proofDirsStr, minerPrivPass)

	if len(uselessDbDirs) > 0 {
		fmt.Printf("Warning: the following directorities are not used:\n%s\n\n",
			strings.Join(uselessDbDirs, "\n"))
	}
}

func runInitCmd() (err error) {
	// parse args
	args, err := parseInitArgs()
	if err != nil {
		return err
	}
	// create non-exist directories
	createdDirs, err := createDbDirs(args.DbDirs)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			removeEmptyDirs(createdDirs)
		}
	}()
	// check db_dir emptiness
	if err = checkDbDirsEmptiness(args.DbDirs); err != nil {
		return err
	}
	// ensure proper capacity
	dbDirs, dbNumbers, uselessDbDirs, err := ensureCapacity(args.DbDirs, args.DbNumbers)
	if err != nil {
		return err
	}
	if len(dbDirs) == 0 {
		return errors.New("disk space is not enough")
	}
	// load miner config
	cfg, err := loadMinerConfig(args.MinerConfig)
	if err != nil {
		return err
	}
	if err := checkPassword(cfg.App.PubPassword); err != nil {
		return fmt.Errorf("invalid config.app.pub_password: %w", err)
	}
	cfg.PrivatePass = args.MinerPrivPass
	if err := checkPassword(cfg.PrivatePass); err != nil {
		return fmt.Errorf("invalid miner_priv_pass: %w", err)
	}
	// init poc wallet
	pocWallet, err := initPoCWallet(cfg)
	if err != nil {
		return err
	}
	defer pocWallet.Close()
	// generate plasterer config
	if err = generatePlastererConfig(pocWallet, dbDirs, dbNumbers); err != nil {
		return err
	}
	// print results
	printInitResults(args.MinerConfig, cfg.PrivatePass, dbDirs, dbNumbers, uselessDbDirs)
	return nil
}
