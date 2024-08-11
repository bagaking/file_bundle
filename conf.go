package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/khicago/irr"

	"github.com/BurntSushi/toml"
)

const (
	defaultName      = "bundle.bundle"
	defaultConfigExt = ".file_bundle_rc"
)

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

// 如果用户执行 file_bundle -touch，则创建一个默认的配置文件或目录，然后退出
func touch() {
	defaultConf := Config{
		Entry:   []string{"./*"},
		Exclude: []string{".bundle", ".bundle.txt"},
	}
	var buf bytes.Buffer
	// 检查是否需要创建目录
	if flag.Arg(0) == "dir" {
		err := toml.NewEncoder(&buf).Encode(defaultConf)
		if err != nil {
			exitWith("Could not encode default config: %v", err)
		}
		if err = createConfDir(buf.Bytes()); err != nil {
			exitWith("Could not create directory: %v", err)
		}
		exitWith("bundle directory with .file_bundle_rc and Makefile created successfully.")
	} else {
		defaultConf.Output = defaultName
		err := toml.NewEncoder(&buf).Encode(defaultConf)
		if err != nil {
			exitWith("Could not encode default config: %v\n", err)
		}
		fileName := "_" + defaultConfigExt
		err = os.WriteFile(fileName, buf.Bytes(), 0o644)
		if err != nil {
			exitWith("Could not write default config file: %v\n", err)
		}
		exitWith("%s created successfully.", fileName)
	}
}

func createConfDir(buf []byte) error {
	err := os.Mkdir("bundle", 0o755)
	if err != nil {
		return irr.Wrap(err, "could not create directory")
	}

	err = os.WriteFile("bundle/_all"+defaultConfigExt, buf, 0o644)
	if err != nil {
		return irr.Wrap(err, "could not write config file")
	}

	makefileContent := `
# Makefile of file_bundle

# hint: To embed this makefile as a sub-cmd into the room Makefile
#
# bundle:
# 	$(MAKE) -C bundle -f Makefile clean
#	$(MAKE) -f bundle/Makefile

.PHONY: all bundle clean

# 获取当前目录下所有 file_bundle_rc 文件列表
FILE_BUNDLE_RCS := $(shell find . -name '*.file_bundle_rc')

# 定义后缀替换规则：将 .file_bundle_rc 结尾的文件替换为 .bundle.txt
BUNDLES := $(FILE_BUNDLE_RCS:.file_bundle_rc=.bundle.txt)

# 默认目标
all: clean bundle

# 打包
bundle: $(BUNDLES)

# 规则来生成 bundle 文件
%.bundle.txt: %.file_bundle_rc
	file_bundle -v -i $< -o $@

# 清理
clean:
	rm -f $(BUNDLES)
`

	if err = os.WriteFile("bundle/Makefile", []byte(makefileContent), 0o644); err != nil {
		return irr.Wrap(err, "could not write makefile")
	}

	return nil
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
		exitWith("Could not get config file, make sure it exists in the current directory.\nerror= %v\n", err)
		printHelp()
	}

	content, err := os.ReadFile(configFileName)
	if err != nil {
		exitWith("Could not read config file. Make sure it is present in the current directory.\nerror= %v\n", err)
		printHelp()
	}

	var config Config
	_, err = toml.Decode(string(content), &config)
	if err != nil {
		exitWith("Invalid config file format. It should be a valid TOML.\nerror= %v\n", err)
		printHelp()
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
		return "", irr.Error("config file not found")
	}

	return configFile, nil
}

func exitWith(format string, args ...any) {
	f := strings.TrimSpace(format)
	if !strings.HasSuffix(f, "\n") {
		f += "\n"
	}
	hasError := false
	for _, a := range args {
		if _, ok := a.(error); ok {
			hasError = true
		}
	}
	if hasError {
		f = "Error: " + f
	}
	fmt.Printf(f, args...)
	os.Exit(1)
}
