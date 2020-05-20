package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/disk"
	"massnet.org/mass/config"
	configpb "massnet.org/mass/config/pb"
	"massnet.org/mass/poc/wallet/keystore"
)

const (
	GiB         uint64 = 1 << 30
	GiB32              = GiB * 32
	BitLength32        = 32
)

func parseDbDirs(dbDirsStr string) ([]string, error) {
	if dbDirsStr == "" {
		return nil, errors.New("require at least one db_dir")
	}
	dbDirs := strings.Split(dbDirsStr, ",")
	dbDirsMap := make(map[string]struct{})
	for i, dir := range dbDirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return nil, err
		}
		if _, ok := dbDirsMap[absDir]; ok {
			return nil, fmt.Errorf("duplicate db_dir: index %d %s", i, absDir)
		}
		dbDirs[i] = absDir
		dbDirsMap[absDir] = struct{}{}
	}
	return dbDirs, nil
}

func parseDbNumbers(dbNumbersStr string, count int) (dbNumbers []int, err error) {
	dbNumbers = make([]int, count)

	if dbNumbersStr == "" {
		return dbNumbers, nil
	}

	numSlice := strings.Split(dbNumbersStr, ",")
	if len(numSlice) != count {
		return nil, fmt.Errorf("lenght of db_numbers should be %d, not %d", count, len(numSlice))
	}
	for i, str := range numSlice {
		num, err := strconv.Atoi(str)
		if err != nil {
			return nil, fmt.Errorf("parseDbNumbers: %w", err)
		}
		if num < 0 {
			return nil, fmt.Errorf("parseDbNumbers: index = %d, %d is smaller than 0", i, num)
		}
		dbNumbers[i] = num
	}
	return dbNumbers, nil
}

func createDbDirs(dbDirs []string) (createdDirs []string, err error) {
	createdDirs = make([]string, 0, len(dbDirs))
	for _, dir := range dbDirs {
		if info, err := os.Stat(dir); err == nil {
			if !info.IsDir() {
				return createdDirs, fmt.Errorf("db_dir is not directory: %s", dir)
			}
		} else if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0700); err != nil {
				return createdDirs, fmt.Errorf("cannot mkdirall: %s, %w", dir, err)
			}
			createdDirs = append(createdDirs, dir)
		} else {
			return createdDirs, fmt.Errorf("cannot get db_dir stat: %s, %w", dir, err)
		}
	}
	return
}

func getDbDirUsages(dbDirs []string) (usages []*disk.UsageStat, errs []error) {
	usages = make([]*disk.UsageStat, len(dbDirs))
	errs = make([]error, len(dbDirs))
	for i, dir := range dbDirs {
		usages[i], errs[i] = disk.Usage(dir)
	}
	return
}

func ensureCapacity(dbDirs []string, dbNumbers []int) (fixedDbDirs []string, fixedDbNumbers []int, uselessDbDirs []string, err error) {
	fixedDbDirs = make([]string, 0, len(dbDirs))
	fixedDbNumbers = make([]int, 0, len(dbDirs))
	uselessDbDirs = make([]string, 0)
	usages, errs := getDbDirUsages(dbDirs)
	for i, dir := range dbDirs {
		if errs[i] != nil {
			return nil, nil, nil, fmt.Errorf("get dir usage: %s, %w", dir, errs[i])
		}
		num := fixDbDirNumber(usages[i], dbNumbers[i])
		if num > 0 {
			fixedDbDirs = append(fixedDbDirs, dir)
			fixedDbNumbers = append(fixedDbNumbers, num)
		} else {
			uselessDbDirs = append(uselessDbDirs, dir)
		}
	}
	return
}

func fixDbDirNumber(usage *disk.UsageStat, dbNumber int) (fixedDbNumber int) {
	max := int(usage.Free/GiB32) - 3
	fixedDbNumber = dbNumber
	if fixedDbNumber == 0 || fixedDbNumber > max {
		fixedDbNumber = max
	}
	if fixedDbNumber < 0 {
		fixedDbNumber = 0
	}
	return
}

func removeEmptyDirs(dbDirs []string) {
	for _, dir := range dbDirs {
		err := os.Remove(dir)
		if err != nil {
			fmt.Println("warning: error removing empty dir", dir, err)
		}
	}
}

func checkDbDirsEmptiness(dbDirs []string) error {
	for _, dir := range dbDirs {
		infos, err := ioutil.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("check db_dir emptiness: %s, %w", dir, err)
		}
		if len(infos) > 0 {
			return fmt.Errorf("db_dir must be empty: %s", dir)
		}
	}
	return nil
}

func loadMinerConfig(configFile string) (cfg *config.Config, err error) {
	cfg = &config.Config{
		ConfigFile: configFile,
		Config:     configpb.NewConfig(),
	}
	cfg, err = config.LoadConfig(cfg)
	if err != nil {
		return cfg, err
	}
	return config.CheckConfig(cfg)
}

func checkPassword(password string) error {
	pass := []byte(password)
	if length := len(pass); length < 6 || length > 40 {
		return errors.New("the recommended length of password is between 6 and 40")
	}
	if !keystore.ValidatePassphrase(pass) {
		return errors.New("the password is only allowed to contain numbers, letters and some symbols(@#$%^&)")
	}
	return nil
}
