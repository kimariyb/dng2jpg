# dng2jpg

Convert DNG raw camera files to high-quality JPEG or PNG using **libraw** — the same engine that powers dcraw.

## Table of Contents

- [Tech Stack](#tech-stack)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Usage](#usage)
- [Flags Reference](#flags-reference)
- [How It Works](#how-it-works)
- [Project Structure](#project-structure)
- [Building from Source](#building-from-source)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [License](#license)

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.22+ |
| RAW engine | libraw (C, compiled via CGo) |
| Image encoding | Go stdlib (`image/jpeg`, `image/png`) |
| Build system | GNU Make + CGo |

No external Go dependencies at runtime. libraw is vendored as a git submodule and compiled statically.

---

## Prerequisites

- **Go 1.22** or later
- **GCC** or **Clang** (for CGo — included with Xcode on macOS, or `build-essential` on Linux)
- **Make** (GNU Make)

macOS:
```bash
# Xcode Command Line Tools (includes clang)
xcode-select --install
```

Linux:
```bash
sudo apt-get install build-essential
```

---

## Quick Start

```bash
git clone <repo-url> dng2jpg
cd dng2jpg

# One-time: download and build libraw (~30s)
make setup

# Compile dng2jpg
make build

# Convert a single file
./bin/dng2jpg photo.dng

# Convert a directory
./bin/dng2jpg ./dng -o ./output
```

The `make setup` step clones libraw as a git submodule and compiles it into a static library (`libraw/LibRaw/lib/libraw.a`). You only need to run this once (or after `git submodule update`).

---

## Usage

```
dng2jpg INPUT [-o OUTPUT] [flags]
```

- `INPUT` — a single `.dng` file or a directory containing `.dng` files
- `-o OUTPUT` — output file (single mode) or output directory (batch mode). If omitted, defaults to the current directory.

### Examples

```bash
# Single file — outputs photo.jpg in current directory
./bin/dng2jpg photo.dng

# Specify output path
./bin/dng2jpg photo.dng -o result.jpg

# Convert directory — all .dng → .jpg into out/
./bin/dng2jpg dng/ -o out/

# PNG output
./bin/dng2jpg photo.dng -o result.png --format png

# Extract camera embedded thumbnail (instant, bypasses RAW development)
./bin/dng2jpg photo.dng -o thumb.jpg --thumbnail

# Camera white balance + denoise + exposure adjustment
./bin/dng2jpg dng/ -o out/ --wb camera --denoise 100 --exposure -0.5

# Adobe RGB color space at 16-bit depth
./bin/dng2jpg photo.dng -o result.jpg --colorspace AdobeRGB --depth 16
```

### Batch output

When converting a directory, each file produces one output file named `<original-basename>.jpg` (or `.png`). Duplicate names get `_1`, `_2` suffixes appended — never overwritten.

A summary log file is written to the output directory (`dng2jpg-<timestamp>.log`) with per-file timing and failure details.

```
[1/54] photo1.dng -> photo1.jpg (1.8s)
[2/54] photo2.dng -> photo2.jpg (2.1s)
[3/54] FAIL bad.dng: unsupported format
...
Done: 53 ok, 1 failed / 54 total in 3m42s
See out/dng2jpg-2026-06-12-131225.log for details
```

---

## Flags Reference

### Output

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-o` | string | auto | Output file path (single mode) or directory (batch mode) |
| `--format` | string | `jpeg` | `jpeg` or `png` |
| `--quality` | int | `95` | JPEG quality 1–100 (ignored for PNG) |

### White Balance

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--wb` | string | `auto` | `auto` — automatic; `camera` — as shot; `custom` — manual coefficients |
| `--wb-mul` | string | — | Custom multipliers `R,G1,B,G2` (requires `--wb custom`) |

### Exposure

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--exposure` | float | `0` | Exposure compensation in EV (whole stops only; libraw uses int internally) |
| `--brightness` | float | `1.0` | Brightness multiplier (> 0) |
| `--highlight` | int | `0` | `0`=clip, `1`=unclip, `2`=blend, `3`=rebuild |
| `--auto-brightness` | bool | false | Apply auto brightness correction |

### Color

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--colorspace` | string | `sRGB` | `sRGB`, `AdobeRGB`, `ProPhoto`, `WideGamut` |
| `--depth` | int | `8` | Bit depth: `8` or `16` |

### Noise Reduction & Sharpening

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--denoise` | float | `0` | Wavelet denoise threshold (0 = off) |
| `--median` | int | `0` | Median filter passes (0 = off) |

### Misc

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--half-size` | bool | false | Output at half resolution |
| `--thumbnail` | bool | false | Extract embedded camera JPEG (fast, no RAW development) |
| `--no-auto-orient` | bool | false | Disable automatic rotation based on camera orientation |
| `--log` | string | auto | Summary log file path |

---

## How It Works

```
Input DNG
    │
    ▼
┌─────────────────────────────┐
│  libraw (CGo)               │
│  libraw_init → open_file    │
│  → unpack → apply_params   │
│  → dcraw_process            │
│  → dcraw_ppm_tiff_writer   │
│  → temp .ppm file           │
└─────────────────────────────┘
    │
    ▼
┌─────────────────────────────┐
│  Go PPM decoder             │
│  Parse P6 binary header     │
│  → image.RGBA               │
└─────────────────────────────┘
    │
    ▼
┌─────────────────────────────┐
│  Go encoder                  │
│  image/jpeg  or  image/png  │
│  → output .jpg / .png       │
└─────────────────────────────┘
```

**Why PPM?** libraw's PPM writer produces pixel-identical output to the `dcraw` command-line tool. By routing through a PPM intermediate file (automatically deleted), we get guaranteed-correct RAW decoding without depending on a specific libraw bitmap memory layout. Go's standard library handles JPEG/PNG encoding reliably.

**Auto-brightness** is always enabled internally (`auto_bright_thr = 0.01`) because `libraw_init` zeroes the params struct, which would otherwise disable it.

---

## Project Structure

```
dng2jpg/
├── cmd/dng2jpg/
│   ├── main.go              # CLI: flag parsing, file discovery, sequential loop, log
│   └── main_test.go         # Path derivation & unique-path unit tests
├── internal/libraw/
│   ├── converter.go          # CGo wrapper: libraw pipeline + PPM decoding + JPEG/PNG encoding
│   ├── params.go             # Options, enums (OutputFormat, WBMode, ColorSpace), validation
│   ├── converter_test.go     # End-to-end conversion integration tests
│   └── params_test.go        # Options default, validation, and String() unit tests
├── libraw/
│   └── LibRaw/               # libraw C source (git submodule)
├── go.mod / go.sum           # Go module (zero external deps at runtime)
├── Makefile                  # setup, build, test, clean
└── README.md
```

### Key packages

**`internal/libraw`** — CGo boundary. Exposes `Convert(inPath, outPath string, opts *Options) error`. Handles all libraw interaction: init, open, unpack, param mapping, dcraw_process, PPM write-out, PPM decode, JPEG/PNG encode. The caller never touches CGo.

**`cmd/dng2jpg`** — Pure Go CLI. Flag parsing with `flag`, file discovery (single `.dng` or directory scan), sequential conversion loop with per-file output, summary log generation.

---

## Building from Source

### 1. Clone

```bash
git clone <repo-url>
cd dng2jpg
```

### 2. Build libraw

```bash
make setup
```

This runs:
```makefile
git submodule add https://github.com/LibRaw/LibRaw.git libraw/LibRaw
git submodule update --init --recursive
cd libraw/LibRaw && make -f Makefile.devel -j$(nproc) library
```

The resulting static library is at `libraw/LibRaw/lib/libraw.a` (~11 MB).

### 3. Build dng2jpg

```bash
make build
# → bin/dng2jpg  (~5.8 MB, statically linked with libraw)
```

The binary is self-contained — no shared library dependencies beyond `libstdc++` and system libraries.

### 4. Verify

```bash
./bin/dng2jpg --help
```

---

## Testing

```bash
# All tests (unit + integration)
make test

# Or with verbose output
go test ./... -v -count=1

# Specific packages
go test ./internal/libraw/ -v       # Type tests + real DNG conversion
go test ./cmd/dng2jpg/ -v           # Path derivation & helpers
```

### What's tested

| Package | Tests | Coverage |
|---------|-------|----------|
| `internal/libraw` | 16 param tests + 3 conversion tests | Types, validation, end-to-end DNG→JPEG |
| `cmd/dng2jpg` | 6 helper tests | Path derivation, unique-path conflict resolution |

The conversion tests use real `.dng` files from the `dng/` directory. If no `.dng` files are found, conversion tests skip automatically.

---

## Troubleshooting

### `make setup` fails — `autoreconf` or `make` not found

On macOS, install Xcode Command Line Tools:
```bash
xcode-select --install
```

On Linux:
```bash
sudo apt-get install build-essential
```

### `go build` fails — CGo linker errors

Ensure libraw is built first:
```bash
make setup
ls libraw/LibRaw/lib/libraw.a   # should exist
```

### "no .dng files found"

The tool only processes `.dng` files. Ensure your input directory contains `.dng` files (case-sensitive).

### Conversion succeeds but image is dark

Try adjusting exposure or highlight mode:
```bash
./bin/dng2jpg photo.dng -o out.jpg --highlight 0 --exposure 0.5
```

### Flags must come before the input path

Go's `flag` package stops parsing at the first non-flag argument:
```bash
# Correct
./bin/dng2jpg -o out/ --format png dng/

# Wrong — --format is ignored
./bin/dng2jpg dng/ --format png
```

---

## License

MIT
