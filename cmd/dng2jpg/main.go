package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"dng2jpg/internal/libraw"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		outPath      string
		format       string
		quality      int
		wb           string
		wbMul        string
		exposure     float64
		brightness   float64
		highlight    int
		colorspace   string
		depth        int
		denoise      float64
		median       int
		autoBright   bool
		halfSize     bool
		thumbnail    bool
		noAutoOrient bool
		logFile      string
	)

	flag.StringVar(&outPath, "o", "", "output path (file for single, directory for batch)")
	flag.StringVar(&format, "format", "jpeg", "output format: jpeg | png")
	flag.IntVar(&quality, "quality", 95, "JPEG quality 1-100")
	flag.StringVar(&wb, "wb", "auto", "white balance: auto | camera | custom")
	flag.StringVar(&wbMul, "wb-mul", "", "custom WB: R,G1,B,G2")
	flag.Float64Var(&exposure, "exposure", 0.0, "exposure compensation EV")
	flag.Float64Var(&brightness, "brightness", 1.0, "brightness multiplier")
	flag.IntVar(&highlight, "highlight", 0, "highlight: 0=clip 1=unclip 2=blend 3=rebuild")
	flag.StringVar(&colorspace, "colorspace", "sRGB", "color space: sRGB | AdobeRGB | ProPhoto | WideGamut")
	flag.IntVar(&depth, "depth", 8, "bit depth: 8 | 16")
	flag.Float64Var(&denoise, "denoise", 0, "wavelet denoise threshold")
	flag.IntVar(&median, "median", 0, "median filter passes")
	flag.BoolVar(&autoBright, "auto-brightness", false, "auto brightness")
	flag.BoolVar(&halfSize, "half-size", false, "half resolution")
	flag.BoolVar(&thumbnail, "thumbnail", false, "use embedded thumbnail")
	flag.BoolVar(&noAutoOrient, "no-auto-orient", false, "disable auto rotation")
	flag.StringVar(&logFile, "log", "", "summary log path")
	flag.Parse()

	input := flag.Arg(0)
	if input == "" {
		fmt.Fprintf(os.Stderr, "Usage: dng2jpg INPUT [-o OUTPUT] [flags]\n")
		flag.PrintDefaults()
		return 1
	}

	opts := libraw.DefaultOptions()

	switch format {
	case "jpeg":
		opts.Format = libraw.FormatJPEG
	case "png":
		opts.Format = libraw.FormatPNG
	default:
		fmt.Fprintf(os.Stderr, "invalid format: %s\n", format)
		return 1
	}
	opts.JPEGQuality = quality

	switch wb {
	case "auto":
		opts.WhiteBalance = libraw.WBAuto
	case "camera":
		opts.WhiteBalance = libraw.WBCamera
	case "custom":
		opts.WhiteBalance = libraw.WBCustom
		if wbMul != "" {
			if _, err := fmt.Sscanf(wbMul, "%f,%f,%f,%f",
				&opts.CustomWBMul[0], &opts.CustomWBMul[1],
				&opts.CustomWBMul[2], &opts.CustomWBMul[3]); err != nil {
				fmt.Fprintf(os.Stderr, "invalid wb-mul: %s\n", wbMul)
				return 1
			}
		}
	default:
		fmt.Fprintf(os.Stderr, "invalid wb: %s\n", wb)
		return 1
	}

	opts.ExposureComp = exposure
	opts.Brightness = brightness
	opts.HighlightMode = highlight

	switch colorspace {
	case "sRGB":
		opts.ColorSpace = libraw.ColorSpaceSRGB
	case "AdobeRGB":
		opts.ColorSpace = libraw.ColorSpaceAdobeRGB
	case "ProPhoto":
		opts.ColorSpace = libraw.ColorSpaceProPhoto
	case "WideGamut":
		opts.ColorSpace = libraw.ColorSpaceWideGamut
	default:
		fmt.Fprintf(os.Stderr, "invalid colorspace: %s\n", colorspace)
		return 1
	}

	opts.OutputDepth = depth
	opts.DenoiseThreshold = denoise
	opts.MedianFilterPasses = median
	opts.AutoBrightness = autoBright
	opts.HalfSize = halfSize
	opts.UseThumbnail = thumbnail
	opts.NoAutoOrient = noAutoOrient

	if err := opts.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid options: %v\n", err)
		return 1
	}

	// --- Discover files ---
	var files []string
	fi, err := os.Stat(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "input not found: %v\n", err)
		return 1
	}
	if fi.IsDir() {
		entries, err := os.ReadDir(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read dir: %v\n", err)
			return 1
		}
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".dng" {
				files = append(files, filepath.Join(input, e.Name()))
			}
		}
		if len(files) == 0 {
			fmt.Fprintf(os.Stderr, "no .dng files in %s\n", input)
			return 1
		}
		if outPath == "" {
			outPath = input
		}
	} else {
		files = []string{input}
		if outPath == "" {
			outPath = deriveOutput(input, opts.Format.Ext())
		}
	}

	// --- Ensure output directory exists ---
	if err := os.MkdirAll(outPath, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating output dir: %v\n", err)
		return 1
	}

	startTime := time.Now()

	// --- Sequential conversion ---
	success, failed := 0, 0
	var failures []libraw.ConvertResult

	for i, f := range files {
		outFile := outPath
		if fi, _ := os.Stat(outPath); fi != nil && fi.IsDir() {
			base := filepath.Base(f)
			name := base[:len(base)-len(filepath.Ext(base))]
			outFile = filepath.Join(outPath, name+opts.Format.Ext())
		}
		outFile = uniquePath(outFile)

		t0 := time.Now()
		err := libraw.Convert(f, outFile, opts)
		elapsed := time.Since(t0)

		r := libraw.ConvertResult{
			InputPath:  f,
			OutputPath: outFile,
			Duration:   elapsed,
			Err:        err,
		}

		if err != nil {
			failed++
			failures = append(failures, r)
			fmt.Fprintf(os.Stderr, "[%d/%d] FAIL %s: %v\n", i+1, len(files), filepath.Base(f), err)
		} else {
			success++
			fmt.Printf("[%d/%d] %s -> %s (%s)\n", i+1, len(files), filepath.Base(f), filepath.Base(outFile), elapsed.Round(time.Millisecond))
		}
	}

	totalTime := time.Since(startTime)

	// --- Summary ---
	fmt.Printf("\nDone: %d ok, %d failed / %d total in %s\n", success, failed, len(files), totalTime.Round(time.Millisecond))

	logPath := logFile
	if logPath == "" {
		logPath = filepath.Join(outPath, "dng2jpg-"+time.Now().Format("2006-01-02-150405")+".log")
	}
	writeLog(logPath, input, outPath, opts, success, failed, len(files), totalTime, failures)

	if failed > 0 {
		fmt.Fprintf(os.Stderr, "See %s for details\n", logPath)
	}
	if failed == len(files) {
		return 1
	}
	return 0
}

func deriveOutput(input, ext string) string {
	base := filepath.Base(input)
	return base[:len(base)-len(filepath.Ext(base))] + ext
}

func uniquePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	base := path[:len(path)-len(ext)]
	for i := 1; i < 1000; i++ {
		c := fmt.Sprintf("%s_%d%s", base, i, ext)
		if _, err := os.Stat(c); os.IsNotExist(err) {
			return c
		}
	}
	return path
}

func writeLog(logPath, inputDir, outputDir string, opts *libraw.Options, success, failed, total int, elapsed time.Duration, failures []libraw.ConvertResult) {
	f, err := os.Create(logPath)
	if err != nil {
		log.Printf("failed to create log file: %v", err)
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "dng2jpg report — %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(f, "=====================================\n")
	fmt.Fprintf(f, "Input : %s\n", inputDir)
	fmt.Fprintf(f, "Output: %s\n", outputDir)
	fmt.Fprintf(f, "Format: %s  quality=%d  wb=%s\n", opts.Format, opts.JPEGQuality, opts.WhiteBalance)
	fmt.Fprintf(f, "\nResults:\n")
	fmt.Fprintf(f, "  Total  : %d\n", total)
	fmt.Fprintf(f, "  OK     : %d\n", success)
	fmt.Fprintf(f, "  Failed : %d\n", failed)
	fmt.Fprintf(f, "  Time   : %s\n", elapsed.Round(time.Millisecond))
	if total > 0 {
		fmt.Fprintf(f, "  Avg    : %s\n", (elapsed / time.Duration(total)).Round(time.Millisecond))
	}
	if len(failures) > 0 {
		fmt.Fprintf(f, "\nFailed:\n")
		for _, r := range failures {
			fmt.Fprintf(f, "  [%s] %v\n", r.InputPath, r.Err)
		}
	}
}
