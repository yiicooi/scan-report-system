package mcp

func QueryOrderReporters(arguments map[string]interface{}) ([]OrderReportersResult, error) {
	detailsResults, err := QueryOrderReportDetails(arguments)
	if err != nil {
		return nil, err
	}

	results := make([]OrderReportersResult, 0, len(detailsResults))
	for _, orderDetails := range detailsResults {
		byUser := map[uint]*ReporterItem{}
		processSeen := map[uint]map[string]bool{}
		for _, detail := range orderDetails.Details {
			item, ok := byUser[detail.UserID]
			if !ok {
				item = &ReporterItem{
					UserID:         detail.UserID,
					UserName:       detail.UserName,
					DepartmentName: detail.DepartmentName,
				}
				byUser[detail.UserID] = item
				processSeen[detail.UserID] = map[string]bool{}
			}
			item.TotalReceived += detail.ReceivedQty
			item.TotalCompleted += detail.CompletedQty
			item.TotalScrap += detail.ScrapQty
			item.ReportCount++
			if detail.ReportedAt.After(item.LastReportedAt) {
				item.LastReportedAt = detail.ReportedAt
			}
			if detail.ProcessName != "" && !processSeen[detail.UserID][detail.ProcessName] {
				item.ProcessNames = append(item.ProcessNames, detail.ProcessName)
				processSeen[detail.UserID][detail.ProcessName] = true
			}
		}

		reporters := make([]ReporterItem, 0, len(byUser))
		for _, item := range byUser {
			reporters = append(reporters, *item)
		}
		results = append(results, OrderReportersResult{
			Order:     orderDetails.Order,
			Reporters: reporters,
		})
	}
	return results, nil
}
