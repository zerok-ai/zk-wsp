package utils

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

func ProcessArgs(config interface{}) error {
	filePath := flag.String("c", "", "config file path")
	flag.Parse()

	if *filePath == "" {
		fmt.Println("Please provide a file path using the -c option.")
		return fmt.Errorf("file path provided is empty")
	}

	fileContents, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Printf("Failed to read file: %s\n", err)
		return err
	}

	err = yaml.Unmarshal(fileContents, config)
	if err != nil {
		fmt.Printf("Failed to unmarshal yaml: %s\n", err)
		return err
	}

	return nil
}
