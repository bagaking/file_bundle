# File Bundle

File Bundle is a Go CLI and source file bundling utility. It reads a TOML
configuration file, expands file and glob entries, excludes matching paths, and
writes the selected file contents into one bundle file.

## Features

- Configure bundle inputs with TOML.
- Include files by exact path or glob pattern.
- Exclude paths by glob pattern.
- Override the output file from the command line.
- Use shrink mode to trim unnecessary whitespace.
- Use verbose mode to print bundling details.
- Generate a starter configuration with the touch command.

## Installation

```sh
go install github.com/bagaking/file_bundle@latest
```

## Configuration

Create a `.file_bundle_rc` TOML file in the project directory.

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
description = "optional bundle description"
```

Parameters:

- `entry`: file paths or glob patterns to include.
- `exclude`: glob patterns to exclude.
- `output`: bundle output path.
- `description`: optional text written into bundle file headers.

## Usage

Bundle files with a specific config and output path:

```sh
file_bundle -i .file_bundle_rc -o my_project.bundle
```

Show command-line help:

```sh
file_bundle -h
```

Generate a default config in the current directory:

```sh
file_bundle -touch
```

Create a `bundle/` directory with a config file and Makefile:

```sh
file_bundle -touch dir
```

Run shrink mode:

```sh
file_bundle -s
```

Run verbose mode:

```sh
file_bundle -v
```

## Behavior Notes

- Config paths are resolved from the current working directory.
- If `-i` is omitted, `file_bundle` uses the first `*.file_bundle_rc` file in
  the current directory only.
- `entry` and `exclude` use doublestar glob patterns, so `**` can match nested
  directories.
- The output file is skipped when it also matches an entry pattern, which avoids
  recursively bundling a previous run's output.
- `bundle/Makefile` is a generated template for projects that run
  `file_bundle -touch dir`; the repository-level validation entry point is the
  root `Makefile`.

## Local Validation

Run the test suite before submitting changes:

```sh
go test ./...
```

Or use the repository validation target:

```sh
make check
```

## License

File Bundle is released under the MIT License. See [LICENSE](LICENSE).
