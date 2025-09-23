package resource

import "testing"

func TestToSnake(t *testing.T) {
	for _, c := range []struct {
		in, out string
	}{
		{"", ""},
		{"unchanged", "unchanged"},
		{"with-dash", "with_dash"},
		{"caseChange", "case_change"},
		{"ALLUPPER", "allupper"},
		{"-leading-dash", "_leading_dash"}, // probably not ideal, but what can you do
		{"LeadingCaps", "leading_caps"},
		{"both-Together", "both_together"},
		{"EverY-ThinG", "ever_y_thin_g"},
		{"multiple--dashes", "multiple__dashes"},
		{"LoTSOfCASEchanges", "lo_tsof_casechanges"},
		{"underscore_Capital", "underscore_capital"},
	} {
		t.Run(c.in, func(t *testing.T) {
			got := ToSnake(c.in)
			if got != c.out {
				t.Errorf("ToSnake(%q):\n got: %q\nwant: %q", c.in, got, c.out)
			}
		})
	}
}
