package app

import "testing"

func TestExtractFallbackToolArgsCleansProgressQuestion(t *testing.T) {
	args := extractFallbackToolArgs("查一下齿轮到哪儿了?")

	if args["part_name"] != "齿轮" {
		t.Fatalf("expected part_name 齿轮, got %#v", args["part_name"])
	}
}

func TestExtractAfterLabelHandlesChinesePartName(t *testing.T) {
	got := extractAfterLabel("零件名称：齿轮", `零件名称|零件|产品|品名`)

	if got != "齿轮" {
		t.Fatalf("expected 齿轮, got %q", got)
	}
}

func TestSelectToolNameForReporterQuestion(t *testing.T) {
	got := selectToolName("查一下齿轮谁做的", "")

	if got != "query_order_reporters" {
		t.Fatalf("expected query_order_reporters, got %q", got)
	}
}

func TestSelectToolNameForReportDetailsQuestion(t *testing.T) {
	got := selectToolName("查一下齿轮的报工明细", "")

	if got != "query_order_report_details" {
		t.Fatalf("expected query_order_report_details, got %q", got)
	}
}

func TestCleanProgressKeywordRemovesReporterQuestion(t *testing.T) {
	got := cleanProgressKeyword("谁做的")

	if got != "" {
		t.Fatalf("expected empty keyword, got %q", got)
	}
}

func TestApplyPreviousQueryContextForFollowUp(t *testing.T) {
	userID := uint(99)
	aiQueryContexts.Delete(userID)
	saveQueryContext(userID, map[string]interface{}{"part_name": "齿轮", "limit": 5})

	args := applyPreviousQueryContext(userID, map[string]interface{}{"limit": 5})

	if args["part_name"] != "齿轮" {
		t.Fatalf("expected previous part_name 齿轮, got %#v", args["part_name"])
	}
	aiQueryContexts.Delete(userID)
}
