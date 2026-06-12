package libraw

import (
	"fmt"
	"time"
)

// OutputFormat specifies the target image format.
type OutputFormat int

const (
	FormatJPEG OutputFormat = iota
	FormatPNG
)

func (f OutputFormat) String() string {
	switch f {
	case FormatJPEG:
		return "jpeg"
	case FormatPNG:
		return "png"
	default:
		return "unknown"
	}
}

func (f OutputFormat) Ext() string {
	switch f {
	case FormatJPEG:
		return ".jpg"
	case FormatPNG:
		return ".png"
	default:
		return ""
	}
}

// WBMode controls white balance behavior.
type WBMode int

const (
	WBAuto   WBMode = iota // automatic white balance
	WBCamera               // use camera-as-shot white balance
	WBCustom               // use CustomWBMul coefficients
)

func (w WBMode) String() string {
	switch w {
	case WBAuto:
		return "auto"
	case WBCamera:
		return "camera"
	case WBCustom:
		return "custom"
	default:
		return "unknown"
	}
}

// ColorSpace selects the output color space.
type ColorSpace int

const (
	ColorSpaceSRGB      ColorSpace = iota // sRGB
	ColorSpaceAdobeRGB                    // Adobe RGB (1998)
	ColorSpaceProPhoto                    // ProPhoto RGB
	ColorSpaceWideGamut                   // Wide Gamut RGB
)

func (c ColorSpace) String() string {
	switch c {
	case ColorSpaceSRGB:
		return "sRGB"
	case ColorSpaceAdobeRGB:
		return "AdobeRGB"
	case ColorSpaceProPhoto:
		return "ProPhoto"
	case ColorSpaceWideGamut:
		return "WideGamut"
	default:
		return "unknown"
	}
}

// Options holds all conversion parameters.
type Options struct {
	Format          OutputFormat // output format (default FormatJPEG)
	JPEGQuality     int          // 1-100 (default 95)
	WhiteBalance    WBMode       // white balance mode (default WBAuto)
	CustomWBMul     [4]float64   // custom R, G1, B, G2 multipliers
	ExposureComp    float64      // exposure compensation in EV (default 0)
	Brightness      float64      // brightness multiplier (default 1.0)
	HighlightMode   int          // 0=clip, 1=unclip, 2=blend, 3=rebuild (default 1)
	ColorSpace      ColorSpace   // output color space (default sRGB)
	OutputDepth     int          // 8 or 16 bits per channel (default 8)
	DenoiseThreshold float64     // wavelet denoising threshold (default 0=off)
	MedianFilterPasses int       // median filter passes (default 0=off)
	AutoBrightness  bool         // auto-brightness correction
	HalfSize        bool         // output at half resolution
	UseThumbnail    bool         // extract embedded thumbnail instead of developing
	NoAutoOrient    bool         // disable automatic rotation
}

// DefaultOptions returns an Options with sensible defaults.
func DefaultOptions() *Options {
	return &Options{
		Format:        FormatJPEG,
		JPEGQuality:   95,
		WhiteBalance:  WBAuto,
		Brightness:    1.0,
		HighlightMode: 0,
		ColorSpace:    ColorSpaceSRGB,
		OutputDepth:   8,
	}
}

// Validate checks the Options for invalid values and returns an error.
func (o *Options) Validate() error {
	if o.Format != FormatJPEG && o.Format != FormatPNG {
		return fmt.Errorf("unknown output format: %d", o.Format)
	}
	if o.JPEGQuality < 1 || o.JPEGQuality > 100 {
		return fmt.Errorf("quality must be 1-100, got %d", o.JPEGQuality)
	}
	if o.WhiteBalance < WBAuto || o.WhiteBalance > WBCustom {
		return fmt.Errorf("unknown white balance mode: %d", o.WhiteBalance)
	}
	if o.OutputDepth != 8 && o.OutputDepth != 16 {
		return fmt.Errorf("depth must be 8 or 16, got %d", o.OutputDepth)
	}
	if o.HighlightMode < 0 || o.HighlightMode > 3 {
		return fmt.Errorf("highlight mode must be 0-3, got %d", o.HighlightMode)
	}
	if o.ColorSpace < ColorSpaceSRGB || o.ColorSpace > ColorSpaceWideGamut {
		return fmt.Errorf("unknown color space: %d", o.ColorSpace)
	}
	if o.Brightness <= 0 {
		return fmt.Errorf("brightness must be > 0, got %f", o.Brightness)
	}
	return nil
}

// ConvertResult holds the outcome of a single file conversion.
type ConvertResult struct {
	InputPath  string
	OutputPath string
	Duration   time.Duration
	Err        error
}
