package jsfx

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJSFXDSLParser_ParseBasicEffect tests parsing a basic gain effect
func TestJSFXDSLParser_ParseBasicEffect(t *testing.T) {
	dsl := `effect(name="Simple Gain", tags="utility volume");slider(id=1, default=0, min=-60, max=12, step=0.1, name="Gain (dB)");init_code(code="gain = 1;");slider_code(code="gain = 10^(slider1/20);");sample_code(code="spl0 *= gain; spl1 *= gain;")`

	parser, err := NewJSFXDSLParser()
	require.NoError(t, err)

	jsfxCode, err := parser.ParseDSL(dsl)
	require.NoError(t, err)

	// Verify JSFX structure
	assert.Contains(t, jsfxCode, "desc:Simple Gain")
	assert.Contains(t, jsfxCode, "tags:utility volume")
	assert.Contains(t, jsfxCode, "slider1:")
	assert.Contains(t, jsfxCode, "@init")
	assert.Contains(t, jsfxCode, "@slider")
	assert.Contains(t, jsfxCode, "@sample")
}

// TestJSFXDSLParser_GenerateJSFXCode tests the JSFX code generation
func TestJSFXDSLParser_GenerateJSFXCode(t *testing.T) {
	dsl := `effect(name="Test Effect", tags="test");slider(id=1, default=50, min=0, max=100, step=1, name="Mix")`

	parser, err := NewJSFXDSLParser()
	require.NoError(t, err)

	jsfxCode, err := parser.ParseDSL(dsl)
	require.NoError(t, err)

	// Verify structure
	assert.Contains(t, jsfxCode, "desc:Test Effect")
	assert.Contains(t, jsfxCode, "tags:test")
	assert.Contains(t, jsfxCode, "slider1:")
	assert.Contains(t, jsfxCode, "<0.00,100.00,")
}

// TestJSFXDSLParser_CompressorEffect tests parsing a compressor effect
func TestJSFXDSLParser_CompressorEffect(t *testing.T) {
	dsl := `effect(name="Simple Compressor", tags="dynamics compressor");` +
		`slider(id=1, default=-20, min=-60, max=0, step=0.1, name="Threshold (dB)");` +
		`slider(id=2, default=4, min=1, max=20, step=0.1, name="Ratio");` +
		`slider(id=3, default=10, min=0.1, max=100, step=0.1, name="Attack (ms)");` +
		`slider(id=4, default=100, min=10, max=1000, step=1, name="Release (ms)");` +
		`init_code(code="env = 0; attack_coef = 0; release_coef = 0;");` +
		`slider_code(code="thresh_lin = 10^(slider1/20);\nratio_inv = 1/slider2;\nattack_coef = exp(-1/(srate*slider3/1000));\nrelease_coef = exp(-1/(srate*slider4/1000));");` +
		`sample_code(code="input_level = max(abs(spl0), abs(spl1));\nenv = input_level > env ? attack_coef*(env-input_level)+input_level : release_coef*(env-input_level)+input_level;\ngain = env > thresh_lin ? (thresh_lin/env)^(1-ratio_inv) : 1;\nspl0 *= gain; spl1 *= gain;")`

	parser, err := NewJSFXDSLParser()
	require.NoError(t, err)

	jsfxCode, err := parser.ParseDSL(dsl)
	require.NoError(t, err)

	// Verify compressor structure
	assert.Contains(t, jsfxCode, "desc:Simple Compressor")
	assert.Contains(t, jsfxCode, "tags:dynamics compressor")
	assert.Contains(t, jsfxCode, "slider1:")
	assert.Contains(t, jsfxCode, "slider2:")
	assert.Contains(t, jsfxCode, "slider3:")
	assert.Contains(t, jsfxCode, "slider4:")
	assert.Contains(t, jsfxCode, "Threshold")
	assert.Contains(t, jsfxCode, "Ratio")
	assert.Contains(t, jsfxCode, "Attack")
	assert.Contains(t, jsfxCode, "Release")
}

// TestJSFXDSLParser_FilterEffect tests parsing a filter effect
func TestJSFXDSLParser_FilterEffect(t *testing.T) {
	dsl := `effect(name="Lowpass Filter", tags="filter eq");` +
		`slider(id=1, default=1000, min=20, max=20000, step=1, name="Cutoff (Hz)");` +
		`slider(id=2, default=0.707, min=0.1, max=10, step=0.01, name="Q");` +
		`init_code(code="a0=a1=a2=b1=b2=x1l=x2l=y1l=y2l=x1r=x2r=y1r=y2r=0;");` +
		`slider_code(code="omega = 2*$pi*slider1/srate;\nsn = sin(omega); cs = cos(omega);\nalpha = sn/(2*slider2);\nb0 = (1-cs)/2; b1 = 1-cs; b2 = (1-cs)/2;\na0 = 1+alpha; a1 = -2*cs; a2 = 1-alpha;\nb0 /= a0; b1 /= a0; b2 /= a0; a1 /= a0; a2 /= a0;");` +
		`sample_code(code="y0l = b0*spl0 + b1*x1l + b2*x2l - a1*y1l - a2*y2l;\nx2l=x1l; x1l=spl0; y2l=y1l; y1l=y0l; spl0=y0l;\ny0r = b0*spl1 + b1*x1r + b2*x2r - a1*y1r - a2*y2r;\nx2r=x1r; x1r=spl1; y2r=y1r; y1r=y0r; spl1=y0r;")`

	parser, err := NewJSFXDSLParser()
	require.NoError(t, err)

	jsfxCode, err := parser.ParseDSL(dsl)
	require.NoError(t, err)

	// Verify filter structure
	assert.Contains(t, jsfxCode, "desc:Lowpass Filter")
	assert.Contains(t, jsfxCode, "tags:filter eq")
	assert.Contains(t, jsfxCode, "Cutoff")
	assert.Contains(t, jsfxCode, "@init")
	assert.Contains(t, jsfxCode, "@slider")
	assert.Contains(t, jsfxCode, "@sample")
}

// TestJSFXDSLParser_FullFeaturedEffect tests an effect with pins and all sections
func TestJSFXDSLParser_FullFeaturedEffect(t *testing.T) {
	dsl := `effect(name="Stereo Processor", tags="utility stereo");` +
		`pin(type=in, name="Left In", channel=0);` +
		`pin(type=in, name="Right In", channel=1);` +
		`pin(type=out, name="Left Out", channel=0);` +
		`pin(type=out, name="Right Out", channel=1);` +
		`slider(id=1, default=0, min=-12, max=12, step=0.1, name="Gain (dB)");` +
		`slider(id=2, default=100, min=0, max=200, step=1, name="Width (%)");` +
		`init_code(code="gain = 1; width = 1;");` +
		`slider_code(code="gain = 10^(slider1/20); width = slider2/100;");` +
		`sample_code(code="mid = (spl0+spl1)/2; side = (spl0-spl1)/2;\nspl0 = (mid + side*width) * gain;\nspl1 = (mid - side*width) * gain;");` +
		`gfx_code(code="gfx_drawstr(\"Stereo Processor\");", width=300, height=100)`

	parser, err := NewJSFXDSLParser()
	require.NoError(t, err)

	jsfxCode, err := parser.ParseDSL(dsl)
	require.NoError(t, err)

	// Verify full structure
	assert.Contains(t, jsfxCode, "desc:Stereo Processor")
	assert.Contains(t, jsfxCode, "in_pin:Left In")
	assert.Contains(t, jsfxCode, "in_pin:Right In")
	assert.Contains(t, jsfxCode, "out_pin:Left Out")
	assert.Contains(t, jsfxCode, "out_pin:Right Out")
	assert.Contains(t, jsfxCode, "@init")
	assert.Contains(t, jsfxCode, "@slider")
	assert.Contains(t, jsfxCode, "@sample")
	assert.Contains(t, jsfxCode, "@gfx 300 100")
}

// TestJSFXDSLParser_MIDIEffect tests parsing a MIDI effect
func TestJSFXDSLParser_MIDIEffect(t *testing.T) {
	dsl := `effect(name="MIDI Transpose", tags="midi utility");` +
		`slider(id=1, default=0, min=-24, max=24, step=1, name="Semitones");` +
		`block_code(code="while (midirecv(offset, msg1, msg2, msg3)) (\n  status = msg1 & 0xF0;\n  (status == 0x90 || status == 0x80) ? (\n    msg2 = min(max(msg2 + slider1, 0), 127);\n  );\n  midisend(offset, msg1, msg2, msg3);\n);")`

	parser, err := NewJSFXDSLParser()
	require.NoError(t, err)

	jsfxCode, err := parser.ParseDSL(dsl)
	require.NoError(t, err)

	// Verify MIDI structure
	assert.Contains(t, jsfxCode, "desc:MIDI Transpose")
	assert.Contains(t, jsfxCode, "tags:midi utility")
	assert.Contains(t, jsfxCode, "@block")
	assert.Contains(t, jsfxCode, "midirecv")
	assert.Contains(t, jsfxCode, "midisend")
}

// TestJSFXDSLParser_SliderFormatting tests slider line formatting
func TestJSFXDSLParser_SliderFormatting(t *testing.T) {
	tests := []struct {
		name     string
		dsl      string
		expected string
	}{
		{
			name:     "basic slider",
			dsl:      `effect(name="Test");slider(id=1, default=0, min=-60, max=12, step=0.1, name="Gain")`,
			expected: "slider1:0.00<-60.00,12.00,0.1000>Gain",
		},
		{
			name:     "slider with var",
			dsl:      `effect(name="Test");slider(id=2, default=50, min=0, max=100, step=1, name="Mix", var=mix_amt)`,
			expected: "slider2:mix_amt=50.00<0.00,100.00,1.0000>Mix",
		},
		{
			name:     "hidden slider",
			dsl:      `effect(name="Test");slider(id=3, default=1, min=0, max=1, step=1, name="Internal", hidden=true)`,
			expected: "slider3:1.00<0.00,1.00,1.0000>-Internal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewJSFXDSLParser()
			require.NoError(t, err)

			jsfxCode, err := parser.ParseDSL(tt.dsl)
			require.NoError(t, err)

			assert.Contains(t, jsfxCode, tt.expected)
		})
	}
}

// TestJSFXDSLParser_CodeUnescaping tests that code strings are properly unescaped
func TestJSFXDSLParser_CodeUnescaping(t *testing.T) {
	dsl := `effect(name="Test");init_code(code="a = 1;\nb = 2;\nc = \"test\";")`

	parser, err := NewJSFXDSLParser()
	require.NoError(t, err)

	jsfxCode, err := parser.ParseDSL(dsl)
	require.NoError(t, err)

	// Should have unescaped newlines
	assert.Contains(t, jsfxCode, "a = 1;\nb = 2;")
}

// TestJSFXDSLParser_EmptyDSL tests handling of empty DSL
func TestJSFXDSLParser_EmptyDSL(t *testing.T) {
	parser, err := NewJSFXDSLParser()
	require.NoError(t, err)

	_, err = parser.ParseDSL("")
	assert.Error(t, err, "Should error on empty DSL")
}

// TestJSFXDSLParser_MultipleSliders tests multiple sliders are ordered correctly
func TestJSFXDSLParser_MultipleSliders(t *testing.T) {
	dsl := `effect(name="Multi");` +
		`slider(id=1, default=0, min=0, max=100, step=1, name="First");` +
		`slider(id=2, default=50, min=0, max=100, step=1, name="Second");` +
		`slider(id=3, default=100, min=0, max=100, step=1, name="Third")`

	parser, err := NewJSFXDSLParser()
	require.NoError(t, err)

	jsfxCode, err := parser.ParseDSL(dsl)
	require.NoError(t, err)

	// Verify all sliders present
	assert.Contains(t, jsfxCode, "slider1:")
	assert.Contains(t, jsfxCode, "slider2:")
	assert.Contains(t, jsfxCode, "slider3:")

	// Verify order (slider1 appears before slider2, etc.)
	idx1 := strings.Index(jsfxCode, "slider1:")
	idx2 := strings.Index(jsfxCode, "slider2:")
	idx3 := strings.Index(jsfxCode, "slider3:")
	assert.Less(t, idx1, idx2, "slider1 should come before slider2")
	assert.Less(t, idx2, idx3, "slider2 should come before slider3")
}
