package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func LoadFile(filename string, prefix string) error {
	envMap, err := readEnvFile(filename)
	if err != nil {
		return fmt.Errorf("reading environment file: %w", err)
	}

	for key, value := range envMap {
		realKey := prefix + "_" + key
		if _, ok := os.LookupEnv(realKey); !ok {
			os.Setenv(strings.ToUpper(realKey), value)
		}
	}

	return nil
}

func readEnvFile(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("opening environment file: %w", err)
	}
	defer file.Close()

	return godotenv.Parse(file)
}
