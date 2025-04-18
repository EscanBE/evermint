//go:build renamechain

package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/EscanBE/evermint/constants"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

// TODO: remove this 'rename_chain' directory after renamed chain

// NewRenameChainCmd creates a helper command that convert account bech32 address into hex address or vice versa
//
//goland:noinspection GoBoolExpressions
func main() {
	if !isLinux() && !isDarwin() {
		fmt.Println("ERR: Only support Linux & MacOS")
		os.Exit(1)
	}

	if !isPathFileOrDirExists("go.mod") {
		fmt.Println("ERR: must run at root of project, go.mod could not be found")
		os.Exit(1)
	}

	if !isPathFileOrDirExists(fmt.Sprintf("cmd/%s", EvermintOg_ApplicationBinaryName)) {
		fmt.Printf("ERR: must run at root of project, cmd/%s could not be found\n", EvermintOg_ApplicationBinaryName)
		os.Exit(1)
	}

	// validation
	if !regexp.MustCompile("^https://github.com/[^/]+/[^/]+$").MatchString(constants.GitHubRepo) {
		fmt.Println("ERR: invalid GitHub repo format of", constants.GitHubRepo)
		fmt.Println("It should be in format: https://github.com/<owner>/<repo>")
		os.Exit(1)
	} else if constants.GitHubRepo == EvermintOg_GitHubRepo {
		fmt.Println("ERR: must change github repo at variable constants.GitHubRepo")
		os.Exit(1)
	}

	if constants.Bech32Prefix == EvermintOg_Bech32Prefix {
		fmt.Println("ERR: must change bech32 prefix at variable constants.Bech32Prefix")
		os.Exit(1)
	} else if strings.ToLower(constants.Bech32Prefix) != constants.Bech32Prefix {
		fmt.Println("ERR: bech32 prefix must be in lower case")
		os.Exit(1)
	}

	if constants.ApplicationName == EvermintOg_ApplicationName {
		fmt.Println("ERR: must change application name at variable constants.ApplicationName")
		os.Exit(1)
	} else if strings.ToLower(constants.ApplicationName) != constants.ApplicationName {
		fmt.Println("ERR: application name must be in lower case")
		os.Exit(1)
	}

	if constants.ApplicationBinaryName == EvermintOg_ApplicationBinaryName {
		fmt.Println("ERR: must change application binary name at variable constants.ApplicationBinaryName")
		os.Exit(1)
	} else if strings.ToLower(constants.ApplicationBinaryName) != constants.ApplicationBinaryName {
		fmt.Println("ERR: application binary name must be in lower case")
		os.Exit(1)
	}

	if constants.ApplicationHome == EvermintOg_ApplicationHome {
		fmt.Println("ERR: must change application home at variable constants.ApplicationHome")
		os.Exit(1)
	} else if strings.ToLower(constants.ApplicationHome) != constants.ApplicationHome {
		fmt.Println("ERR: application home must be in lower case")
		os.Exit(1)
	} else if !strings.HasPrefix(constants.ApplicationHome, ".") {
		fmt.Println("ERR: application home MUST start with '.'")
		os.Exit(1)
	}

	if constants.MainnetChainID == EvermintOg_MainnetChainID {
		fmt.Println("ERR: must change mainnet chain id at variable constants.MainnetChainID")
		os.Exit(1)
	} else if strings.ToLower(constants.MainnetChainID) != constants.MainnetChainID {
		fmt.Println("ERR: mainnet chain id must be in lower case")
		os.Exit(1)
	} else if !strings.HasSuffix(constants.MainnetChainID, fmt.Sprintf("_%d", constants.MainnetEIP155ChainId)) {
		fmt.Printf("ERR: invalid mainnet chain-id: (constants.MainnetChainID) %s must containts %d (constants.MainnetEIP155ChainId)\n", constants.MainnetChainID, constants.MainnetEIP155ChainId)
		os.Exit(1)
	}

	if constants.TestnetChainID == EvermintOg_TestnetChainID {
		fmt.Println("ERR: must change testnet chain id at variable constants.TestnetChainID")
		os.Exit(1)
	} else if strings.ToLower(constants.TestnetChainID) != constants.TestnetChainID {
		fmt.Println("ERR: testnet chain id must be in lower case")
		os.Exit(1)
	} else if !strings.HasSuffix(constants.TestnetChainID, fmt.Sprintf("_%d", constants.TestnetEIP155ChainId)) {
		fmt.Printf("ERR: invalid testnet chain-id: (constants.TestnetChainID) %s must containts %d (constants.TestnetEIP155ChainId)\n", constants.TestnetChainID, constants.TestnetEIP155ChainId)
		os.Exit(1)
	}

	if constants.DevnetChainID == EvermintOg_DevnetChainID {
		fmt.Println("ERR: must change devnet chain id at variable constants.DevnetChainID")
		os.Exit(1)
	} else if strings.ToLower(constants.DevnetChainID) != constants.DevnetChainID {
		fmt.Println("ERR: devnet chain id must be in lower case")
		os.Exit(1)
	} else if !strings.HasSuffix(constants.DevnetChainID, fmt.Sprintf("_%d", constants.DevnetEIP155ChainId)) {
		fmt.Printf("ERR: invalid devnet chain-id: (constants.DevnetChainID) %s must containts %d (constants.DevnetEIP155ChainId)\n", constants.DevnetChainID, constants.DevnetEIP155ChainId)
		os.Exit(1)
	}

	if strings.ToLower(constants.BaseDenom) != constants.BaseDenom {
		fmt.Println("ERR: base denom must be in lower case")
		os.Exit(1)
	}

	if strings.ToLower(constants.DisplayDenom) != constants.DisplayDenom {
		fmt.Println("ERR: display denom must be in lower case")
		os.Exit(1)
	}

	if strings.ToUpper(constants.SymbolDenom) != constants.SymbolDenom {
		fmt.Println("ERR: symbol denom must be in upper case")
		os.Exit(1)
	}

	_ = os.RemoveAll("rename_chain")

	ogGitHubWithoutScheme := strings.TrimSuffix(strings.Split(EvermintOg_GitHubRepo, "://")[1], ".git")
	ogGoModule := ogGitHubWithoutScheme
	splOgGitHub := strings.Split(ogGitHubWithoutScheme, "/")
	ogGitOwnerAndRepo := fmt.Sprintf("%s/%s", splOgGitHub[len(splOgGitHub)-2], splOgGitHub[len(splOgGitHub)-1])
	ogGitHubWithoutRepo := strings.TrimSuffix(ogGitHubWithoutScheme, fmt.Sprintf("/%s", splOgGitHub[len(splOgGitHub)-1]))

	newGitHubWithoutScheme := strings.TrimSuffix(strings.Split(constants.GitHubRepo, "://")[1], ".git")
	newGoModule := newGitHubWithoutScheme
	splNewGitHub := strings.Split(newGitHubWithoutScheme, "/")
	newGitOwnerAndRepo := fmt.Sprintf("%s/%s", splNewGitHub[len(splNewGitHub)-2], splNewGitHub[len(splNewGitHub)-1])
	newGitHubWithoutRepo := strings.TrimSuffix(newGitHubWithoutScheme, fmt.Sprintf("/%s", splNewGitHub[len(splNewGitHub)-1]))

	anyFiles := getFileListRecursive()
	goFiles := getFileListRecursive("go")

	for _, anyFile := range anyFiles {
		sed(anyFile, EvermintOg_MainnetFullChainId, constants.MainnetFullChainId)
		sed(anyFile, EvermintOg_TestnetFullChainId, constants.TestnetFullChainId)
		sed(anyFile, EvermintOg_DevnetFullChainId, constants.DevnetFullChainId)
	}

	for _, anyFile := range anyFiles {
		sed(anyFile, ogGoModule, newGoModule)
		sed(anyFile, ogGitHubWithoutScheme, newGitHubWithoutScheme)
		sed(anyFile, ogGitHubWithoutRepo, newGitHubWithoutRepo)
		sed(anyFile, ogGitOwnerAndRepo, newGitOwnerAndRepo)
		sed(anyFile, EvermintOg_ApplicationHome, constants.ApplicationHome)
	}

	for _, shellFile := range getFileListRecursive("sh", "bat") {
		sed(shellFile, EvermintOg_BaseDenom, constants.BaseDenom)
		sed(shellFile, EvermintOg_ApplicationBinaryName, constants.ApplicationBinaryName)
	}

	for _, goFile := range goFiles {
		sed(goFile, EvermintOg_BaseDenom, constants.BaseDenom)
	}

	for _, jsonFile := range getFileListRecursive("json") {
		sed(jsonFile, strings.ToUpper(EvermintOg_ApplicationName[:1])+EvermintOg_ApplicationName[1:], strings.ToUpper(constants.ApplicationName[:1])+strings.ToLower(constants.ApplicationName[1:]))
		sed(jsonFile, fmt.Sprintf("\"%s\"", EvermintOg_BaseDenom), fmt.Sprintf("\"%s\"", constants.BaseDenom))
		sed(jsonFile, fmt.Sprintf("\"%s\"", EvermintOg_DisplayDenom), fmt.Sprintf("\"%s\"", constants.DisplayDenom))
		sed(jsonFile, fmt.Sprintf("\"%s\"", EvermintOg_SymbolDenom), fmt.Sprintf("\"%s\"", constants.SymbolDenom))
	}

	for _, anyFile := range anyFiles {
		sed(anyFile, EvermintOg_ApplicationBinaryName, constants.ApplicationBinaryName)
		sed(anyFile, strings.ToUpper(EvermintOg_ApplicationName[:1])+EvermintOg_ApplicationName[1:], strings.ToUpper(constants.ApplicationName[:1])+strings.ToLower(constants.ApplicationName[1:]))
		sed(anyFile, EvermintOg_ApplicationName, constants.ApplicationName)
	}

	sed(path.Join("cmd", EvermintOg_ApplicationBinaryName, "root.go"), strings.ToUpper(EvermintOg_ApplicationName), strings.ToUpper(constants.ApplicationName))

	{ // restore dependency in go.mod & go.sum
		const invalidGethFork = "github.com/EscanBE/go-ethereum-for-" + constants.ApplicationName
		const correctGethFork = "github.com/EscanBE/go-ethereum-for-evermint"
		sed("go.mod", invalidGethFork, correctGethFork)
		sed("go.sum", invalidGethFork, correctGethFork)
	}

	patternMarker := regexp.MustCompile(`marker\.(\w+)\("(\w+)"\)`)
	for _, goFile := range goFiles {
		if strings.HasSuffix(goFile, "_test.go") {
			// ok
		} else if strings.HasSuffix(goFile, "integration_test_util/accounts.go") {
			// ok
		} else {
			continue
		}

		if !isFileContains(goFile, "marker.ReplaceAble") {
			continue
		}

		bz, err := os.ReadFile(goFile)
		if err != nil {
			fmt.Println("ERR: failed to read file", goFile, ":", err)
			os.Exit(1)
		}

		matches := patternMarker.FindAllStringSubmatch(string(bz), -1)
		if len(matches) < 1 {
			continue
		}

		for _, match := range matches {
			if len(match) != 3 {
				panic(fmt.Sprintf("unexpected matches on %s with values matched values: %s", goFile, strings.Join(match, ", ")))
			}

			ogBech32Address := match[2]

			hrp := constants.Bech32Prefix
			if strings.HasPrefix(ogBech32Address, EvermintOg_Bech32PrefixValAddr) {
				hrp = constants.Bech32PrefixValAddr
			} else if strings.HasPrefix(ogBech32Address, EvermintOg_Bech32PrefixValConsAddr) {
				hrp = constants.Bech32PrefixConsAddr
			}

			if match[1] == "ReplaceAbleAddress" {
				_, bytes, err := bech32.DecodeAndConvert(ogBech32Address)
				if err != nil {
					fmt.Println("failed to decode bech32 address:", err)
					continue
				}
				newBech32Address, err := bech32.ConvertAndEncode(hrp, bytes)
				if err != nil {
					fmt.Println("failed to encode bech32 address", ogBech32Address, ":", err)
					os.Exit(1)
				}

				sed(goFile, match[0], fmt.Sprintf("\"%s\"", newBech32Address))
			} else if match[1] == "ReplaceAbleWithBadChecksum" {
				bytes := make([]byte, 20)
				_, _ = rand.Read(bytes)

				newBech32Address, err := bech32.ConvertAndEncode(hrp, bytes)
				if err != nil {
					fmt.Println("failed to encode bech32 address", "0x"+hex.EncodeToString(bytes), ":", err)
					os.Exit(1)
				}

				// clear last 6 characters
				if newBech32Address[len(newBech32Address)-6:] != "aaaaaa" {
					newBech32Address = newBech32Address[:len(newBech32Address)-6] + "aaaaaa"
				} else {
					newBech32Address = newBech32Address[:len(newBech32Address)-6] + "bbbbbb"
				}

				sed(goFile, match[0], fmt.Sprintf("\"%s\"", newBech32Address))
			} else {
				panic(fmt.Sprintf("unexpected matches on %s with function signature: %s", goFile, match[1]))
			}
		}

		sed(goFile, "\""+newGoModule+"/rename_chain/marker\"", "")
	}

	launchAppWithDirectStd("mv", path.Join("cmd", EvermintOg_ApplicationBinaryName), path.Join("cmd", constants.ApplicationBinaryName))

	{ // fix proto for Ethermint native module
		launchAppWithDirectStd("mv", path.Join("proto", splOgGitHub[len(splOgGitHub)-1]), path.Join("proto", splNewGitHub[len(splNewGitHub)-1]))
		launchAppWithDirectStd("make", "proto-gen")
	}
}

func isLinux() bool {
	//goland:noinspection GoBoolExpressions
	return runtime.GOOS == "linux"
}

func isDarwin() bool {
	//goland:noinspection GoBoolExpressions
	return runtime.GOOS == "darwin"
}

func isPathFileOrDirExists(file string) bool {
	_, err := os.Stat(file)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}

func isFileContains(file, pattern string) bool {
	bz, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("cat '%s' | grep '%s' | wc -l | tr -d ' '", file, pattern)).Output()
	if err != nil {
		fmt.Println("ERR: failed to check if pattern exists in file:", err)
		fmt.Println("Pattern:", pattern)
		fmt.Println("File:", file)
		os.Exit(1)
	}

	count := strings.TrimSpace(string(bz))
	if count == "0" {
		return false
	}
	return true
}

func sed(file, pattern, replacement string) {
	if pattern == replacement {
		return
	}

	if !isFileContains(file, pattern) {
		return
	}

	var sedCmd []string

	sedCmd = append(sedCmd, "sed", "-i")

	if isDarwin() {
		sedCmd = append(sedCmd, "''")
	}
	splitor := "/"
	if strings.Contains(pattern, "/") {
		splitor = "#"
	}

	if strings.Contains(pattern, splitor) {
		fmt.Println("Pattern contains splitor", splitor, "which is not supported:", pattern)
		os.Exit(1)
	}

	if strings.Contains(replacement, splitor) {
		fmt.Println("Replacement contains splitor", splitor, "which is not supported:", replacement)
		os.Exit(1)
	}

	sedCmd = append(sedCmd, fmt.Sprintf("'s%s%s%s%s%sg'", splitor, strings.ReplaceAll(pattern, ".", "\\."), splitor, strings.ReplaceAll(replacement, ".", "\\."), splitor), fmt.Sprintf("'%s'", file))

	ec := launchAppWithDirectStd(
		"/bin/bash", "-c", strings.Join(sedCmd, " "),
	)

	if ec != 0 {
		fmt.Println("ERR: failed to sed", file)
		fmt.Println(strings.Join(sedCmd, " "))
		os.Exit(1)
	}
}

func launchAppWithDirectStd(appName string, args ...string) int {
	return launchAppWithSetup(appName, args, func(launchCmd *exec.Cmd) {
		launchCmd.Stdin = os.Stdin
		launchCmd.Stdout = os.Stdout
		launchCmd.Stderr = os.Stderr
	})
}

func launchAppWithSetup(appName string, args []string, setup func(launchCmd *exec.Cmd)) int {
	launchCmd := exec.Command(appName, args...)
	setup(launchCmd)
	err := launchCmd.Run()
	if err != nil {
		fmt.Printf("problem when running process %s: %s\n", appName, err.Error())
		return 1
	}
	return 0
}

func getFileListRecursive(whiteListExt ...string) []string {
	mapOfExt := make(map[string]bool)
	for _, ext := range whiteListExt {
		mapOfExt[ext] = true
	}

	var fileList []string
	err := filepath.Walk(".",
		func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if strings.HasPrefix(p, ".git") {
				return nil
			}
			if strings.HasPrefix(p, ".idea") {
				return nil
			}
			if strings.HasPrefix(p, "build/") {
				return nil
			}
			if strings.HasPrefix(p, "rename_chain/") {
				return nil
			}
			if strings.Contains(p, "/node_modules/") {
				return nil
			}
			if strings.HasSuffix(p, "rename-chain.sh") {
				return nil
			}

			if len(mapOfExt) > 0 {
				ext := path.Ext(p)
				if len(ext) > 1 && strings.HasPrefix(ext, ".") {
					ext = ext[1:]
				}
				if !mapOfExt[ext] {
					return nil
				}
			}

			fileList = append(fileList, p)

			return nil
		})
	if err != nil {
		fmt.Println("ERR: failed to walk through files:", err)
		os.Exit(1)
	}
	return fileList
}

const (
	EvermintOg_GitHubRepo = "https://github.com/EscanBE/evermint"

	EvermintOg_ApplicationName       = "evermint"
	EvermintOg_ApplicationBinaryName = "evmd"
	EvermintOg_ApplicationHome       = ".evermint"

	EvermintOg_BaseDenom    = "wei"
	EvermintOg_DisplayDenom = "ether"
	EvermintOg_SymbolDenom  = "ETH"

	EvermintOg_Bech32Prefix            = "evm"
	EvermintOg_Bech32PrefixValAddr     = "evmvaloper"
	EvermintOg_Bech32PrefixValConsAddr = "evmvalcons"

	EvermintOg_MainnetChainID = "evermint_90909"
	EvermintOg_TestnetChainID = "evermint_80808"
	EvermintOg_DevnetChainID  = "evermint_70707"

	EvermintOg_MainnetFullChainId = constants.MainnetChainID + "-1"
	EvermintOg_TestnetFullChainId = constants.TestnetChainID + "-1"
	EvermintOg_DevnetFullChainId  = constants.DevnetChainID + "-1"

	EvermintOg_MainnetEIP155ChainId = 90909
	EvermintOg_TestnetEIP155ChainId = 80808
	EvermintOg_DevnetEIP155ChainId  = 70707
)
