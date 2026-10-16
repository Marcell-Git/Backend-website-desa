package report

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"

	"rt26/backend/internal/pkg/helper"
)

func generatePDF(title string, headers []string, rows [][]string, summary string) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 10, "RT 26 Kelurahan Tegalyoso", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, "Kecamatan Klaten Selatan, Kabupaten Klaten", "", 1, "C", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, title, "", 1, "C", false, 0, "")
	pdf.Ln(2)

	periodText := buildPeriodText()
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(0, 6, periodText, "", 1, "C", false, 0, "")
	pdf.Ln(4)

	colWidths := make([]float64, len(headers))
	availW := 190.0
	if len(headers) > 0 {
		w := availW / float64(len(headers))
		for i := range colWidths {
			colWidths[i] = w
		}
		if len(headers) > 0 {
			colWidths[0] = 30
			remaining := availW - 30
			if len(headers) > 1 {
				each := remaining / float64(len(headers)-1)
				for i := 1; i < len(headers); i++ {
					colWidths[i] = each
				}
			}
		}
	}

	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetFillColor(230, 230, 230)
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Helvetica", "", 8)
	for _, row := range rows {
		if pdf.GetY() > 260 {
			pdf.AddPage()
			pdf.SetFont("Helvetica", "B", 8)
			pdf.SetFillColor(230, 230, 230)
			for i, h := range headers {
				pdf.CellFormat(colWidths[i], 7, h, "1", 0, "C", true, 0, "")
			}
			pdf.Ln(-1)
			pdf.SetFont("Helvetica", "", 8)
		}
		for i, cell := range row {
			align := "L"
			if i == len(row)-1 || (i > 0 && len(row) > 2) {
				align = "R"
			}
			pdf.CellFormat(colWidths[i], 6, cell, "1", 0, align, false, 0, "")
		}
		pdf.Ln(-1)
	}

	if summary != "" {
		pdf.Ln(3)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(0, 7, summary, "", 1, "L", false, 0, "")
	}

	pdf.Ln(8)
	pdf.SetFont("Helvetica", "I", 8)
	now := helper.FormatTanggalWaktu(time.Now())
	pdf.CellFormat(0, 5, fmt.Sprintf("Dicetak pada: %s", now), "", 1, "R", false, 0, "")

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildPeriodText() string {
	return "Periode Laporan"
}
