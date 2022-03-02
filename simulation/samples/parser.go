package sample

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/ChainBoost/onet/log"
)

type Parser struct {
	Path      string
	Magic     [4]byte
	CurrentId uint32
}

func NewParser(path string, magic [4]byte) (parser *Parser, err error) {
	parser = new(Parser)
	parser.Path = path
	parser.Magic = magic
	parser.CurrentId = 0
	return
}

// SimulDirToBlockDir creates a path to the 'protocols/byzcoin/block'-dir by
// using 'dir' which comes from 'cothority/simul'
// If that directory doesn't exist, it will be created.
func SimulDirToBlockDir(dir string) string {
	reg, _ := regexp.Compile("simul/.*")
	//raha what direc
	blockDir := string(reg.ReplaceAll([]byte(dir), []byte("protocols/byzcoin/block")))
	if _, err := os.Stat(blockDir); os.IsNotExist(err) {
		if err := os.Mkdir(blockDir, 0777); err != nil {
			log.Error("Couldn't create blocks directory", err)
		}
	}
	return blockDir
}

// CheckBlockAvailable looks if the directory with the block exists or not.
// It takes 'dir' as the base-directory, generated from 'cothority/simul'.
func GetBlockName(dir string) string {
	blockDir := SimulDirToBlockDir(dir)
	m, _ := filepath.Glob(blockDir + "/*.dat")
	if m != nil {
		return m[0]
	} else {
		return ""
	}
}

// Gets the block-directory starting from the current directory - this will
// hold up when running it with 'simul'
func GetBlockDir() string {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal("Couldn't get working dir:", err)
	}
	return SimulDirToBlockDir(dir)
}

// DownloadBlock takes 'dir' as the directory where to download the block.
// It returns the downloaded file
func DownloadBlock(dir string) (string, error) {
	blockDir := SimulDirToBlockDir(dir)
	cmd := exec.Command("wget", "--no-check-certificate", "-O",
		blockDir+"/blk00000.dat", "-c",
		//"https://icsil1-box.epfl.ch:5001/fbsharing/IzTFdOxf")
		"https://blockchain.info/rawblock/0000000000000bae09a7a393a8acded75aa67e46cb81f7acaa5ad94f9eacd103?format=hex")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Lvl1("Cmd is", cmd)
	if err := cmd.Start(); err != nil {
		return "", err
	}
	if err := cmd.Wait(); err != nil {
		return "", err
	}
	return GetBlockName(dir), nil
}

// EnsureBlockIsAvailable tests if the block is already downloaded, else it will
// download it. Finally the block will be copied to the 'simul'-provided
// directory for simulation.
func EnsureBlockIsAvailable(dir string) error {
	block := GetBlockName(dir)
	if block == "" {
		var err error
		block, err = DownloadBlock(dir)
		if err != nil || block == "" {
			return err
		}
	}
	destDir := SimulDirToBlockDir(dir)
	//if err := os.Mkdir(destDir, 0777); err != nil {
	//	return err
	//}
	cmd := exec.Command("cp", block, destDir)
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}
