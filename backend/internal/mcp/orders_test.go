package mcp

import "testing"

func TestPartNameCandidatesStripsSizePrefix(t *testing.T) {
	candidates := partNameCandidates("大齿轮")

	if len(candidates) < 2 || candidates[0] != "大齿轮" || candidates[1] != "齿轮" {
		t.Fatalf("expected candidates to start with 大齿轮, 齿轮; got %#v", candidates)
	}
}
