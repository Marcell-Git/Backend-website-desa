package report

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
)

func generateExcel(sheetName string, headers []string, rows [][]string, summaryRow []string) ([]byte, error) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	f.SetSheetName(sheet, sheetName)
	sheet = sheetName

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	subtitleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 9, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"059669"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})
	dataStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 9},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	summaryStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"F3F4F6"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})

	f.MergeCell(sheet, "A1", "A1")
	f.SetCellValue(sheet, "A1", "RT 26 Kelurahan Tegalyoso")
	f.SetCellStyle(sheet, "A1", "A1", titleStyle)

	f.SetCellValue(sheet, "A2", "Kecamatan Klaten Selatan, Kabupaten Klaten")
	f.SetCellStyle(sheet, "A2", "A2", subtitleStyle)

	f.SetCellValue(sheet, "A4", sheetName)
	f.SetCellStyle(sheet, "A4", "A4", titleStyle)

	startRow := 6
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, startRow)
		f.SetCellValue(sheet, cell, h)
	}
	endHeader, _ := excelize.CoordinatesToCellName(len(headers), startRow)
	f.SetCellStyle(sheet, "A"+fmt.Sprintf("%d", startRow), endHeader, headerStyle)

	for ri, row := range rows {
		r := startRow + 1 + ri
		for ci, val := range row {
			cell, _ := excelize.CoordinatesToCellName(ci+1, r)
			f.SetCellValue(sheet, cell, val)
		}
		endData, _ := excelize.CoordinatesToCellName(len(headers), r)
		f.SetCellStyle(sheet, "A"+fmt.Sprintf("%d", r), endData, dataStyle)
	}

	if len(summaryRow) > 0 {
		sr := startRow + 1 + len(rows)
		for ci, val := range summaryRow {
			cell, _ := excelize.CoordinatesToCellName(ci+1, sr)
			f.SetCellValue(sheet, cell, val)
		}
		endSummary, _ := excelize.CoordinatesToCellName(len(headers), sr)
		f.SetCellStyle(sheet, "A"+fmt.Sprintf("%d", sr), endSummary, summaryStyle)
	}

	for i := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, 18)
	}

	var buf bytes.Buffer
	err := f.Write(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
