package jsfx

import (
	"strings"
	"testing"
)

func TestJSFXDSLParser_ParseBasicEffect(t *testing.T) {
	parser, err := NewJSFXDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `effect(name="Simple Gain", tags="utility volume");
slider(id=1, default=0, min=-60, max=12, step=0.1, name="Gain (dB)");
init_code(code="gain = 1;");
slider_code(code="gain = 10^(slider1/20);");
sample_code(code="spl0 *= gain; spl1 *= gain;")`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("Failed to parse DSL: %v", err)
	}

	if len(actions) != 5 {
		t.Errorf("Expected 5 actions, got %d", len(actions))
	}

	// Check effect action
	effectAction := actions[0]
	if effectAction["action"] != "effect" {
		t.Errorf("Expected 'effect' action, got %v", effectAction["action"])
	}
	if effectAction["name"] != "Simple Gain" {
		t.Errorf("Expected name 'Simple Gain', got %v", effectAction["name"])
	}
	if effectAction["tags"] != "utility volume" {
		t.Errorf("Expected tags 'utility volume', got %v", effectAction["tags"])
	}

	// Check slider action
	sliderAction := actions[1]
	if sliderAction["action"] != "slider" {
		t.Errorf("Expected 'slider' action, got %v", sliderAction["action"])
	}
	if sliderAction["id"].(float64) != 1 {
		t.Errorf("Expected slider id 1, got %v", sliderAction["id"])
	}
	if sliderAction["min"].(float64) != -60 {
		t.Errorf("Expected min -60, got %v", sliderAction["min"])
	}
	if sliderAction["max"].(float64) != 12 {
		t.Errorf("Expected max 12, got %v", sliderAction["max"])
	}
}

func TestJSFXDSLParser_GenerateJSFXCode(t *testing.T) {
	actions := []map[string]any{
		{
			"action": "effect",
			"name":   "Test Effect",
			"tags":   "test",
		},
		{
			"action":  "slider",
			"id":      float64(1),
			"default": float64(0),
			"min":     float64(-60),
			"max":     float64(12),
			"step":    float64(0.1),
			"name":    "Gain (dB)",
		},
		{
			"action": "init_code",
			"code":   "gain = 1;",
		},
		{
			"action": "slider_code",
			"code":   "gain = 10^(slider1/20);",
		},
		{
			"action": "sample_code",
			"code":   "spl0 *= gain; spl1 *= gain;",
		},
	}

	jsfxCode := generateJSFXCode(actions)

	// Verify structure
	if !strings.Contains(jsfxCode, "desc:Test Effect") {
		t.Errorf("Missing desc line")
	}
	if !strings.Contains(jsfxCode, "tags:test") {
		t.Errorf("Missing tags line")
	}
	if !strings.Contains(jsfxCode, "slider1:") {
		t.Errorf("Missing slider definition")
	}
	if !strings.Contains(jsfxCode, "@init") {
		t.Errorf("Missing @init section")
	}
	if !strings.Contains(jsfxCode, "@slider") {
		t.Errorf("Missing @slider section")
	}
	if !strings.Contains(jsfxCode, "@sample") {
		t.Errorf("Missing @sample section")
	}
	if !strings.Contains(jsfxCode, "gain = 1") {
		t.Errorf("Missing init code content")
	}
	if !strings.Contains(jsfxCode, "spl0 *= gain") {
		t.Errorf("Missing sample code content")
	}
}

func TestJSFXDSLParser_CompressorEffect(t *testing.T) {
	parser, err := NewJSFXDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `effect(name="Simple Compressor", tags="dynamics compressor", type=compressor);
slider(id=1, default=-20, min=-60, max=0, step=1, name="Threshold (dB)");
slider(id=2, default=4, min=1, max=20, step=0.1, name="Ratio");
slider(id=3, default=10, min=0.1, max=100, step=0.1, name="Attack (ms)");
slider(id=4, default=100, min=10, max=1000, step=1, name="Release (ms)");
init_code(code="env = 0; thresh_lin = 0.1; ratio_inv = 0.25;");
slider_code(code="thresh_lin = 10^(slider1/20); ratio_inv = 1/slider2; attack_coef = exp(-1/(srate*slider3/1000)); release_coef = exp(-1/(srate*slider4/1000));");
sample_code(code="input_level = max(abs(spl0), abs(spl1)); env = input_level > env ? attack_coef*(env-input_level)+input_level : release_coef*(env-input_level)+input_level; gain_reduction = env > thresh_lin ? (thresh_lin/env)^(1-ratio_inv) : 1; spl0 *= gain_reduction; spl1 *= gain_reduction;")`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("Failed to parse DSL: %v", err)
	}

	// Should have: 1 effect + 4 sliders + 3 code sections = 8 actions
	if len(actions) != 8 {
		t.Errorf("Expected 8 actions, got %d", len(actions))
	}

	// Generate and verify code
	jsfxCode := generateJSFXCode(actions)
	if !strings.Contains(jsfxCode, "Simple Compressor") {
		t.Errorf("Missing effect name")
	}
	if !strings.Contains(jsfxCode, "Threshold") {
		t.Errorf("Missing threshold slider")
	}
	if !strings.Contains(jsfxCode, "gain_reduction") {
		t.Errorf("Missing gain_reduction in sample code")
	}
}

func TestJSFXDSLParser_FilterEffect(t *testing.T) {
	parser, err := NewJSFXDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `effect(name="Lowpass Filter", tags="filter eq", type=filter);
slider(id=1, default=1000, min=20, max=20000, step=1, name="Cutoff (Hz)");
slider(id=2, default=0.707, min=0.1, max=10, step=0.01, name="Q");
init_code(code="a0=a1=a2=b1=b2=x1l=x2l=y1l=y2l=x1r=x2r=y1r=y2r=0;");
slider_code(code="omega = 2*$pi*slider1/srate; alpha = sin(omega)/(2*slider2); cosw = cos(omega); a0 = (1-cosw)/2; a1 = 1-cosw; a2 = a0; b0 = 1+alpha; b1 = -2*cosw; b2 = 1-alpha; a0/=b0; a1/=b0; a2/=b0; b1/=b0; b2/=b0;");
sample_code(code="y0l = a0*spl0 + a1*x1l + a2*x2l - b1*y1l - b2*y2l; x2l=x1l; x1l=spl0; y2l=y1l; y1l=y0l; spl0=y0l; y0r = a0*spl1 + a1*x1r + a2*x2r - b1*y1r - b2*y2r; x2r=x1r; x1r=spl1; y2r=y1r; y1r=y0r; spl1=y0r;")`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("Failed to parse DSL: %v", err)
	}

	// Should have: 1 effect + 2 sliders + 3 code sections = 6 actions
	if len(actions) != 6 {
		t.Errorf("Expected 6 actions, got %d", len(actions))
	}

	jsfxCode := generateJSFXCode(actions)
	if !strings.Contains(jsfxCode, "Lowpass Filter") {
		t.Errorf("Missing effect name")
	}
	if !strings.Contains(jsfxCode, "Cutoff") {
		t.Errorf("Missing cutoff slider")
	}
}

func TestJSFXDSLParser_FullFeaturedEffect(t *testing.T) {
	parser, err := NewJSFXDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Test with pins, imports, options, and all code sections
	dsl := `effect(name="Stereo Processor", tags="utility stereo");
pin(type=in, name="Left In", channel=0);
pin(type=in, name="Right In", channel=1);
pin(type=out, name="Left Out", channel=0);
pin(type=out, name="Right Out", channel=1);
slider(id=1, default=0, min=-12, max=12, step=0.1, name="Gain (dB)");
slider(id=2, default=100, min=0, max=200, step=1, name="Width (%)");
init_code(code="gain = 1; width = 1;");
slider_code(code="gain = 10^(slider1/20); width = slider2/100;");
sample_code(code="mid = (spl0+spl1)*0.5; side = (spl0-spl1)*0.5*width; spl0 = (mid+side)*gain; spl1 = (mid-side)*gain;");
gfx_code(code="gfx_set(1,1,1); gfx_drawstr(\"Stereo Processor\");", width=300, height=100)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("Failed to parse DSL: %v", err)
	}

	// 1 effect + 4 pins + 2 sliders + 3 code sections + 1 gfx = 11 actions
	if len(actions) != 11 {
		t.Errorf("Expected 11 actions, got %d", len(actions))
	}

	jsfxCode := generateJSFXCode(actions)

	// Verify all sections
	if !strings.Contains(jsfxCode, "desc:Stereo Processor") {
		t.Errorf("Missing effect desc")
	}
	if !strings.Contains(jsfxCode, "in_pin:Left In") {
		t.Errorf("Missing input pin")
	}
	if !strings.Contains(jsfxCode, "out_pin:Left Out") {
		t.Errorf("Missing output pin")
	}
	if !strings.Contains(jsfxCode, "slider1:") {
		t.Errorf("Missing slider definition")
	}
	if !strings.Contains(jsfxCode, "@init") {
		t.Errorf("Missing @init section")
	}
	if !strings.Contains(jsfxCode, "@sample") {
		t.Errorf("Missing @sample section")
	}
	if !strings.Contains(jsfxCode, "@gfx 300 100") {
		t.Errorf("Missing @gfx section with dimensions")
	}
	if !strings.Contains(jsfxCode, "mid+side") {
		t.Errorf("Missing M/S processing code")
	}
}

func TestJSFXDSLParser_MIDIEffect(t *testing.T) {
	parser, err := NewJSFXDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `effect(name="MIDI Transpose", tags="midi utility", type=midi);
slider(id=1, default=0, min=-24, max=24, step=1, name="Semitones");
block_code(code="while (midirecv(offset, msg1, msg2, msg3)) ( status = msg1 & 0xF0; status == 0x90 || status == 0x80 ? ( msg2 = max(0, min(127, msg2 + slider1)); ); midisend(offset, msg1, msg2, msg3); );")`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("Failed to parse DSL: %v", err)
	}

	// 1 effect + 1 slider + 1 block_code = 3 actions
	if len(actions) != 3 {
		t.Errorf("Expected 3 actions, got %d", len(actions))
	}

	jsfxCode := generateJSFXCode(actions)
	if !strings.Contains(jsfxCode, "MIDI Transpose") {
		t.Errorf("Missing effect name")
	}
	if !strings.Contains(jsfxCode, "@block") {
		t.Errorf("Missing @block section")
	}
	if !strings.Contains(jsfxCode, "midirecv") {
		t.Errorf("Missing MIDI code")
	}
}

