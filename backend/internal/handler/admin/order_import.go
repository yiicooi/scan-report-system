package admin

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sui/scan-report/internal/database"
	"github.com/sui/scan-report/internal/model"
	"github.com/sui/scan-report/internal/repository"
	"gorm.io/gorm"
)

type importOrderInput struct {
	ExternalNo  string
	PartName    string
	DrawingNo   string
	TotalQty    int
	UnitPrice   float64
	TotalAmount float64
	OrderDate   time.Time
}

type importResult struct {
	Row        int    `json:"row"`
	Success    bool   `json:"success"`
	InternalNo string `json:"internal_no,omitempty"`
	Error      string `json:"error,omitempty"`
}

// ImportOrdersExcel POST /api/admin/orders/import-excel
func ImportOrdersExcel(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择 Excel 文件"})
		return
	}
	if !strings.EqualFold(path.Ext(file.Filename), ".xlsx") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 .xlsx 文件"})
		return
	}

	rows, err := readXLSXRows(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(rows) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Excel 至少需要表头和一行数据"})
		return
	}

	headers := buildHeaderMap(rows[0])
	var results []importResult
	var successCount int
	userID := c.GetUint("user_id")
	seenExternalNos := map[string]int{}

	for i, row := range rows[1:] {
		rowNo := i + 2
		if isEmptyRow(row) {
			continue
		}
		input, err := rowToImportOrder(row, headers)
		if err != nil {
			results = append(results, importResult{Row: rowNo, Success: false, Error: err.Error()})
			continue
		}
		if input.ExternalNo != "" {
			if firstRow, ok := seenExternalNos[input.ExternalNo]; ok {
				results = append(results, importResult{Row: rowNo, Success: false, Error: fmt.Sprintf("外部单号与第 %d 行重复", firstRow)})
				continue
			}
			seenExternalNos[input.ExternalNo] = rowNo
		}

		var internalNo string
		err = database.DB.Transaction(func(tx *gorm.DB) error {
			if input.ExternalNo != "" {
				var count int64
				if err := tx.Model(&model.Order{}).Where("external_no = ?", input.ExternalNo).Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					return fmt.Errorf("外部单号已存在：%s", input.ExternalNo)
				}
			}
			orderNo, genErr := repository.GenerateInternalNo(tx, time.Now())
			if genErr != nil {
				return genErr
			}
			internalNo = orderNo
			order := model.Order{
				InternalNo:     orderNo,
				ExternalNo:     input.ExternalNo,
				PartName:       input.PartName,
				DrawingNo:      input.DrawingNo,
				TotalQty:       input.TotalQty,
				UnitPrice:      input.UnitPrice,
				TotalAmount:    input.TotalAmount,
				OrderDate:      input.OrderDate,
				Status:         model.OrderStatusDraft,
				OrderType:      model.OrderTypeNormal,
				TotalCompleted: 0,
				CreatedBy:      userID,
			}
			if err := tx.Create(&order).Error; err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			results = append(results, importResult{Row: rowNo, Success: false, Error: err.Error()})
			continue
		}
		successCount++
		results = append(results, importResult{Row: rowNo, Success: true, InternalNo: internalNo})
	}

	c.JSON(http.StatusOK, gin.H{
		"success_count": successCount,
		"fail_count":    len(results) - successCount,
		"results":       results,
	})
}

func rowToImportOrder(row []string, headers map[string]int) (importOrderInput, error) {
	value := func(key string) string {
		if idx, ok := headers[key]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}
	qty, err := parseInt(value("total_qty"))
	if err != nil || qty <= 0 {
		return importOrderInput{}, fmt.Errorf("订单数量必须大于 0")
	}
	orderDate, err := parseFlexibleDate(value("order_date"))
	if err != nil {
		return importOrderInput{}, fmt.Errorf("订单日期格式错误：%v", err)
	}
	if orderDate.IsZero() {
		orderDate = time.Now()
	}

	return importOrderInput{
		ExternalNo:  value("external_no"),
		PartName:    value("part_name"),
		DrawingNo:   value("drawing_no"),
		TotalQty:    qty,
		UnitPrice:   parseFloat(value("unit_price")),
		TotalAmount: parseFloat(value("total_amount")),
		OrderDate:   orderDate,
	}, nil
}

func buildHeaderMap(headers []string) map[string]int {
	aliases := map[string][]string{
		"external_no":  {"外部单号", "外部订单号", "客户单号", "external_no"},
		"part_name":    {"零件名称", "产品名称", "品名", "part_name"},
		"drawing_no":   {"图纸编号", "图号", "drawing_no"},
		"total_qty":    {"订单数量", "数量", "total_qty"},
		"unit_price":   {"单价", "unit_price"},
		"total_amount": {"总额", "金额", "total_amount"},
		"order_date":   {"订单日期", "下单日期", "order_date"},
	}
	result := map[string]int{}
	normalized := make([]string, len(headers))
	for i, h := range headers {
		normalized[i] = normalizeHeader(h)
	}
	for key, names := range aliases {
		for _, name := range names {
			target := normalizeHeader(name)
			for idx, header := range normalized {
				if header == target {
					result[key] = idx
					break
				}
			}
			if _, ok := result[key]; ok {
				break
			}
		}
	}
	return result
}

func normalizeHeader(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), " ", ""))
}

func parseInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int(f), nil
	}
	return strconv.Atoi(s)
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func parseFlexibleDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && f > 0 {
		return excelSerialDate(f), nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"2006/01/02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("请使用 YYYY-MM-DD 或 YYYY-MM-DD HH:mm")
}

func excelSerialDate(serial float64) time.Time {
	base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.Local)
	days := int(serial)
	fraction := serial - float64(days)
	return base.AddDate(0, 0, days).Add(time.Duration(fraction * float64(24*time.Hour)))
}

func readXLSXRows(file *multipart.FileHeader) ([][]string, error) {
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("无法读取 xlsx 文件")
	}

	sharedStrings, err := readSharedStrings(reader)
	if err != nil {
		return nil, err
	}
	sheetPath, err := firstSheetPath(reader)
	if err != nil {
		return nil, err
	}
	return readSheetRows(reader, sheetPath, sharedStrings)
}

func readSharedStrings(reader *zip.Reader) ([]string, error) {
	f := findZipFile(reader, "xl/sharedStrings.xml")
	if f == nil {
		return nil, nil
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	decoder := xml.NewDecoder(rc)
	var result []string
	var current strings.Builder
	inSI := false
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "si" {
				inSI = true
				current.Reset()
			}
		case xml.CharData:
			if inSI {
				current.Write([]byte(t))
			}
		case xml.EndElement:
			if t.Name.Local == "si" {
				inSI = false
				result = append(result, current.String())
			}
		}
	}
	return result, nil
}

func firstSheetPath(reader *zip.Reader) (string, error) {
	if findZipFile(reader, "xl/worksheets/sheet1.xml") != nil {
		return "xl/worksheets/sheet1.xml", nil
	}
	for _, f := range reader.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/") && strings.HasSuffix(f.Name, ".xml") {
			return f.Name, nil
		}
	}
	return "", fmt.Errorf("未找到工作表")
}

func readSheetRows(reader *zip.Reader, sheetPath string, sharedStrings []string) ([][]string, error) {
	f := findZipFile(reader, sheetPath)
	if f == nil {
		return nil, fmt.Errorf("未找到工作表")
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	decoder := xml.NewDecoder(rc)

	var rows [][]string
	var row []string
	var cellRef, cellType string
	var cellValue strings.Builder
	inRow := false
	inCell := false
	inValue := false

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "row":
				inRow = true
				row = nil
			case "c":
				inCell = true
				cellRef = attr(t, "r")
				cellType = attr(t, "t")
				cellValue.Reset()
			case "v", "t":
				if inCell {
					inValue = true
				}
			}
		case xml.CharData:
			if inValue {
				cellValue.Write([]byte(t))
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "v", "t":
				inValue = false
			case "c":
				col := columnIndex(cellRef)
				for len(row) <= col {
					row = append(row, "")
				}
				row[col] = resolveCellValue(cellValue.String(), cellType, sharedStrings)
				inCell = false
			case "row":
				if inRow {
					rows = append(rows, row)
				}
				inRow = false
			}
		}
	}
	return rows, nil
}

func resolveCellValue(raw, cellType string, sharedStrings []string) string {
	raw = strings.TrimSpace(raw)
	if cellType == "s" {
		idx, err := strconv.Atoi(raw)
		if err == nil && idx >= 0 && idx < len(sharedStrings) {
			return strings.TrimSpace(sharedStrings[idx])
		}
	}
	return raw
}

func findZipFile(reader *zip.Reader, name string) *zip.File {
	for _, f := range reader.File {
		if f.Name == name {
			return f
		}
	}
	return nil
}

func attr(el xml.StartElement, name string) string {
	for _, a := range el.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func columnIndex(ref string) int {
	if ref == "" {
		return 0
	}
	idx := 0
	for _, ch := range ref {
		if ch < 'A' || ch > 'Z' {
			break
		}
		idx = idx*26 + int(ch-'A'+1)
	}
	if idx == 0 {
		return 0
	}
	return idx - 1
}

func isEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}
