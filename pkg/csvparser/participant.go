package csvparser

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/usecase/participant"
)

// ParsedInput holds a parsed CSV row with its original 1-based row number.
type ParsedInput struct {
	Row   int
	Input participant.CreateParticipantInput
}

// RowError holds a parse error for a specific CSV row.
type RowError struct {
	Row     int
	Email   string
	Message string
}

// utf8BOMLen is the byte length of the UTF-8 BOM sequence (0xEF 0xBB 0xBF).
const utf8BOMLen = 3

// stripBOM returns a reader with the UTF-8 BOM (0xEF 0xBB 0xBF) removed if present.
func stripBOM(r io.Reader) io.Reader {
	br := bufio.NewReader(r)
	bs, err := br.Peek(utf8BOMLen)
	if err == nil && bs[0] == 0xEF && bs[1] == 0xBB && bs[2] == 0xBF {
		_, _ = br.Discard(utf8BOMLen)
	}
	return br
}

// ParseParticipantCSV parses a CSV reader into participant inputs.
// Returns (parsedInputs, rowErrors, fileError).
// fileError is non-nil for structural issues (empty file, missing required columns).
// rowErrors collects per-row parse issues; valid rows are still returned in parsedInputs.
func ParseParticipantCSV(r io.Reader) ([]ParsedInput, []RowError, error) {
	reader := csv.NewReader(stripBOM(r))
	reader.TrimLeadingSpace = true

	colIndex, err := readAndValidateHeader(reader)
	if err != nil {
		return nil, nil, err
	}

	return readDataRows(reader, colIndex)
}

// readAndValidateHeader reads the CSV header and validates required columns.
func readAndValidateHeader(reader *csv.Reader) (map[string]int, error) {
	headers, err := reader.Read()
	if err == io.EOF {
		return nil, fmt.Errorf("CSV file is empty")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	colIndex := buildColumnIndex(headers)

	if _, ok := colIndex["name"]; !ok {
		return nil, fmt.Errorf("required column 'name' not found in CSV header")
	}
	if _, ok := colIndex["email"]; !ok {
		return nil, fmt.Errorf("required column 'email' not found in CSV header")
	}

	return colIndex, nil
}

// readDataRows reads all data rows from the CSV and returns parsed inputs and row errors.
func readDataRows(reader *csv.Reader, colIndex map[string]int) ([]ParsedInput, []RowError, error) {
	var inputs []ParsedInput
	var rowErrors []RowError
	csvRowNum := 1 // 1 for header row

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		csvRowNum++
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read CSV row %d: %w", csvRowNum, err)
		}

		dataRowNum := csvRowNum - 1 // 1-based data row number (excluding header)
		email := getField(colIndex, row, "email")
		input, parseErr := parseRow(colIndex, row)
		if parseErr != nil {
			rowErrors = append(rowErrors, RowError{
				Row:     dataRowNum,
				Email:   email,
				Message: parseErr.Error(),
			})
			continue
		}
		inputs = append(inputs, ParsedInput{Row: dataRowNum, Input: input})
	}

	if len(inputs) == 0 && len(rowErrors) == 0 {
		return nil, nil, fmt.Errorf("CSV file contains no data rows")
	}

	return inputs, rowErrors, nil
}

// buildColumnIndex maps lowercase column name to index in header slice.
func buildColumnIndex(headers []string) map[string]int {
	index := make(map[string]int, len(headers))
	for i, h := range headers {
		index[h] = i
	}
	return index
}

// getField returns the value at the named column, or "" if out of range or column not present.
func getField(colIndex map[string]int, row []string, name string) string {
	i, ok := colIndex[name]
	if !ok || i >= len(row) {
		return ""
	}
	return row[i]
}

// ptrStr returns a *string for a non-empty value, nil otherwise.
func ptrStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// StatusFormat controls how participant status is rendered in CSV export.
type StatusFormat int

const (
	// StatusFormatEnglish outputs English strings ("confirmed", "tentative", etc.). Default.
	StatusFormatEnglish StatusFormat = iota
	// StatusFormatJapanese outputs Japanese symbols (○, △, ×).
	StatusFormatJapanese
)

// csvExportHeaders defines the column order for CSV export.
var csvExportHeaders = []string{
	"id", "name", "email", "employee_id", "phone", "qr_email",
	"status", "qr_code", "qr_code_generated_at", "qr_distribution_url",
	"payment_status", "payment_amount", "payment_date",
	"checked_in", "checked_in_at", "metadata", "created_at", "updated_at",
}

// ExportParticipantCSV writes participants as CSV to w.
// Optional fields are written as empty strings when nil.
// Timestamps are formatted as RFC3339 UTC.
// format controls whether status is written as English strings or Japanese ○△× symbols.
func ExportParticipantCSV(w io.Writer, participants []*entity.Participant, format StatusFormat) error {
	writer := csv.NewWriter(w)

	if err := writer.Write(csvExportHeaders); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	for _, p := range participants {
		if err := writer.Write(participantToCSVRow(p, format)); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	return writer.Error()
}

func participantToCSVRow(p *entity.Participant, format StatusFormat) []string {
	statusStr := string(p.Status)
	if format == StatusFormatJapanese {
		statusStr = participantStatusToSymbol(p.Status)
	}
	return []string{
		p.ID.String(),
		p.Name,
		p.Email,
		derefStr(p.EmployeeID),
		derefStr(p.Phone),
		derefStr(p.QREmail),
		statusStr,
		p.QRCode,
		p.QRCodeGeneratedAt.UTC().Format(time.RFC3339),
		p.QRDistributionURL,
		string(p.PaymentStatus),
		formatExportFloat(p.PaymentAmount),
		formatExportTime(p.PaymentDate),
		strconv.FormatBool(p.CheckedIn),
		formatExportTime(p.CheckedInAt),
		formatExportMetadata(p.Metadata),
		p.CreatedAt.UTC().Format(time.RFC3339),
		p.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func formatExportFloat(f *float64) string {
	if f == nil {
		return ""
	}
	return strconv.FormatFloat(*f, 'f', -1, 64)
}

func formatExportTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func formatExportMetadata(m *json.RawMessage) string {
	if m == nil {
		return ""
	}
	return string(*m)
}

// parseRow converts a CSV row into a CreateParticipantInput.
func parseRow(colIndex map[string]int, row []string) (participant.CreateParticipantInput, error) {
	input := participant.CreateParticipantInput{
		Name:          getField(colIndex, row, "name"),
		Email:         getField(colIndex, row, "email"),
		EmployeeID:    ptrStr(getField(colIndex, row, "employee_id")),
		Phone:         ptrStr(getField(colIndex, row, "phone")),
		Status:        entity.ParticipantStatusTentative,
		PaymentStatus: entity.PaymentUnpaid,
	}

	if s := getField(colIndex, row, "status"); s != "" {
		input.Status = normalizeParticipantStatus(s)
	}
	if s := getField(colIndex, row, "payment_status"); s != "" {
		input.PaymentStatus = entity.PaymentStatus(s)
	}

	if s := getField(colIndex, row, "payment_amount"); s != "" {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return participant.CreateParticipantInput{}, fmt.Errorf("invalid payment_amount %q: %w", s, err)
		}
		input.PaymentAmount = &v
	}

	if s := getField(colIndex, row, "payment_date"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			err := fmt.Errorf("invalid payment_date %q: expected RFC3339 format (e.g. 2025-11-08T12:30:00Z)", s)
			return participant.CreateParticipantInput{}, err
		}
		input.PaymentDate = &t
	}

	if s := getField(colIndex, row, "metadata"); s != "" {
		input.Metadata = &s
	}

	return input, nil
}

// normalizeParticipantStatus maps Japanese symbols to ParticipantStatus values.
// ○ → confirmed, △ → tentative, × → cancelled
func normalizeParticipantStatus(s string) entity.ParticipantStatus {
	switch s {
	case "○":
		return entity.ParticipantStatusConfirmed
	case "△":
		return entity.ParticipantStatusTentative
	case "×":
		return entity.ParticipantStatusCancelled
	default:
		return entity.ParticipantStatus(s)
	}
}

// participantStatusToSymbol converts a ParticipantStatus to a Japanese symbol.
// confirmed → ○, tentative → △, cancelled/declined → ×
func participantStatusToSymbol(s entity.ParticipantStatus) string {
	switch s {
	case entity.ParticipantStatusConfirmed:
		return "○"
	case entity.ParticipantStatusTentative:
		return "△"
	case entity.ParticipantStatusCancelled, entity.ParticipantStatusDeclined:
		return "×"
	default:
		return string(s)
	}
}
