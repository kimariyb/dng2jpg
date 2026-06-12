package libraw

/*
#cgo CFLAGS: -I${SRCDIR}/../../libraw/LibRaw
#cgo LDFLAGS: -L${SRCDIR}/../../libraw/LibRaw/lib -lraw -lstdc++ -lm -lz
#include "libraw/libraw.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"unsafe"
)

// Convert transforms a DNG raw file into a JPEG or PNG image.
func Convert(inPath, outPath string, opts *Options) error {
	if opts == nil {
		opts = DefaultOptions()
	}
	if err := opts.Validate(); err != nil {
		return fmt.Errorf("convert: %w", err)
	}

	raw := C.libraw_init(0)
	if raw == nil {
		return fmt.Errorf("libraw_init failed")
	}
	defer C.libraw_close(raw)
	defer C.libraw_recycle(raw)

	cIn := C.CString(inPath)
	defer C.free(unsafe.Pointer(cIn))
	if e := C.libraw_open_file(raw, cIn); e != C.LIBRAW_SUCCESS {
		return fmt.Errorf("open: %s", C.GoString(C.libraw_strerror(e)))
	}

	if opts.UseThumbnail {
		return extractThumbnail(raw, outPath)
	}

	if e := C.libraw_unpack(raw); e != C.LIBRAW_SUCCESS {
		return fmt.Errorf("unpack: %s", C.GoString(C.libraw_strerror(e)))
	}

	applyParams(&raw.params, opts)

	if e := C.libraw_dcraw_process(raw); e != C.LIBRAW_SUCCESS {
		return fmt.Errorf("process: %s", C.GoString(C.libraw_strerror(e)))
	}

	// Write PPM via libraw (reference-quality output, identical to dcraw CLI).
	ppm, err := os.CreateTemp("", "dng2jpg-*.ppm")
	if err != nil {
		return fmt.Errorf("temp: %w", err)
	}
	ppmPath := ppm.Name()
	ppm.Close()
	defer os.Remove(ppmPath)

	cOut := C.CString(ppmPath)
	defer C.free(unsafe.Pointer(cOut))
	if e := C.libraw_dcraw_ppm_tiff_writer(raw, cOut); e != C.LIBRAW_SUCCESS {
		return fmt.Errorf("ppm write: %s", C.GoString(C.libraw_strerror(e)))
	}

	img, err := decodePPM(ppmPath)
	if err != nil {
		return err
	}

	return encodeImage(outPath, img, opts)
}

// --- thumbnail ---------------------------------------------------------------

func extractThumbnail(raw *C.libraw_data_t, outPath string) error {
	if e := C.libraw_unpack_thumb(raw); e != C.LIBRAW_SUCCESS {
		return fmt.Errorf("unpack_thumb: %s", C.GoString(C.libraw_strerror(e)))
	}
	var ec C.int
	thumb := C.libraw_dcraw_make_mem_thumb(raw, &ec)
	if thumb == nil {
		return fmt.Errorf("mem_thumb: %s", C.GoString(C.libraw_strerror(ec)))
	}
	defer C.libraw_dcraw_clear_mem(thumb)
	if thumb._type != C.LIBRAW_IMAGE_JPEG {
		return fmt.Errorf("thumbnail is not JPEG (type=%d)", thumb._type)
	}
	data := C.GoBytes(unsafe.Pointer(&thumb.data[0]), C.int(thumb.data_size))
	return os.WriteFile(outPath, data, 0644)
}

// --- params ------------------------------------------------------------------

func applyParams(p *C.libraw_output_params_t, opts *Options) {
	switch opts.WhiteBalance {
	case WBAuto:
		p.use_auto_wb = 1
	case WBCamera:
		p.use_camera_wb = 1
	case WBCustom:
		p.user_mul[0] = C.float(opts.CustomWBMul[0])
		p.user_mul[1] = C.float(opts.CustomWBMul[1])
		p.user_mul[2] = C.float(opts.CustomWBMul[2])
		p.user_mul[3] = C.float(opts.CustomWBMul[3])
	}

	p.exp_correc = C.int(opts.ExposureComp)
	p.exp_shift = C.float(1.0)
	if opts.Brightness != 1.0 {
		p.bright = C.float(opts.Brightness)
	}
	p.highlight = C.int(opts.HighlightMode)

	switch opts.ColorSpace {
	case ColorSpaceSRGB:
		p.output_color = 1
	case ColorSpaceAdobeRGB:
		p.output_color = 2
	case ColorSpaceProPhoto:
		p.output_color = 4
	case ColorSpaceWideGamut:
		p.output_color = 5
	}

	if opts.OutputDepth == 16 {
		p.output_bps = 16
	} else {
		p.output_bps = 8
	}

	p.fbdd_noiserd = C.int(opts.DenoiseThreshold)
	if opts.MedianFilterPasses > 0 {
		p.med_passes = C.int(opts.MedianFilterPasses)
	}

	// libraw_init zeroes params; auto_bright_thr=0 disables auto-brightness.
	p.auto_bright_thr = C.float(0.01)

	if opts.HalfSize {
		p.half_size = 1
	}
	if opts.NoAutoOrient {
		p.use_camera_matrix = 3
	}
}

// --- PPM decoder -------------------------------------------------------------
// Parses binary PPM (P6):  header "P6\nW H\nMAXVAL\n" then raw RGB bytes.

func decodePPM(path string) (image.Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read ppm: %w", err)
	}

	// Find header end: three newlines.
	hdrEnd := 0
	nl := 0
	for hdrEnd < len(data) && nl < 3 {
		if data[hdrEnd] == '\n' {
			nl++
		}
		hdrEnd++
	}
	if nl < 3 {
		return nil, fmt.Errorf("ppm: malformed header")
	}

	var w, h, maxval int
	if _, err := fmt.Sscanf(string(data[:hdrEnd]), "P6\n%d %d\n%d", &w, &h, &maxval); err != nil {
		return nil, fmt.Errorf("ppm: parse header: %w", err)
	}
	if w <= 0 || h <= 0 || maxval <= 0 {
		return nil, fmt.Errorf("ppm: bad dims %dx%d max=%d", w, h, maxval)
	}

	byteCount := w * h * 3
	if hdrEnd+byteCount > len(data) {
		return nil, fmt.Errorf("ppm: truncated: need %d, have %d", hdrEnd+byteCount, len(data))
	}

	// Convert RGB bytes → image.RGBA (RGBA interleaved, A=255).
	pix := data[hdrEnd : hdrEnd+byteCount]
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	dst := img.Pix
	for i := 0; i < byteCount/3; i++ {
		di := i * 4
		si := i * 3
		dst[di] = pix[si]     // R
		dst[di+1] = pix[si+1] // G
		dst[di+2] = pix[si+2] // B
		dst[di+3] = 255       // A
	}

	return img, nil
}

// --- encoder -----------------------------------------------------------------

func encodeImage(path string, img image.Image, opts *Options) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer f.Close()

	switch opts.Format {
	case FormatJPEG:
		return jpeg.Encode(f, img, &jpeg.Options{Quality: opts.JPEGQuality})
	case FormatPNG:
		return png.Encode(f, img)
	default:
		return fmt.Errorf("unknown format: %v", opts.Format)
	}
}
