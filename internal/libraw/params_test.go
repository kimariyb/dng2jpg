package libraw

import "testing"

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.Format != FormatJPEG {
		t.Errorf("expected FormatJPEG, got %v", opts.Format)
	}
	if opts.JPEGQuality != 95 {
		t.Errorf("expected quality 95, got %d", opts.JPEGQuality)
	}
	if opts.WhiteBalance != WBAuto {
		t.Errorf("expected WBAuto, got %v", opts.WhiteBalance)
	}
	if opts.Brightness != 1.0 {
		t.Errorf("expected brightness 1.0, got %f", opts.Brightness)
	}
	if opts.HighlightMode != 0 {
		t.Errorf("expected highlight mode 0, got %d", opts.HighlightMode)
	}
	if opts.ColorSpace != ColorSpaceSRGB {
		t.Errorf("expected sRGB, got %v", opts.ColorSpace)
	}
	if opts.OutputDepth != 8 {
		t.Errorf("expected depth 8, got %d", opts.OutputDepth)
	}
}

func TestOptionsValidate_Valid(t *testing.T) {
	opts := DefaultOptions()
	if err := opts.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestOptionsValidate_BadQuality(t *testing.T) {
	opts := DefaultOptions()
	opts.JPEGQuality = 0
	if err := opts.Validate(); err == nil {
		t.Error("expected error for quality=0")
	}
	opts.JPEGQuality = 101
	if err := opts.Validate(); err == nil {
		t.Error("expected error for quality=101")
	}
}

func TestOptionsValidate_BadDepth(t *testing.T) {
	opts := DefaultOptions()
	opts.OutputDepth = 10
	if err := opts.Validate(); err == nil {
		t.Error("expected error for depth=10")
	}
}

func TestOptionsValidate_BadHighlight(t *testing.T) {
	opts := DefaultOptions()
	opts.HighlightMode = 4
	if err := opts.Validate(); err == nil {
		t.Error("expected error for highlight mode=4")
	}
}

func TestOptionsValidate_BadFormat(t *testing.T) {
	opts := DefaultOptions()
	opts.Format = OutputFormat(99)
	if err := opts.Validate(); err == nil {
		t.Error("expected error for unknown format")
	}
}

func TestOptionsValidate_BadWB(t *testing.T) {
	opts := DefaultOptions()
	opts.WhiteBalance = WBMode(99)
	if err := opts.Validate(); err == nil {
		t.Error("expected error for unknown white balance")
	}
}

func TestOptionsValidate_BadColorSpace(t *testing.T) {
	opts := DefaultOptions()
	opts.ColorSpace = ColorSpace(99)
	if err := opts.Validate(); err == nil {
		t.Error("expected error for unknown color space")
	}
}

func TestWBModeString_Unknown(t *testing.T) {
	if got := WBMode(99).String(); got != "unknown" {
		t.Errorf("expected 'unknown' for bad WBMode, got %q", got)
	}
}

func TestColorSpaceString_Unknown(t *testing.T) {
	if got := ColorSpace(99).String(); got != "unknown" {
		t.Errorf("expected 'unknown' for bad ColorSpace, got %q", got)
	}
}

func TestOutputFormatExt(t *testing.T) {
	if FormatJPEG.Ext() != ".jpg" {
		t.Errorf("expected .jpg, got %s", FormatJPEG.Ext())
	}
	if FormatPNG.Ext() != ".png" {
		t.Errorf("expected .png, got %s", FormatPNG.Ext())
	}
}

func TestOutputFormatExt_Unknown(t *testing.T) {
	if got := OutputFormat(99).Ext(); got != "" {
		t.Errorf("expected empty string for unknown format, got %q", got)
	}
}

func TestOutputFormatString_Unknown(t *testing.T) {
	if got := OutputFormat(99).String(); got != "unknown" {
		t.Errorf("expected 'unknown' for bad format, got %q", got)
	}
}

func TestOutputFormatString(t *testing.T) {
	if FormatJPEG.String() != "jpeg" {
		t.Errorf("expected jpeg, got %s", FormatJPEG.String())
	}
	if FormatPNG.String() != "png" {
		t.Errorf("expected png, got %s", FormatPNG.String())
	}
}

func TestWBModeString(t *testing.T) {
	tests := []struct {
		mode WBMode
		want string
	}{
		{WBAuto, "auto"},
		{WBCamera, "camera"},
		{WBCustom, "custom"},
	}
	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.want {
			t.Errorf("WBMode(%d).String() = %s, want %s", tt.mode, got, tt.want)
		}
	}
}

func TestColorSpaceString(t *testing.T) {
	tests := []struct {
		cs   ColorSpace
		want string
	}{
		{ColorSpaceSRGB, "sRGB"},
		{ColorSpaceAdobeRGB, "AdobeRGB"},
		{ColorSpaceProPhoto, "ProPhoto"},
		{ColorSpaceWideGamut, "WideGamut"},
	}
	for _, tt := range tests {
		if got := tt.cs.String(); got != tt.want {
			t.Errorf("ColorSpace(%d).String() = %s, want %s", tt.cs, got, tt.want)
		}
	}
}
