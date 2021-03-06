package clitable

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

var (
	defaultTableStyle = &TableStyle{
		VerticalBorder:   "|",
		HorizontalBorder: "-",
		Corner:           "+",
	}
)

type TableStyle struct {
	VerticalBorder   string
	HorizontalBorder string
	Corner           string
}

type Table struct {
	columns    []*Column
	columnsMap map[string]*Column
	Style      *TableStyle
	rows       []*Row
}

func NewTable(names ...interface{}) *Table {
	table := &Table{
		Style:      defaultTableStyle,
		columns:    make([]*Column, len(names)),
		columnsMap: make(map[string]*Column),
		rows:       make([]*Row, 0),
	}
	for i, rawName := range names {
		name := rawName.(string)
		column := NewColumn(name)
		table.columns[i] = column
		table.columnsMap[name] = column
	}
	table.addHeader(names...)
	return table
}

func (t *Table) addHeader(datas ...interface{}) {
	row := NewRow()
	row.isHeader = true
	t.addRow(row, datas...)
}

func (t *Table) AddRow(datas ...interface{}) {
	row := NewRow()
	t.addRow(row, datas...)
}

func (t *Table) addRow(row *Row, datas ...interface{}) {
	var data interface{}
	datasLen := len(datas)
	for i, _ := range t.columns {
		if i >= 0 && i < datasLen {
			data = datas[i]
		} else {
			data = ""
		}
		cell := NewCell(data)
		row.cells = append(row.cells, cell)
	}
	t.rows = append(t.rows, row)
}

func (t *Table) getVerticalBorderWidth() int {
	return utf8.RuneCountInString(t.Style.VerticalBorder)
}

func (t *Table) Print() {
	fmt.Print(t.String())
}

func (t *Table) String() string {
	cornerWidth := utf8.RuneCountInString(t.Style.Corner)
	verticalBorderWidth := t.getVerticalBorderWidth()

	for _, row := range t.rows {
		for i, cell := range row.cells {
			column := t.columns[i]
			style := column.getStyleByRow(row)
			columnWidth := cell.width + style.PaddingLeft + style.PaddingRight
			if column.width < columnWidth {
				column.width = columnWidth
			}
		}
	}

	maxRowWidth := 0
	for _, column := range t.columns {
		maxRowWidth += column.width
	}

	fullRowWidth := maxRowWidth + verticalBorderWidth*(len(t.columns)+1)
	winCol := int(WinSize.Col)

	if fullRowWidth > winCol && winCol > 0 {
		excess := float64(fullRowWidth-winCol) + 5
		maxExcess := excess
		columnsCount := len(t.columns)
		meanColumnWidth := float64(maxRowWidth) / float64(columnsCount)
		maxWidth := 0
		maxWidthColumn := 0
		var currentRate float64
		for i, column := range t.columns {
			rate := (100 * float64(column.width)) / float64(maxRowWidth)
			currentRate += rate
			if float64(column.width)+maxExcess-excess > meanColumnWidth {
				excessColumn := excess * currentRate / 100
				column.width -= int(math.Floor(excessColumn))
				excess -= excessColumn
			}
			if maxWidth < column.width {
				maxWidth = column.width
				maxWidthColumn = i
			}
		}
		if excess > 0 {
			t.columns[maxWidthColumn].width -= int(math.Floor(excess))
		}
		for _, row := range t.rows {
			for i, cell := range row.cells {
				column := t.columns[i]
				style := column.getStyleByRow(row)
				if cell.width > column.width {
					columnWidth := column.width - (style.PaddingLeft + style.PaddingRight)
					srcParts := strings.Split(cell.data, WS)
					srcPartsLen := len(srcParts)
					lastStrPart := srcPartsLen - 1
					dstParts := make([]string, 0)
					cellBuf := new(bytes.Buffer)
					for j := 0; j < srcPartsLen; j++ {
						srcPart := srcParts[j]
						srcPartLen := utf8.RuneCountInString(srcPart)
						if srcPartLen > columnWidth {
							dstParts = append(dstParts, srcPart[0:column.width-1])
						} else {
							cellBufNextLen := utf8.RuneCount(cellBuf.Bytes()) + srcPartLen
							if cellBufNextLen < columnWidth {
								if cellBufNextLen+1 < columnWidth {
									cellBuf.WriteString(srcPart)
									cellBuf.WriteString(WS)
								} else {
									cellBuf.WriteString(srcPart)
								}
							} else {
								dstParts = append(dstParts, strings.TrimRight(cellBuf.String(), WS))
								cellBuf.Reset()
								cellBuf.WriteString(srcPart)
								cellBuf.WriteString(WS)
							}
						}
						if j == lastStrPart && cellBuf.Len() > 0 {
							dstParts = append(dstParts, strings.TrimRight(cellBuf.String(), WS))
						}
					}
					dstPartsLen := len(dstParts)
					nextHeight := dstPartsLen + style.PaddingTop + style.PaddingBottom
					if dstPartsLen > 1 {
						cell.parts = dstParts
						cell.partsLen = dstPartsLen
					}
					if nextHeight > row.height {
						row.height = nextHeight
					}
				} else {
					nextHeight := style.PaddingTop + style.PaddingBottom + 1
					if nextHeight > row.height {
						row.height = nextHeight
					}
				}
			}
		}
	}

	buf := new(bytes.Buffer)
	for _, row := range t.rows {
		t.writeLine(buf, cornerWidth, verticalBorderWidth)
		for x := 0; x < row.height; x++ {
			for i, cell := range row.cells {
				column := t.columns[i]
				style := column.getStyleByRow(row)
				buf.WriteString(t.Style.VerticalBorder)
				columnWidth := column.width - (style.PaddingLeft + style.PaddingRight)
				if x < style.PaddingTop || x > row.height-style.PaddingBottom {
					buf.WriteString(t.createEmptyLine(columnWidth))
				} else {
					t.writeHorizontalPadding(buf, style.PaddingLeft)
					if cell.partsLen > 0 {
						var start int
						switch style.VerticalAlign {
						case ColumnVerticalAlignTop:
							start += style.PaddingTop
						case ColumnVerticalAlignMiddle:
							start = (row.height-cell.partsLen)/2 + style.PaddingTop
						case ColumnVerticalAlignBottom:
							start = row.height - cell.partsLen
						}
						end := cell.partsLen + start
						if x >= start && x < end {
							j := x - start
							t.writeCell(buf, columnWidth, utf8.RuneCountInString(cell.parts[j]), cell.parts[j], style)
						} else {
							buf.WriteString(t.createEmptyLine(columnWidth))
						}
					} else {
						var j int
						switch style.VerticalAlign {
						case ColumnVerticalAlignTop:
							j = style.PaddingTop
						case ColumnVerticalAlignMiddle:
							j = (row.height-(style.PaddingTop+style.PaddingBottom))/2 + style.PaddingTop
						case ColumnVerticalAlignBottom:
							j = row.height - 1 - style.PaddingBottom
						}
						if x == j {
							t.writeCell(buf, columnWidth, cell.width, cell.data, style)
						} else {
							buf.WriteString(t.createEmptyLine(columnWidth))
						}
					}
					t.writeHorizontalPadding(buf, style.PaddingRight)
				}
			}
			buf.WriteString(t.Style.VerticalBorder)
			buf.Write(EOL)
		}
	}
	t.writeLine(buf, cornerWidth, verticalBorderWidth)
	return buf.String()
}

func (t *Table) writeLine(buf *bytes.Buffer, cornerWidth, verticalBorderWidth int) {
	for _, column := range t.columns {
		buf.WriteString(t.Style.Corner)
		buf.WriteString(
			strings.Repeat(
				t.Style.HorizontalBorder,
				verticalBorderWidth+column.width-cornerWidth,
			),
		)
	}
	buf.WriteString(t.Style.Corner)
	buf.Write(EOL)
}

func (t *Table) writeHorizontalPadding(buf *bytes.Buffer, width int) {
	buf.WriteString(t.createEmptyLine(width))
}

func (t *Table) writeCell(buf *bytes.Buffer, columnWidth, cellWidth int, data string, style *ColumnStyle) {
	isWriteWhiteSpace := columnWidth > cellWidth
	diff := columnWidth - cellWidth
	switch style.Align {
	case ColumnAlignLeft:
		buf.WriteString(data)
		if isWriteWhiteSpace {
			buf.WriteString(strings.Repeat(WS, diff))
		}
	case ColumnAlignCenter:
		side := diff / 2
		if isWriteWhiteSpace {
			buf.WriteString(strings.Repeat(WS, side))
		}
		buf.WriteString(data)
		if isWriteWhiteSpace {
			buf.WriteString(strings.Repeat(WS, diff-side))
		}
	case ColumnAlignRight:
		if isWriteWhiteSpace {
			buf.WriteString(strings.Repeat(WS, diff))
		}
		buf.WriteString(data)
	}
}

func (t *Table) createEmptyLine(width int) string {
	return strings.Repeat(WS, width)
}

func (t *Table) GetColumnByNum(i int) *Column {
	if i >= 0 && i <= len(t.columns)-1 {
		return t.columns[i]
	} else {
		return nil
	}
}

func (t *Table) GetColumnByName(name string) *Column {
	if column, ok := t.columnsMap[name]; ok {
		return column
	} else {
		return nil
	}
}

func (t *Table) Clean() {
	t.rows = append([]*Row{}, t.rows[0])
}
