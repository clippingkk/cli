package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path"

	"github.com/clippingkk/cli/core"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var inputPath string
var outputPath string

var rootCmd = &cobra.Command{
	Use:   "ck-cli",
	Short: "clippingkk cli to parse and sync your `My Clippings.txt` in cli",
	Long:  `clippingkk cli to parse and sync your "My Clippings.txt" in cli`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fileData, err := getClippingsText()
		if err != nil {
			return err
		}

		if len(fileData) == 0 {
			return errors.New("error")
		}

		parser := core.NewClippingParser(fileData)
		if err := parser.Prepare(); err != nil {
			return err
		}

		result, err := parser.DoParse()
		if err != nil {
			return err
		}

		resultBuf, err := json.MarshalIndent(result, "", "\t")
		if err != nil {
			return err
		}

		if outputPath != "" {
			if err := saveDataToLocal(resultBuf); err != nil {
				return err
			}
		} else {
			_, err := os.Stdout.Write(resultBuf)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

func saveDataToLocal(data []byte) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	distPath := path.Join(wd, outputPath)
	return os.WriteFile(distPath, data, 0777)
}

func getClippingsText() (fileData []byte, err error) {
	if inputPath == "" {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			fileData = append(fileData, scanner.Bytes()...)
			fileData = append(fileData, []byte("\n")...)
		}
		err := scanner.Err()
		return fileData, err
	}

	wd, err := os.Getwd()
	if err != nil {
		return fileData, err
	}
	distPath := path.Join(wd, inputPath)

	fd, err := os.ReadFile(distPath)
	if err != nil {
		return fileData, err
	}
	fileData = fd
	return
}

func Execute() {
	rootCmd.PersistentFlags().StringVar(&inputPath, "i", "", "`My Clippings.txt` file path")
	rootCmd.PersistentFlags().StringVar(&outputPath, "o", "", "output filepath")
	if err := rootCmd.Execute(); err != nil {
		logrus.Errorln(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
