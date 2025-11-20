# fdu - Fast Disk Usage

[![Go Version](https://img.shields.io/github/go-mod/go-version/ajoyka/fdu)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A high-performance, concurrent disk usage analyzer written in Go. `fdu` rapidly traverses directory trees to analyze disk usage, identify duplicate files (especially images), and generate detailed metadata reports in JSON format.

## Features

- **Concurrent Directory Traversal**: Leverages goroutines for fast, parallel directory scanning
- **Duplicate File Detection**: Identifies duplicate files based on size and metadata, particularly useful for images
- **Image Metadata Extraction**: Extracts EXIF data from images for better organization
- **Multiple Output Formats**: Generates JSON reports sorted by date, size, and file information
- **SQLite Database Integration**: Stores file metadata and duplicate information in a SQLite database
- **Flexible Filtering**: Supports regex-based path exclusion patterns
- **Real-time Progress**: Optional periodic progress updates during scanning
- **Configurable Concurrency**: Adjustable parallelism to balance speed and resource usage

## Installation

### Prerequisites

- Go 1.22.5 or later

### Install from Source

```bash
go get github.com/ajoyka/fdu
cd $GOPATH/src/github.com/ajoyka/fdu
go build -o fdu ./fduapp
```

Or install directly:

```bash
go install github.com/ajoyka/fdu/fduapp@latest
```

## Usage

### Basic Usage

Analyze one or more directories:

```bash
./fdu /path/to/directory
```

Analyze multiple directories:

```bash
./fdu /path/to/dir1 /path/to/dir2 /path/to/dir3
```

### Command-Line Flags

- `-t <number>`: Number of top files/directories to display (default: 10)
- `-c <number>`: Concurrency factor - number of concurrent file operations (default: 20)
- `-s`: Print summary only, without detailed file listings
- `-e <pattern>`: Exclude files/directories matching the regex pattern (e.g., `-e '/a/b|/x/y'`)
- `-f <duration>`: Print progress summary at specified interval (e.g., `-f 5s` for every 5 seconds)

### Examples

Display top 20 largest files with concurrency of 50:

```bash
./fdu -t 20 -c 50 /home/user/Pictures
```

Show summary only, excluding specific paths:

```bash
./fdu -s -e '/tmp|/cache' /home/user
```

Monitor progress every 10 seconds:

```bash
./fdu -f 10s /large/directory
```

## Output Files

`fdu` generates several output files in the current directory:

- **`file-info.json`**: Comprehensive file metadata including size, modification time, and EXIF data
- **`date-info.json`**: File information sorted by modification date
- **`size-info.json`**: File information sorted by file size
- **`duplicates.json`**: List of potential duplicate files
- **SQLite database**: Contains structured file metadata and duplicate information

Existing output files are automatically backed up with a `.bak` extension before being overwritten.

## Use Cases

### Finding Duplicate Photos

Perfect for photographers and digital archivists managing large photo collections:

```bash
./fdu -t 50 ~/Pictures ~/Downloads ~/ExternalDrive/Photos
```

Review the `duplicates.json` file to identify and remove duplicate images.

### Disk Space Analysis

Quickly identify which directories consume the most space:

```bash
./fdu -t 30 /home
```

### Photo Organization

Extract EXIF metadata from images to help organize photos by date and camera:

```bash
./fdu ~/Pictures
```

The generated JSON files contain detailed EXIF information that can be used for sorting and organizing.

## Configuration

### Excluding Paths

By default, `fdu` skips thumbnail directories (e.g., `/Thumbs/`, `@eaDir`, `/rep/ssd/`). You can add your own exclusion patterns:

```bash
./fdu -e '/node_modules|/.git|/vendor' /path/to/scan
```

### Adjusting Concurrency

If you encounter "too many open files" errors, reduce the concurrency factor:

```bash
./fdu -c 10 /path/to/scan
```

For faster processing on systems with many cores and fast storage:

```bash
./fdu -c 100 /path/to/scan
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. 
