package approval

import "testing"

func TestNormalizeChoiceColorsAssignsByIndex(t *testing.T) {
	options := map[string]any{
		"choices": []any{
			map[string]any{"name": "Pending"},
			map[string]any{"name": "Approved"},
			map[string]any{"name": "Rejected"},
		},
	}

	normalizeChoiceColors(options)

	choices := options["choices"].([]any)
	want := []string{"blueLight2", "cyanLight2", "tealLight2"}
	for i, c := range choices {
		got, _ := c.(map[string]any)["color"].(string)
		if got != want[i] {
			t.Fatalf("choice %d: got color %q, want %q", i, got, want[i])
		}
	}
}

func TestNormalizeChoiceColorsPreservesExplicitColor(t *testing.T) {
	options := map[string]any{
		"choices": []any{
			map[string]any{"name": "Hot", "color": "redBright"},
			map[string]any{"name": "Cold"},
		},
	}

	normalizeChoiceColors(options)

	choices := options["choices"].([]any)
	if got := choices[0].(map[string]any)["color"]; got != "redBright" {
		t.Fatalf("explicit color should be preserved, got %v", got)
	}
	if got := choices[1].(map[string]any)["color"]; got != "cyanLight2" {
		t.Fatalf("second choice should get index-1 color cyanLight2, got %v", got)
	}
}

func TestNormalizeChoiceColorsIgnoresOptionsWithoutChoices(t *testing.T) {
	options := map[string]any{"precision": 0}
	out := normalizeChoiceColors(options)
	if _, present := out["choices"]; present {
		t.Fatal("did not expect a choices key to be added to a non-select field")
	}
}
