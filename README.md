# Usage

This program reads the first '*.file_bundle_rc' file from the current directory
and creates 'bundle.bundle'.

The '.file_bundle_rc' file is in TOML format and has a structure as shown below:

entry = ["file1", "dir1/*", "path/to/files/*.txt", "*.go", "file2", "dir2/*", "./*"]
exclude = ["src/experimental/*", "*.md", "static/*.min.js"]

In the above structure, 'file1', 'file2', etc., are the paths of the files
that you want to bundle. The paths should be relative to the current directory.

When this program runs, it will read each file in the 'entry' list,
and append the contents of these files into a single 'bundle.bundle' file.
Each file appended to 'bundle.bundle' file will be preceded by a separator
line and the original path of the file.
