# File Bundle Tool

Welcome to File Bundle – the versatile utility for developers looking to bundle multiple files into a single, neatly packaged file. Designed with simplicity and efficiency in mind, File Bundle streamlines the process of combining files for easier distribution, version control, or archiving purposes.

## Features

- **Configurable Through TOML**: Use the human-friendly TOML format to easily configure your file bundling preferences.
- **Flexible File Selection**: Specify exact file paths, patterns, or glob patterns for including files in your bundle.
- **Exclusion Patterns**: Exclude files from your bundle using flexible glob patterns to avoid bundling unnecessary or sensitive files.
- **Shrink Mode**: Reduce the size of the bundle by trimming unnecessary whitespace from your files.
- **Verbose Mode**: Enable detailed logging for more insight into the bundling process.
- **Quick Initialization**: Get started instantly with a generated default configuration file using a simple command.

## Getting Started

To get started with File Bundle, you'll need to have a `.file_bundle_rc` configuration file in TOML format in your project directory. This file specifies which files should be included or excluded from the bundle, as well as the name of the output bundle file.

### Example .file_bundle_rc

```toml
entry = [
    "src/**/*.go",
    "resources/**/*",
    "configs/*.json"
]
exclude = [
    "vendor/**",
    "*.test",
    "docs/**/*.md"
]
output = "my_project.bundle"
```

This configuration would result in bundling all Go source files from src, resources, and all JSON configuration files, while excluding vendor files, test files, and Markdown files inside the docs directory.


#### All Parameters

- entry: An array of strings that specify file paths or glob patterns for files to include in the bundle. 
  - Example: entry = ["*.go", "assets/**/*"]
- exclude: An array of strings that defines patterns for files to exclude from the bundle.
  - Example: exclude = ["tmp/*", "*.tmp", "tests/*"]
- output: A string that names the output bundle file. This is the final file where all the specified files will be bundled. 
  - Example: output = "bundle_project_v1.bundle"
- description

### Installation

To install File Bundle, use the following go get command:

```sh
go get github.com/bagaking/file_bundle
```


### Usage

Creating a file bundle is as easy as running the following command:

```sh
file_bundle -i .file_bundle_rc -o my_project.bundle
```

For more information on command-line options, use the -h flag:

```sh
file_bundle -h
```

#### Quick Initialization Command

Don't have a .file_bundle_rc file? No problem!

Run the following command to generate a default config in your current directory:

```bash
file_bundle -touch 
```

#### other commands

- shrink: A boolean to indicate whether to engage shrink mode to eliminate unnecessary whitespaces.
    - Example: `file_bundle -s`

- verbose: A boolean that enables verbose logging if set to true. This will provide additional output logs during the bundling process.
    - Example: `file_bundle -v`

## Contribution

We welcome contributions of all kinds: feature requests, bug reports, or pull requests. Please ensure to read through the contributing guidelines first.

## License
File Bundle is released under the MIT License. Enjoy the relief of easily bundling files without the hassle of complex setup or configuration.
