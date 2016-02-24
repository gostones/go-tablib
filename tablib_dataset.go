// Package tablib is a format-agnostic tabular Dataset library, written in Go.
// It allows you to import, export, and manipulate tabular data sets.
// Advanced features include, dynamic columns, tags & filtering, and seamless format import & export.
package tablib

import (
	"fmt"
	"sort"
	"strconv"
	"time"
)

// Dataset represents a set of data, which is a list of data and header for each column.
type Dataset struct {
	// EmptyValue represents the string value to b output if a field cannot be
	// formatted as a string during output of certain formats.
	EmptyValue string
	headers    []string
	data       [][]interface{}
	tags       [][]string
	rows       int
	cols       int
}

// DynamicColumn represents a function that can be evaluated dynamically
// when exporting to a predefined format.
type DynamicColumn func([]interface{}) interface{}

// NewDataset creates a new Dataset.
func NewDataset(headers []string) *Dataset {
	return NewDatasetWithData(headers, nil)
}

// NewDatasetWithData creates a new Dataset.
func NewDatasetWithData(headers []string, data [][]interface{}) *Dataset {
	d := &Dataset{"", headers, data, make([][]string, 0), len(data), len(headers)}
	return d
}

// Headers return the headers of the Dataset.
func (d *Dataset) Headers() []string {
	return d.headers
}

// Width returns the number of columns in the Dataset.
func (d *Dataset) Width() int {
	return d.cols
}

// Height returns the number of rows in the Dataset.
func (d *Dataset) Height() int {
	return d.rows
}

// Append appends a row of values to the Dataset.
func (d *Dataset) Append(row []interface{}) error {
	if len(row) != d.cols {
		return ErrInvalidDimensions
	}
	d.data = append(d.data, row)
	d.tags = append(d.tags, make([]string, 0))
	d.rows++
	return nil
}

// AppendTagged appends a row of values to the Dataset with one or multiple tags
// for filtering purposes.
func (d *Dataset) AppendTagged(row []interface{}, tags ...string) error {
	if err := d.Append(row); err != nil {
		return err
	}
	d.tags[d.rows-1] = tags[:]
	return nil
}

// AppendValues appends a row of values to the Dataset.
func (d *Dataset) AppendValues(row ...interface{}) error {
	return d.Append(row[:])
}

// AppendValuesTagged appends a row of values to the Dataset with one or multiple tags
// for filtering purposes.
func (d *Dataset) AppendValuesTagged(row ...interface{}) error {
	return d.AppendTagged(row[:])
}

// Insert inserts a row at a given index.
func (d *Dataset) Insert(index int, row []interface{}) error {
	if index < 0 || index >= d.rows {
		return ErrInvalidRowIndex
	}

	if len(row) != d.cols {
		return ErrInvalidDimensions
	}

	ndata := make([][]interface{}, 0, d.rows+1)
	ndata = append(ndata, d.data[:index]...)
	ndata = append(ndata, row)
	ndata = append(ndata, d.data[index:]...)
	d.data = ndata
	d.rows++

	ntags := make([][]string, 0, d.rows+1)
	ntags = append(ntags, d.tags[:index]...)
	ntags = append(ntags, make([]string, 0))
	ntags = append(ntags, d.tags[index:]...)
	d.tags = ntags

	return nil
}

// InsertValues inserts a row of values at a given index.
func (d *Dataset) InsertValues(index int, values ...interface{}) error {
	return d.Insert(index, values[:])
}

// InsertTagged inserts a row at a given index with specific tags.
func (d *Dataset) InsertTagged(index int, row []interface{}, tags ...string) error {
	if err := d.Insert(index, row); err != nil {
		return err
	}
	d.Insert(index, row)
	d.tags[index] = tags[:]

	return nil
}

// AppendColumn appends a new column with values to the Dataset.
func (d *Dataset) AppendColumn(header string, cols []interface{}) error {
	if len(cols) != d.rows {
		return ErrInvalidDimensions
	}
	d.headers = append(d.headers, header)
	d.cols++
	for i, e := range d.data {
		d.data[i] = append(e, cols[i])
	}
	return nil
}

// AppendColumnValues appends a new column with values to the Dataset.
func (d *Dataset) AppendColumnValues(header string, cols ...interface{}) error {
	return d.AppendColumn(header, cols[:])
}

// AppendDynamicColumn appends a dynamic column to the Dataset.
func (d *Dataset) AppendDynamicColumn(header string, fn DynamicColumn) {
	d.headers = append(d.headers, header)
	d.cols++
	for i, e := range d.data {
		d.data[i] = append(e, fn)
	}
}

// InsertColumn insert a new column at a given index.
func (d *Dataset) InsertColumn(index int, header string, cols []interface{}) error {
	if index < 0 || index >= d.cols {
		return ErrInvalidColumnIndex
	}

	if len(cols) != d.rows {
		return ErrInvalidDimensions
	}

	d.insertHeader(index, header)

	// for each row, insert the column
	for i, r := range d.data {
		row := make([]interface{}, 0, d.cols)
		row = append(row, r[:index]...)
		row = append(row, cols[i])
		row = append(row, r[index:]...)
		d.data[i] = row
	}

	return nil
}

// InsertDynamicColumn insert a new dynamic column at a given index.
func (d *Dataset) InsertDynamicColumn(index int, header string, fn DynamicColumn) error {
	if index < 0 || index >= d.cols {
		return ErrInvalidColumnIndex
	}

	d.insertHeader(index, header)

	// for each row, insert the column
	for i, r := range d.data {
		row := make([]interface{}, 0, d.cols)
		row = append(row, r[:index]...)
		row = append(row, fn)
		row = append(row, r[index:]...)
		d.data[i] = row
	}

	return nil
}

func (d *Dataset) insertHeader(index int, header string) {
	headers := make([]string, 0, d.cols+1)
	headers = append(headers, d.headers[:index]...)
	headers = append(headers, header)
	headers = append(headers, d.headers[index:]...)
	d.headers = headers
	d.cols++
}

// Stack stacks two Dataset by joining at the row level, and return new combined Dataset.
func (d *Dataset) Stack(other *Dataset) (*Dataset, error) {
	if d.Width() != other.Width() {
		return nil, ErrInvalidDimensions
	}

	nd := NewDataset(d.headers)
	nd.cols = d.cols
	nd.rows = d.rows + other.rows

	nd.tags = make([][]string, 0, nd.rows)
	nd.tags = append(nd.tags, d.tags...)
	nd.tags = append(nd.tags, other.tags...)

	nd.data = make([][]interface{}, 0, nd.rows)
	nd.data = append(nd.data, d.data...)
	nd.data = append(nd.data, other.data...)

	return nd, nil
}

// StackColumn stacks two Dataset by joining them at the column level, and return new combined Dataset.
func (d *Dataset) StackColumn(other *Dataset) (*Dataset, error) {
	if d.Height() != other.Height() {
		return nil, ErrInvalidDimensions
	}

	nheaders := d.headers
	nheaders = append(nheaders, other.headers...)

	nd := NewDataset(nheaders)
	nd.cols = d.cols + nd.cols
	nd.rows = d.rows
	nd.data = make([][]interface{}, nd.rows, nd.rows)
	nd.tags = make([][]string, nd.rows, nd.rows)

	for i := range d.data {
		nd.data[i] = make([]interface{}, 0, nd.cols)
		nd.data[i] = append(nd.data[i], d.data[i]...)
		nd.data[i] = append(nd.data[i], other.data[i]...)

		nd.tags[i] = make([]string, 0, nd.cols)
		nd.tags[i] = append(nd.tags[i], d.tags[i]...)
		nd.tags[i] = append(nd.tags[i], other.tags[i]...)
	}

	return nd, nil
}

// Column returns all the values for a specific column
// returns nil if column is not found.
func (d *Dataset) Column(header string) []interface{} {
	colIndex := indexOfColumn(header, d)
	if colIndex == -1 {
		return nil
	}

	values := make([]interface{}, d.rows)
	for i, e := range d.data {
		switch e[colIndex].(type) {
		case DynamicColumn:
			values[i] = e[colIndex].(DynamicColumn)(e)
		default:
			values[i] = e[colIndex]
		}
	}
	return values
}

// Row returns a map representing a specific row of the Dataset.
// returns tablib.ErrInvalidRowIndex if the row cannot be found
func (d *Dataset) Row(index int) (map[string]interface{}, error) {
	if index < 0 || index >= d.rows {
		return nil, ErrInvalidRowIndex
	}

	row := make(map[string]interface{})
	for i, e := range d.data[index] {
		switch e.(type) {
		case DynamicColumn:
			row[d.headers[i]] = e.(DynamicColumn)(d.data[index])
		default:
			row[d.headers[i]] = e
		}
	}
	return row, nil
}

// Rows returns an array of map representing a set of specific rows of the Dataset.
// returns tablib.ErrInvalidRowIndex if the row cannot be found.
func (d *Dataset) Rows(index ...int) ([]map[string]interface{}, error) {
	for _, i := range index {
		if i < 0 || i >= d.rows {
			return nil, ErrInvalidRowIndex
		}
	}

	rows := make([]map[string]interface{}, 0, len(index))
	for _, i := range index {
		row, _ := d.Row(i)
		rows = append(rows, row)
	}

	return rows, nil
}

// Slice returns a new Dataset representing a slice of the orignal Dataset like a slice of an array.
// returns tablib.ErrInvalidRowIndex if the lower or upper bound is out of range.
func (d *Dataset) Slice(lower, upperNonInclusive int) (*Dataset, error) {
	if lower > upperNonInclusive || lower < 0 || upperNonInclusive > d.rows {
		return nil, ErrInvalidRowIndex
	}

	rowCount := upperNonInclusive - lower
	cols := d.cols
	nd := NewDataset(d.headers)
	nd.data = make([][]interface{}, 0, rowCount)
	nd.tags = make([][]string, 0, rowCount)
	nd.rows = upperNonInclusive - lower
	j := 0
	for i := lower; i < upperNonInclusive; i++ {
		nd.data = append(nd.data, make([]interface{}, 0, cols))
		nd.data[j] = make([]interface{}, 0, cols)
		nd.data[j] = append(nd.data[j], d.data[i]...)
		nd.tags = append(nd.tags, make([]string, 0, cols))
		nd.tags[j] = make([]string, 0, cols)
		nd.tags[j] = append(nd.tags[j], d.tags[i]...)
		j++
	}

	return nd, nil
}

// Filter filters a Dataset, returning a fresh Dataset including only the rows
// previously tagged with one of the given tags. Returns a new Dataset.
func (d *Dataset) Filter(tags ...string) *Dataset {
	nd := NewDataset(d.headers)
	for rowIndex, rowValue := range d.data {
		for _, filterTag := range tags {
			if isTagged(filterTag, d.tags[rowIndex]) {
				nd.AppendTagged(rowValue, d.tags[rowIndex]...) // copy tags
			}
		}
	}
	return nd
}

// Sort sorts the Dataset by a specific column. Returns a new Dataset.
func (d *Dataset) Sort(column string) *Dataset {
	return d.internalSort(column, false)
}

// SortReverse sorts the Dataset by a specific column in reverse order. Returns a new Dataset.
func (d *Dataset) SortReverse(column string) *Dataset {
	return d.internalSort(column, true)
}

func (d *Dataset) internalSort(column string, reverse bool) *Dataset {
	nd := NewDataset(d.headers)
	pairs := make([]entryPair, 0, nd.rows)
	for i, v := range d.Column(column) {
		pairs = append(pairs, entryPair{i, v})
	}

	var how sort.Interface
	// sort by column
	switch pairs[0].value.(type) {
	case string:
		how = byStringValue(pairs)
	case int:
		how = byIntValue(pairs)
	case int64:
		how = byInt64Value(pairs)
	case uint64:
		how = byUint64Value(pairs)
	case float64:
		how = byFloatValue(pairs)
	case time.Time:
		how = byTimeValue(pairs)
	default:
		// nothing
	}

	if !reverse {
		sort.Sort(how)
	} else {
		sort.Sort(sort.Reverse(how))
	}

	// now iterate on the pairs and add the data sorted to the new Dataset
	for _, p := range pairs {
		nd.AppendTagged(d.data[p.index], d.tags[p.index]...)
	}

	return nd
}

// Transpose transposes a Dataset, turning rows into columns and vice versa,
// returning a new Dataset instance. The first row of the original instance
// becomes the new header row.
// TODO
func (d *Dataset) Transpose() *Dataset {
	newHeaders := make([]string, 0, d.cols+1)
	newHeaders = append(newHeaders, d.headers[0])
	for _, c := range d.Column(d.headers[0]) {
		newHeaders = append(newHeaders, d.asString(c))
	}

	//nd := NewDataset(newHeaders)

	panic("Transpose() not yet implemented")
}

// DeleteRow deletes a row at a specific index
func (d *Dataset) DeleteRow(row int) error {
	if row < 0 || row >= d.rows {
		return ErrInvalidRowIndex
	}
	d.data = append(d.data[:row], d.data[row+1:]...)
	d.rows--
	return nil
}

// DeleteColumn deletes a column from the Dataset.
func (d *Dataset) DeleteColumn(header string) error {
	colIndex := indexOfColumn(header, d)
	if colIndex == -1 {
		return ErrInvalidColumnIndex
	}
	d.cols--
	d.headers = append(d.headers[:colIndex], d.headers[colIndex+1:]...)
	// remove the column
	for i := range d.data {
		d.data[i] = append(d.data[i][:colIndex], d.data[i][colIndex+1:]...)
	}
	return nil
}

func indexOfColumn(header string, d *Dataset) int {
	for i, e := range d.headers {
		if e == header {
			return i
		}
	}
	return -1
}

// Dict returns the Dataset as an array of map where each key is a column.
func (d *Dataset) Dict() []interface{} {
	back := make([]interface{}, d.rows)
	for i, e := range d.data {
		m := make(map[string]interface{}, d.cols-1)
		for j, c := range d.headers {
			switch e[j].(type) {
			case DynamicColumn:
				m[c] = e[j].(DynamicColumn)(e)
			default:
				m[c] = e[j]
			}
		}
		back[i] = m
	}
	return back
}

// Records returns the Dataset as an array of array where each entry is a string.
// The first row of the returned 2d array represents the columns of the Dataset.
func (d *Dataset) Records() [][]string {
	records := make([][]string, d.rows+1 /* +1 for header */)
	records[0] = make([]string, d.cols)
	for j, e := range d.headers {
		records[0][j] = e
	}
	for i, e := range d.data {
		rowIndex := i + 1
		j := 0
		records[rowIndex] = make([]string, d.cols)
		for _, v := range e {
			vv := v
			switch v.(type) {
			case DynamicColumn:
				vv = v.(DynamicColumn)(e)
			default:
				// nothing
			}
			records[rowIndex][j] = d.asString(vv)
			j++
		}
	}

	return records
}

func isTagged(tag string, tags []string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (d *Dataset) asString(vv interface{}) string {
	var v string
	switch vv.(type) {
	case string:
		v = vv.(string)
	case int:
		v = strconv.Itoa(vv.(int))
	case int64:
		v = strconv.FormatInt(vv.(int64), 10)
	case uint64:
		v = strconv.FormatUint(vv.(uint64), 10)
	case bool:
		v = strconv.FormatBool(vv.(bool))
	case float64:
		v = strconv.FormatFloat(vv.(float64), 'G', -1, 32)
	case time.Time:
		v = vv.(time.Time).Format(time.RFC3339)
	default:
		if d.EmptyValue != "" {
			v = d.EmptyValue
		} else {
			v = fmt.Sprintf("%s", v)
		}
	}
	return v
}