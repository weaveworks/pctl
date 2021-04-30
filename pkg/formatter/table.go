package formatter

import (
	"bytes"
	"errors"

	"github.com/olekukonko/tablewriter"
)

type tableFormatter struct {
	table  *tablewriter.Table
	buffer *bytes.Buffer
}

// TableContents represents the contents of a table
type TableContents struct {
	Headers []string
	Data    [][]string
}

// NewTableFormatter returns a table formatter
func NewTableFormatter() tableFormatter {
	buf := &bytes.Buffer{}
	return tableFormatter{
		table:  newDefaultTable(buf),
		buffer: buf,
	}
}

// Format receives column names and table data and creates a table in the writer
func (f tableFormatter) Format(contentFunc func() interface{}) (string, error) {
	contents, ok := contentFunc().(TableContents)
	if !ok {
		return "", errors.New("func returned wrong type for table formatter. wanted formatter.TableContents")
	}

	f.table.SetHeader(contents.Headers)
	f.table.AppendBulk(contents.Data)
	f.table.Render()

	return f.buffer.String(), nil
}

func newDefaultTable(buf *bytes.Buffer) *tablewriter.Table {
	table := tablewriter.NewWriter(buf)
	table.SetAutoWrapText(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoFormatHeaders(true)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)
	return table
}
