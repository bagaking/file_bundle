package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const defaultName = "bundle.bundle"
const defaultConfigExt = ".file_bundle_rc"

var (
	input   string
	output  string
	shrink  bool
	verbose bool

	touchCMD bool
)

func init() {
	// 增加 init 参数用于确定是否初始化配置文件
	flag.BoolVar(&touchCMD, "touch", false, "initialize a default .file_bundle_rc")

	flag.StringVar(&input, "i", "", "input .file_bundle_rc file name(s)")
	flag.StringVar(&output, "o", "", "output file name")
	flag.BoolVar(&shrink, "s", false, "shrink mode: trim unnecessary white space")
	flag.BoolVar(&verbose, "v", false, "verbose mode")
}

type Config struct {
	Entry       []string `toml:"entry"`
	Exclude     []string `toml:"exclude"`
	Output      string   `toml:"output"`
	Description string   `toml:"description"`
}

func touch() {
	// 如果用户执行 file_bundle init，则创建一个默认的配置文件并退出程序

	defaultConf := Config{
		Entry:   []string{"./*"},
		Exclude: []string{".bundle"},
		Output:  defaultName,
	}

	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).Encode(defaultConf)
	if err != nil {
		fmt.Printf("Could not encode default config: %v\n", err)
		os.Exit(1)
	}

	fileName := "_" + defaultConfigExt
	err = os.WriteFile(fileName, buf.Bytes(), 0644)
	if err != nil {
		fmt.Printf("Could not write default config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s created successfully.\n", fileName)
}

func initConf() Config {
	flag.Parse()
	if touchCMD {
		touch()
		os.Exit(0)
	}

	configFileName := input
	var err error
	// Use config file provided by -i option if present
	if input != "" {
		if _, err = os.Stat(configFileName); err != nil {
			configFileName += defaultConfigExt
			_, err = os.Stat(configFileName)
		}
	}

	// Read default config file if -i option was not provided or the specified file does not exist
	if configFileName == "" || err != nil {
		configFileName, err = seekConfFileName()
	}

	if err != nil {
		fmt.Printf("Could not get config file, make sure it exists in the current directory.\nerror= %v\n", err)
		printHelp()
		os.Exit(1)
	}

	content, err := ioutil.ReadFile(configFileName)
	if err != nil {
		fmt.Printf("Could not read config file. Make sure it is present in the current directory.\nerror= %v\n", err)
		printHelp()
		os.Exit(1)
	}

	var config Config
	_, err = toml.Decode(string(content), &config)
	if err != nil {
		fmt.Printf("Invalid config file format. It should be a valid TOML.\nerror= %v\n", err)
		printHelp()
		os.Exit(1)
	}

	// If the -o flag provided and not empty, use its value to override config.Output
	if output != "" {
		config.Output = output
	}

	// If config.Output is still empty, set the default value
	if config.Output == "" {
		config.Output = defaultName
	}

	return config
}

func seekConfFileName() (string, error) {
	var configFile string

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() && path != "." {
			return filepath.SkipDir
		}

		if strings.HasSuffix(path, ".file_bundle_rc") {
			configFile = path
			return errors.New("found")
		}

		return nil
	})

	if err == nil {
		return "", fmt.Errorf("config file not found")
	}

	return configFile, nil
}
