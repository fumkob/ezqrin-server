package csvparser

import (
	"encoding/csv"
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

// ParseParticipantCSV parses a CSV reader into participant inputs.
// Returns (parsedInputs, rowErrors, fileError).
// fileError is non-nil for structural issues (empty file, missing required columns).
// rowErrors collects per-row parse issues; valid rows are still returned in parsedInputs.
func ParseParticipantCSV(r io.Reader) ([]ParsedInput, []RowError, error) {
	reader := csv.NewReader(r)
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
		input.Status = entity.ParticipantStatus(s)
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
