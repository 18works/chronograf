package execute

import (
	"fmt"
	"sort"
	"sync/atomic"

	"github.com/influxdata/platform/query"
	"github.com/influxdata/platform/query/values"
)

const (
	DefaultStartColLabel = "_start"
	DefaultStopColLabel  = "_stop"
	DefaultTimeColLabel  = "_time"
	DefaultValueColLabel = "_value"
)

func PartitionKeyForRow(i int, cr query.ColReader) query.PartitionKey {
	key := cr.Key()
	cols := cr.Cols()
	colsCpy := make([]query.ColMeta, 0, len(cols))
	vs := make([]values.Value, 0, len(cols))
	for j, c := range cols {
		if !key.HasCol(c.Label) {
			continue
		}
		colsCpy = append(colsCpy, c)
		switch c.Type {
		case query.TBool:
			vs = append(vs, values.NewBoolValue(cr.Bools(j)[i]))
		case query.TInt:
			vs = append(vs, values.NewIntValue(cr.Ints(j)[i]))
		case query.TUInt:
			vs = append(vs, values.NewUIntValue(cr.UInts(j)[i]))
		case query.TFloat:
			vs = append(vs, values.NewFloatValue(cr.Floats(j)[i]))
		case query.TString:
			vs = append(vs, values.NewStringValue(cr.Strings(j)[i]))
		case query.TTime:
			vs = append(vs, values.NewTimeValue(cr.Times(j)[i]))
		}
	}
	return &partitionKey{
		cols:   colsCpy,
		values: vs,
	}
}

func PartitionKeyForRowOn(i int, cr query.ColReader, on map[string]bool) query.PartitionKey {
	cols := make([]query.ColMeta, 0, len(on))
	vs := make([]values.Value, 0, len(on))
	for j, c := range cr.Cols() {
		if !on[c.Label] {
			continue
		}
		cols = append(cols, c)
		switch c.Type {
		case query.TBool:
			vs = append(vs, values.NewBoolValue(cr.Bools(j)[i]))
		case query.TInt:
			vs = append(vs, values.NewIntValue(cr.Ints(j)[i]))
		case query.TUInt:
			vs = append(vs, values.NewUIntValue(cr.UInts(j)[i]))
		case query.TFloat:
			vs = append(vs, values.NewFloatValue(cr.Floats(j)[i]))
		case query.TString:
			vs = append(vs, values.NewStringValue(cr.Strings(j)[i]))
		case query.TTime:
			vs = append(vs, values.NewTimeValue(cr.Times(j)[i]))
		}
	}
	return NewPartitionKey(cols, vs)
}

// OneTimeBlock is a Block that permits reading data only once.
// Specifically the ValueIterator may only be consumed once from any of the columns.
type OneTimeBlock interface {
	query.Block
	onetime()
}

// CacheOneTimeBlock returns a block that can be read multiple times.
// If the block is not a OneTimeBlock it is returned directly.
// Otherwise its contents are read into a new block.
func CacheOneTimeBlock(b query.Block, a *Allocator) query.Block {
	_, ok := b.(OneTimeBlock)
	if !ok {
		return b
	}
	return CopyBlock(b, a)
}

// CopyBlock returns a copy of the block and is OneTimeBlock safe.
func CopyBlock(b query.Block, a *Allocator) query.Block {
	builder := NewColListBlockBuilder(b.Key(), a)

	cols := b.Cols()
	colMap := make([]int, len(cols))
	for j, c := range cols {
		colMap[j] = j
		builder.AddCol(c)
	}

	AppendBlock(b, builder, colMap)
	// ColListBlockBuilders do not error
	nb, _ := builder.Block()
	return nb
}

// AddBlockCols adds the columns of b onto builder.
func AddBlockCols(b query.Block, builder BlockBuilder) {
	cols := b.Cols()
	for _, c := range cols {
		builder.AddCol(c)
	}
}

func AddBlockKeyCols(key query.PartitionKey, builder BlockBuilder) {
	for _, c := range key.Cols() {
		builder.AddCol(c)
	}
}

// AddNewCols adds the columns of b onto builder that did not already exist.
// Returns the mapping of builder cols to block cols.
func AddNewCols(b query.Block, builder BlockBuilder) []int {
	cols := b.Cols()
	existing := builder.Cols()
	colMap := make([]int, len(existing))
	for j, c := range cols {
		found := false
		for ej, ec := range existing {
			if c.Label == ec.Label {
				colMap[ej] = j
				found = true
				break
			}
		}
		if !found {
			builder.AddCol(c)
			colMap = append(colMap, j)
		}
	}
	return colMap
}

// AppendBlock append data from block b onto builder.
// The colMap is a map of builder column index to block column index.
func AppendBlock(b query.Block, builder BlockBuilder, colMap []int) {
	if len(b.Cols()) == 0 {
		return
	}

	b.Do(func(cr query.ColReader) error {
		AppendCols(cr, builder, colMap)
		return nil
	})
}

// AppendCols appends all columns from cr onto builder.
// The colMap is a map of builder column index to cr column index.
func AppendCols(cr query.ColReader, builder BlockBuilder, colMap []int) {
	for j := range builder.Cols() {
		AppendCol(j, colMap[j], cr, builder)
	}
}

// AppendCol append a column from cr onto builder
// The indexes bj and cj are builder and col reader indexes respectively.
func AppendCol(bj, cj int, cr query.ColReader, builder BlockBuilder) {
	c := cr.Cols()[cj]
	switch c.Type {
	case query.TBool:
		builder.AppendBools(bj, cr.Bools(cj))
	case query.TInt:
		builder.AppendInts(bj, cr.Ints(cj))
	case query.TUInt:
		builder.AppendUInts(bj, cr.UInts(cj))
	case query.TFloat:
		builder.AppendFloats(bj, cr.Floats(cj))
	case query.TString:
		builder.AppendStrings(bj, cr.Strings(cj))
	case query.TTime:
		builder.AppendTimes(bj, cr.Times(cj))
	default:
		PanicUnknownType(c.Type)
	}
}

// AppendMappedRecord appends the record from cr onto builder assuming matching columns.
func AppendRecord(i int, cr query.ColReader, builder BlockBuilder) {
	for j, c := range builder.Cols() {
		switch c.Type {
		case query.TBool:
			builder.AppendBool(j, cr.Bools(j)[i])
		case query.TInt:
			builder.AppendInt(j, cr.Ints(j)[i])
		case query.TUInt:
			builder.AppendUInt(j, cr.UInts(j)[i])
		case query.TFloat:
			builder.AppendFloat(j, cr.Floats(j)[i])
		case query.TString:
			builder.AppendString(j, cr.Strings(j)[i])
		case query.TTime:
			builder.AppendTime(j, cr.Times(j)[i])
		default:
			PanicUnknownType(c.Type)
		}
	}
}

// AppendMappedRecord appends the records from cr onto builder, using colMap as a map of builder index to cr index.
func AppendMappedRecord(i int, cr query.ColReader, builder BlockBuilder, colMap []int) {
	for j, c := range builder.Cols() {
		switch c.Type {
		case query.TBool:
			builder.AppendBool(j, cr.Bools(colMap[j])[i])
		case query.TInt:
			builder.AppendInt(j, cr.Ints(colMap[j])[i])
		case query.TUInt:
			builder.AppendUInt(j, cr.UInts(colMap[j])[i])
		case query.TFloat:
			builder.AppendFloat(j, cr.Floats(colMap[j])[i])
		case query.TString:
			builder.AppendString(j, cr.Strings(colMap[j])[i])
		case query.TTime:
			builder.AppendTime(j, cr.Times(colMap[j])[i])
		default:
			PanicUnknownType(c.Type)
		}
	}
}

// AppendRecordForCols appends the only the columns provided from cr onto builder.
func AppendRecordForCols(i int, cr query.ColReader, builder BlockBuilder, cols []query.ColMeta) {
	for j, c := range cols {
		switch c.Type {
		case query.TBool:
			builder.AppendBool(j, cr.Bools(j)[i])
		case query.TInt:
			builder.AppendInt(j, cr.Ints(j)[i])
		case query.TUInt:
			builder.AppendUInt(j, cr.UInts(j)[i])
		case query.TFloat:
			builder.AppendFloat(j, cr.Floats(j)[i])
		case query.TString:
			builder.AppendString(j, cr.Strings(j)[i])
		case query.TTime:
			builder.AppendTime(j, cr.Times(j)[i])
		default:
			PanicUnknownType(c.Type)
		}
	}
}

func AppendKeyValues(key query.PartitionKey, builder BlockBuilder) {
	for j, c := range key.Cols() {
		idx := ColIdx(c.Label, builder.Cols())
		switch c.Type {
		case query.TBool:
			builder.AppendBool(idx, key.ValueBool(j))
		case query.TInt:
			builder.AppendInt(idx, key.ValueInt(j))
		case query.TUInt:
			builder.AppendUInt(idx, key.ValueUInt(j))
		case query.TFloat:
			builder.AppendFloat(idx, key.ValueFloat(j))
		case query.TString:
			builder.AppendString(idx, key.ValueString(j))
		case query.TTime:
			builder.AppendTime(idx, key.ValueTime(j))
		default:
			PanicUnknownType(c.Type)
		}
	}
}

func ContainsStr(strs []string, str string) bool {
	for _, s := range strs {
		if str == s {
			return true
		}
	}
	return false
}

func ColIdx(label string, cols []query.ColMeta) int {
	for j, c := range cols {
		if c.Label == label {
			return j
		}
	}
	return -1
}
func HasCol(label string, cols []query.ColMeta) bool {
	return ColIdx(label, cols) >= 0
}

// BlockBuilder builds blocks that can be used multiple times
type BlockBuilder interface {
	Key() query.PartitionKey

	NRows() int
	NCols() int
	Cols() []query.ColMeta

	// AddCol increases the size of the block by one column.
	// The index of the column is returned.
	AddCol(query.ColMeta) int

	// Set sets the value at the specified coordinates
	// The rows and columns must exist before calling set, otherwise Set panics.
	SetBool(i, j int, value bool)
	SetInt(i, j int, value int64)
	SetUInt(i, j int, value uint64)
	SetFloat(i, j int, value float64)
	SetString(i, j int, value string)
	SetTime(i, j int, value Time)

	AppendBool(j int, value bool)
	AppendInt(j int, value int64)
	AppendUInt(j int, value uint64)
	AppendFloat(j int, value float64)
	AppendString(j int, value string)
	AppendTime(j int, value Time)

	AppendBools(j int, values []bool)
	AppendInts(j int, values []int64)
	AppendUInts(j int, values []uint64)
	AppendFloats(j int, values []float64)
	AppendStrings(j int, values []string)
	AppendTimes(j int, values []Time)

	// Sort the rows of the by the values of the columns in the order listed.
	Sort(cols []string, desc bool)

	// Clear removes all rows, while preserving the column meta data.
	ClearData()

	// Block returns the block that has been built.
	// Further modifications of the builder will not effect the returned block.
	Block() (query.Block, error)
}

type ColListBlockBuilder struct {
	blk   *ColListBlock
	alloc *Allocator
}

func NewColListBlockBuilder(key query.PartitionKey, a *Allocator) *ColListBlockBuilder {
	return &ColListBlockBuilder{
		blk:   &ColListBlock{key: key},
		alloc: a,
	}
}

func (b ColListBlockBuilder) Key() query.PartitionKey {
	return b.blk.Key()
}

func (b ColListBlockBuilder) NRows() int {
	return b.blk.nrows
}
func (b ColListBlockBuilder) NCols() int {
	return len(b.blk.cols)
}
func (b ColListBlockBuilder) Cols() []query.ColMeta {
	return b.blk.colMeta
}

func (b ColListBlockBuilder) AddCol(c query.ColMeta) int {
	var col column
	switch c.Type {
	case query.TBool:
		col = &boolColumn{
			ColMeta: c,
			alloc:   b.alloc,
		}
	case query.TInt:
		col = &intColumn{
			ColMeta: c,
			alloc:   b.alloc,
		}
	case query.TUInt:
		col = &uintColumn{
			ColMeta: c,
			alloc:   b.alloc,
		}
	case query.TFloat:
		col = &floatColumn{
			ColMeta: c,
			alloc:   b.alloc,
		}
	case query.TString:
		col = &stringColumn{
			ColMeta: c,
			alloc:   b.alloc,
		}
	case query.TTime:
		col = &timeColumn{
			ColMeta: c,
			alloc:   b.alloc,
		}
	default:
		PanicUnknownType(c.Type)
	}
	b.blk.colMeta = append(b.blk.colMeta, c)
	b.blk.cols = append(b.blk.cols, col)
	return len(b.blk.cols) - 1
}

func (b ColListBlockBuilder) SetBool(i int, j int, value bool) {
	b.checkColType(j, query.TBool)
	b.blk.cols[j].(*boolColumn).data[i] = value
}
func (b ColListBlockBuilder) AppendBool(j int, value bool) {
	b.checkColType(j, query.TBool)
	col := b.blk.cols[j].(*boolColumn)
	col.data = b.alloc.AppendBools(col.data, value)
	b.blk.nrows = len(col.data)
}
func (b ColListBlockBuilder) AppendBools(j int, values []bool) {
	b.checkColType(j, query.TBool)
	col := b.blk.cols[j].(*boolColumn)
	col.data = b.alloc.AppendBools(col.data, values...)
	b.blk.nrows = len(col.data)
}

func (b ColListBlockBuilder) SetInt(i int, j int, value int64) {
	b.checkColType(j, query.TInt)
	b.blk.cols[j].(*intColumn).data[i] = value
}
func (b ColListBlockBuilder) AppendInt(j int, value int64) {
	b.checkColType(j, query.TInt)
	col := b.blk.cols[j].(*intColumn)
	col.data = b.alloc.AppendInts(col.data, value)
	b.blk.nrows = len(col.data)
}
func (b ColListBlockBuilder) AppendInts(j int, values []int64) {
	b.checkColType(j, query.TInt)
	col := b.blk.cols[j].(*intColumn)
	col.data = b.alloc.AppendInts(col.data, values...)
	b.blk.nrows = len(col.data)
}

func (b ColListBlockBuilder) SetUInt(i int, j int, value uint64) {
	b.checkColType(j, query.TUInt)
	b.blk.cols[j].(*uintColumn).data[i] = value
}
func (b ColListBlockBuilder) AppendUInt(j int, value uint64) {
	b.checkColType(j, query.TUInt)
	col := b.blk.cols[j].(*uintColumn)
	col.data = b.alloc.AppendUInts(col.data, value)
	b.blk.nrows = len(col.data)
}
func (b ColListBlockBuilder) AppendUInts(j int, values []uint64) {
	b.checkColType(j, query.TUInt)
	col := b.blk.cols[j].(*uintColumn)
	col.data = b.alloc.AppendUInts(col.data, values...)
	b.blk.nrows = len(col.data)
}

func (b ColListBlockBuilder) SetFloat(i int, j int, value float64) {
	b.checkColType(j, query.TFloat)
	b.blk.cols[j].(*floatColumn).data[i] = value
}
func (b ColListBlockBuilder) AppendFloat(j int, value float64) {
	b.checkColType(j, query.TFloat)
	col := b.blk.cols[j].(*floatColumn)
	col.data = b.alloc.AppendFloats(col.data, value)
	b.blk.nrows = len(col.data)
}
func (b ColListBlockBuilder) AppendFloats(j int, values []float64) {
	b.checkColType(j, query.TFloat)
	col := b.blk.cols[j].(*floatColumn)
	col.data = b.alloc.AppendFloats(col.data, values...)
	b.blk.nrows = len(col.data)
}

func (b ColListBlockBuilder) SetString(i int, j int, value string) {
	b.checkColType(j, query.TString)
	b.blk.cols[j].(*stringColumn).data[i] = value
}
func (b ColListBlockBuilder) AppendString(j int, value string) {
	meta := b.blk.cols[j].Meta()
	CheckColType(meta, query.TString)
	col := b.blk.cols[j].(*stringColumn)
	col.data = b.alloc.AppendStrings(col.data, value)
	b.blk.nrows = len(col.data)
}
func (b ColListBlockBuilder) AppendStrings(j int, values []string) {
	b.checkColType(j, query.TString)
	col := b.blk.cols[j].(*stringColumn)
	col.data = b.alloc.AppendStrings(col.data, values...)
	b.blk.nrows = len(col.data)
}

func (b ColListBlockBuilder) SetTime(i int, j int, value Time) {
	b.checkColType(j, query.TTime)
	b.blk.cols[j].(*timeColumn).data[i] = value
}
func (b ColListBlockBuilder) AppendTime(j int, value Time) {
	b.checkColType(j, query.TTime)
	col := b.blk.cols[j].(*timeColumn)
	col.data = b.alloc.AppendTimes(col.data, value)
	b.blk.nrows = len(col.data)
}
func (b ColListBlockBuilder) AppendTimes(j int, values []Time) {
	b.checkColType(j, query.TTime)
	col := b.blk.cols[j].(*timeColumn)
	col.data = b.alloc.AppendTimes(col.data, values...)
	b.blk.nrows = len(col.data)
}

func (b ColListBlockBuilder) checkColType(j int, typ query.DataType) {
	CheckColType(b.blk.colMeta[j], typ)
}

func CheckColType(col query.ColMeta, typ query.DataType) {
	if col.Type != typ {
		panic(fmt.Errorf("column %s is not of type %v", col.Label, typ))
	}
}

func PanicUnknownType(typ query.DataType) {
	panic(fmt.Errorf("unknown type %v", typ))
}

func (b ColListBlockBuilder) Block() (query.Block, error) {
	// Create copy in mutable state
	return b.blk.Copy(), nil
}

// RawBlock returns the underlying block being constructed.
// The Block returned will be modified by future calls to any BlockBuilder methods.
func (b ColListBlockBuilder) RawBlock() *ColListBlock {
	// Create copy in mutable state
	return b.blk
}

func (b ColListBlockBuilder) ClearData() {
	for _, c := range b.blk.cols {
		c.Clear()
	}
	b.blk.nrows = 0
}

func (b ColListBlockBuilder) Sort(cols []string, desc bool) {
	colIdxs := make([]int, len(cols))
	for i, label := range cols {
		for j, c := range b.blk.colMeta {
			if c.Label == label {
				colIdxs[i] = j
				break
			}
		}
	}
	s := colListBlockSorter{cols: colIdxs, desc: desc, b: b.blk}
	sort.Sort(s)
}

// ColListBlock implements Block using list of columns.
// All data for the block is stored in RAM.
// As a result At* methods are provided directly on the block for easy access.
type ColListBlock struct {
	key     query.PartitionKey
	colMeta []query.ColMeta
	cols    []column
	nrows   int

	refCount int32
}

func (b *ColListBlock) RefCount(n int) {
	c := atomic.AddInt32(&b.refCount, int32(n))
	if c == 0 {
		for _, c := range b.cols {
			c.Clear()
		}
	}
}

func (b *ColListBlock) Key() query.PartitionKey {
	return b.key
}
func (b *ColListBlock) Cols() []query.ColMeta {
	return b.colMeta
}
func (b *ColListBlock) Empty() bool {
	return b.nrows == 0
}
func (b *ColListBlock) NRows() int {
	return b.nrows
}

func (b *ColListBlock) Len() int {
	return b.nrows
}

func (b *ColListBlock) Do(f func(query.ColReader) error) error {
	return f(b)
}

func (b *ColListBlock) Bools(j int) []bool {
	CheckColType(b.colMeta[j], query.TBool)
	return b.cols[j].(*boolColumn).data
}
func (b *ColListBlock) Ints(j int) []int64 {
	CheckColType(b.colMeta[j], query.TInt)
	return b.cols[j].(*intColumn).data
}
func (b *ColListBlock) UInts(j int) []uint64 {
	CheckColType(b.colMeta[j], query.TUInt)
	return b.cols[j].(*uintColumn).data
}
func (b *ColListBlock) Floats(j int) []float64 {
	CheckColType(b.colMeta[j], query.TFloat)
	return b.cols[j].(*floatColumn).data
}
func (b *ColListBlock) Strings(j int) []string {
	meta := b.colMeta[j]
	CheckColType(meta, query.TString)
	return b.cols[j].(*stringColumn).data
}
func (b *ColListBlock) Times(j int) []Time {
	CheckColType(b.colMeta[j], query.TTime)
	return b.cols[j].(*timeColumn).data
}

func (b *ColListBlock) Copy() *ColListBlock {
	cpy := new(ColListBlock)
	cpy.key = b.key
	cpy.nrows = b.nrows

	cpy.colMeta = make([]query.ColMeta, len(b.colMeta))
	copy(cpy.colMeta, b.colMeta)

	cpy.cols = make([]column, len(b.cols))
	for i, c := range b.cols {
		cpy.cols[i] = c.Copy()
	}

	return cpy
}

type colListBlockSorter struct {
	cols []int
	desc bool
	b    *ColListBlock
}

func (c colListBlockSorter) Len() int {
	return c.b.nrows
}

func (c colListBlockSorter) Less(x int, y int) (less bool) {
	for _, j := range c.cols {
		if !c.b.cols[j].Equal(x, y) {
			less = c.b.cols[j].Less(x, y)
			break
		}
	}
	if c.desc {
		less = !less
	}
	return
}

func (c colListBlockSorter) Swap(x int, y int) {
	for _, col := range c.b.cols {
		col.Swap(x, y)
	}
}

type column interface {
	Meta() query.ColMeta
	Clear()
	Copy() column
	Equal(i, j int) bool
	Less(i, j int) bool
	Swap(i, j int)
}

type boolColumn struct {
	query.ColMeta
	data  []bool
	alloc *Allocator
}

func (c *boolColumn) Meta() query.ColMeta {
	return c.ColMeta
}

func (c *boolColumn) Clear() {
	c.alloc.Free(len(c.data), boolSize)
	c.data = c.data[0:0]
}
func (c *boolColumn) Copy() column {
	cpy := &boolColumn{
		ColMeta: c.ColMeta,
		alloc:   c.alloc,
	}
	l := len(c.data)
	cpy.data = c.alloc.Bools(l, l)
	copy(cpy.data, c.data)
	return cpy
}
func (c *boolColumn) Equal(i, j int) bool {
	return c.data[i] == c.data[j]
}
func (c *boolColumn) Less(i, j int) bool {
	if c.data[i] == c.data[j] {
		return false
	}
	return c.data[i]
}
func (c *boolColumn) Swap(i, j int) {
	c.data[i], c.data[j] = c.data[j], c.data[i]
}

type intColumn struct {
	query.ColMeta
	data  []int64
	alloc *Allocator
}

func (c *intColumn) Meta() query.ColMeta {
	return c.ColMeta
}

func (c *intColumn) Clear() {
	c.alloc.Free(len(c.data), int64Size)
	c.data = c.data[0:0]
}
func (c *intColumn) Copy() column {
	cpy := &intColumn{
		ColMeta: c.ColMeta,
		alloc:   c.alloc,
	}
	l := len(c.data)
	cpy.data = c.alloc.Ints(l, l)
	copy(cpy.data, c.data)
	return cpy
}
func (c *intColumn) Equal(i, j int) bool {
	return c.data[i] == c.data[j]
}
func (c *intColumn) Less(i, j int) bool {
	return c.data[i] < c.data[j]
}
func (c *intColumn) Swap(i, j int) {
	c.data[i], c.data[j] = c.data[j], c.data[i]
}

type uintColumn struct {
	query.ColMeta
	data  []uint64
	alloc *Allocator
}

func (c *uintColumn) Meta() query.ColMeta {
	return c.ColMeta
}

func (c *uintColumn) Clear() {
	c.alloc.Free(len(c.data), uint64Size)
	c.data = c.data[0:0]
}
func (c *uintColumn) Copy() column {
	cpy := &uintColumn{
		ColMeta: c.ColMeta,
		alloc:   c.alloc,
	}
	l := len(c.data)
	cpy.data = c.alloc.UInts(l, l)
	copy(cpy.data, c.data)
	return cpy
}
func (c *uintColumn) Equal(i, j int) bool {
	return c.data[i] == c.data[j]
}
func (c *uintColumn) Less(i, j int) bool {
	return c.data[i] < c.data[j]
}
func (c *uintColumn) Swap(i, j int) {
	c.data[i], c.data[j] = c.data[j], c.data[i]
}

type floatColumn struct {
	query.ColMeta
	data  []float64
	alloc *Allocator
}

func (c *floatColumn) Meta() query.ColMeta {
	return c.ColMeta
}

func (c *floatColumn) Clear() {
	c.alloc.Free(len(c.data), float64Size)
	c.data = c.data[0:0]
}
func (c *floatColumn) Copy() column {
	cpy := &floatColumn{
		ColMeta: c.ColMeta,
		alloc:   c.alloc,
	}
	l := len(c.data)
	cpy.data = c.alloc.Floats(l, l)
	copy(cpy.data, c.data)
	return cpy
}
func (c *floatColumn) Equal(i, j int) bool {
	return c.data[i] == c.data[j]
}
func (c *floatColumn) Less(i, j int) bool {
	return c.data[i] < c.data[j]
}
func (c *floatColumn) Swap(i, j int) {
	c.data[i], c.data[j] = c.data[j], c.data[i]
}

type stringColumn struct {
	query.ColMeta
	data  []string
	alloc *Allocator
}

func (c *stringColumn) Meta() query.ColMeta {
	return c.ColMeta
}

func (c *stringColumn) Clear() {
	c.alloc.Free(len(c.data), stringSize)
	c.data = c.data[0:0]
}
func (c *stringColumn) Copy() column {
	cpy := &stringColumn{
		ColMeta: c.ColMeta,
		alloc:   c.alloc,
	}

	l := len(c.data)
	cpy.data = c.alloc.Strings(l, l)
	copy(cpy.data, c.data)
	return cpy
}
func (c *stringColumn) Equal(i, j int) bool {
	return c.data[i] == c.data[j]
}
func (c *stringColumn) Less(i, j int) bool {
	return c.data[i] < c.data[j]
}
func (c *stringColumn) Swap(i, j int) {
	c.data[i], c.data[j] = c.data[j], c.data[i]
}

type timeColumn struct {
	query.ColMeta
	data  []Time
	alloc *Allocator
}

func (c *timeColumn) Meta() query.ColMeta {
	return c.ColMeta
}

func (c *timeColumn) Clear() {
	c.alloc.Free(len(c.data), timeSize)
	c.data = c.data[0:0]
}
func (c *timeColumn) Copy() column {
	cpy := &timeColumn{
		ColMeta: c.ColMeta,
		alloc:   c.alloc,
	}
	l := len(c.data)
	cpy.data = c.alloc.Times(l, l)
	copy(cpy.data, c.data)
	return cpy
}
func (c *timeColumn) Equal(i, j int) bool {
	return c.data[i] == c.data[j]
}
func (c *timeColumn) Less(i, j int) bool {
	return c.data[i] < c.data[j]
}
func (c *timeColumn) Swap(i, j int) {
	c.data[i], c.data[j] = c.data[j], c.data[i]
}

type BlockBuilderCache interface {
	// BlockBuilder returns an existing or new BlockBuilder for the given meta data.
	// The boolean return value indicates if BlockBuilder is new.
	BlockBuilder(key query.PartitionKey) (BlockBuilder, bool)
	ForEachBuilder(f func(query.PartitionKey, BlockBuilder))
}

type blockBuilderCache struct {
	blocks *PartitionLookup
	alloc  *Allocator

	triggerSpec query.TriggerSpec
}

func NewBlockBuilderCache(a *Allocator) *blockBuilderCache {
	return &blockBuilderCache{
		blocks: NewPartitionLookup(),
		alloc:  a,
	}
}

type blockState struct {
	builder BlockBuilder
	trigger Trigger
}

func (d *blockBuilderCache) SetTriggerSpec(ts query.TriggerSpec) {
	d.triggerSpec = ts
}

func (d *blockBuilderCache) Block(key query.PartitionKey) (query.Block, error) {
	b, ok := d.lookupState(key)
	if !ok {
		return nil, fmt.Errorf("block not found with key %v", key)
	}
	return b.builder.Block()
}

func (d *blockBuilderCache) lookupState(key query.PartitionKey) (blockState, bool) {
	v, ok := d.blocks.Lookup(key)
	if !ok {
		return blockState{}, false
	}
	return v.(blockState), true
}

// BlockBuilder will return the builder for the specified block.
// If no builder exists, one will be created.
func (d *blockBuilderCache) BlockBuilder(key query.PartitionKey) (BlockBuilder, bool) {
	b, ok := d.lookupState(key)
	if !ok {
		builder := NewColListBlockBuilder(key, d.alloc)
		t := NewTriggerFromSpec(d.triggerSpec)
		b = blockState{
			builder: builder,
			trigger: t,
		}
		d.blocks.Set(key, b)
	}
	return b.builder, !ok
}

func (d *blockBuilderCache) ForEachBuilder(f func(query.PartitionKey, BlockBuilder)) {
	d.blocks.Range(func(key query.PartitionKey, value interface{}) {
		f(key, value.(blockState).builder)
	})
}

func (d *blockBuilderCache) DiscardBlock(key query.PartitionKey) {
	b, ok := d.lookupState(key)
	if ok {
		b.builder.ClearData()
	}
}

func (d *blockBuilderCache) ExpireBlock(key query.PartitionKey) {
	b, ok := d.blocks.Delete(key)
	if ok {
		b.(blockState).builder.ClearData()
	}
}

func (d *blockBuilderCache) ForEach(f func(query.PartitionKey)) {
	d.blocks.Range(func(key query.PartitionKey, value interface{}) {
		f(key)
	})
}

func (d *blockBuilderCache) ForEachWithContext(f func(query.PartitionKey, Trigger, BlockContext)) {
	d.blocks.Range(func(key query.PartitionKey, value interface{}) {
		b := value.(blockState)
		f(key, b.trigger, BlockContext{
			Key:   key,
			Count: b.builder.NRows(),
		})
	})
}
