package parse

import (
	"fmt"
	"github.com/godyy/gexcels"
	"github.com/godyy/gexcels/internal/utils"
	"math"
	"reflect"
	"sort"
	"strings"
)

// TableLink 配置表链接
type TableLink struct {
	srcField []string // 源字段，可以指向结构体内部字段
	dstTable string   // 目标配置表
	dstField string   // 目标字段，ID或者unique
}

func newTableLink(srcField []string, dstTable, dstField string) *TableLink {
	return &TableLink{
		srcField: srcField,
		dstTable: dstTable,
		dstField: dstField,
	}
}

// TableCompositeKey 配置表组合键
type TableCompositeKey struct {
	Name            string         // key名
	fieldIndex2Name map[int]string // index映射 [keyIndex]fieldName
	fieldNames      []string       // 按顺序排列的字段名
}

func newTableCompositeKey(keyName string) *TableCompositeKey {
	return &TableCompositeKey{
		Name:            keyName,
		fieldIndex2Name: make(map[int]string),
	}
}

// addField 添加字段
func (tck *TableCompositeKey) addField(index int, name string) bool {
	if _, ok := tck.fieldIndex2Name[index]; ok {
		return false
	} else {
		tck.fieldIndex2Name[index] = name
		return true
	}
}

// FieldNames 获取字段名
func (tck *TableCompositeKey) FieldNames() []string {
	if tck.fieldNames != nil {
		return tck.fieldNames
	}

	keyIndexes := make([]int, 0, len(tck.fieldIndex2Name))
	for i := range tck.fieldIndex2Name {
		keyIndexes = append(keyIndexes, i)
	}
	sort.Ints(keyIndexes)

	tck.fieldNames = make([]string, len(tck.fieldIndex2Name))
	for i := range tck.fieldIndex2Name {
		tck.fieldNames[i] = tck.fieldIndex2Name[i]
	}
	return tck.fieldNames
}

// TableGroup 配置表分组
type TableGroup struct {
	Name            string         // 分组名
	fieldIndex2Name map[int]string // index映射 [index]fieldName
	fieldNames      []string       // 按顺序排列的字段名
}

func newTableGroup(name string) *TableGroup {
	return &TableGroup{
		Name:            name,
		fieldIndex2Name: make(map[int]string),
	}
}

// addField 添加字段
func (tg *TableGroup) addField(index int, name string) bool {
	if _, ok := tg.fieldIndex2Name[index]; ok {
		return false
	} else {
		tg.fieldIndex2Name[index] = name
		return true
	}
}

// FieldNames 获取字段名
func (tg *TableGroup) FieldNames() []string {
	if tg.fieldNames != nil {
		return tg.fieldNames
	}

	keyIndexes := make([]int, 0, len(tg.fieldIndex2Name))
	for i := range tg.fieldIndex2Name {
		keyIndexes = append(keyIndexes, i)
	}
	sort.Ints(keyIndexes)

	tg.fieldNames = make([]string, len(tg.fieldIndex2Name))
	for i := range tg.fieldIndex2Name {
		tg.fieldNames[i] = tg.fieldIndex2Name[i]
	}
	return tg.fieldNames
}

// addLink 添加外链规则
func (td *Table) addLink(rule ...*TableLink) {
	td.links = append(td.links, rule...)
}

// addCompositeKey 添加组合键
func (td *Table) addCompositeKey(keyName string, keyIndex int, fieldName string) bool {
	var ck *TableCompositeKey
	if n := sort.Search(len(td.CompositeKeys), func(i int) bool {
		return keyName <= td.CompositeKeys[i].Name
	}); n < len(td.CompositeKeys) && keyName == td.CompositeKeys[n].Name {
		ck = td.CompositeKeys[n]
	} else {
		ck = newTableCompositeKey(keyName)
		td.CompositeKeys = append(td.CompositeKeys, nil)
		copy(td.CompositeKeys[n+1:], td.CompositeKeys[n:])
		td.CompositeKeys[n] = ck
	}
	return ck.addField(keyIndex, fieldName)
}

// addCompositeKeyValue 添加组合键值
func (td *Table) addCompositeKeyValue(entry gexcels.TableEntry) error {
	fieldId := td.GetFieldID()
	var sb strings.Builder
	for _, ck := range td.CompositeKeys {
		fields := ck.FieldNames()
		for i, field := range fields {
			if i > 0 {
				sb.WriteString("_")
			}
			value := entry[field]
			if value == nil {
				return fmt.Errorf("row[%s=%v] composite-key %s field %s is nil", fieldId.Name, entry[fieldId.Name], ck.Name, field)
			}
			sb.WriteString(convertUniqueValue2String(entry[field]))
		}
		value := sb.String()
		if !td.addUniqueValue(ck.Name, value) {
			return fmt.Errorf("row[%s=%v] composite-key %s duplicate", fieldId.Name, entry[fieldId.Name], ck.Name)
		}
		sb.Reset()
	}
	return nil
}

// addGroup 添加分组
func (td *Table) addGroup(group string, index int, fieldName string) bool {
	var g *TableGroup
	if n := sort.Search(len(td.Groups), func(i int) bool {
		return group <= td.Groups[i].Name
	}); n < len(td.Groups) && group == td.Groups[n].Name {
		g = td.Groups[n]
	} else {
		g = newTableGroup(group)
		td.Groups = append(td.Groups, nil)
		copy(td.Groups[n+1:], td.Groups[n:])
		td.Groups[n] = g
	}
	return g.addField(index, fieldName)
}

// getFieldTableLinks 从字段fd中获取TableLink
func (p *Parser) getFieldTableLinks(fd *gexcels.Field, fieldPath *[]string, links *[]*TableLink) {
	if fd.Type.Primitive() || (fd.Type == gexcels.FTArray && fd.ElementType.Primitive()) {
		if ruleLink := fd.GetFRLink(); ruleLink != nil {
			srcField := make([]string, len(*fieldPath)+1)
			copy(srcField, *fieldPath)
			srcField[len(srcField)-1] = fd.Name
			*links = append(*links, newTableLink(srcField, ruleLink.TableName, ruleLink.FieldName))
		}
	} else if fd.Type == gexcels.FTStruct || fd.ElementType == gexcels.FTStruct {
		sd := p.GetStructByName(fd.StructName)
		if sd == nil {
			return
		}

		*fieldPath = append(*fieldPath, fd.Name)
		for _, fd := range sd.Fields {
			p.getFieldTableLinks(fd, fieldPath, links)
		}
		*fieldPath = (*fieldPath)[:len(*fieldPath)-1]
	}
}

// checkTableLinks 检查配置表td外链的其它配置表
func (p *Parser) checkTableLinks(td *Table) (errs []error) {
	for _, link := range td.links {
		if err := p.checkTableLinkTable(td, link); err != nil {
			errs = append(errs, err...)
		}
	}
	return
}

// checkTableLinkTable 检查配置表td根据link外链的配置表
func (p *Parser) checkTableLinkTable(td *Table, link *TableLink) (errs []error) {
	var (
		srcTable, dstTable *Table
		srcField, dstField *gexcels.Field
	)

	srcTable = td
	dstTable = p.getTableByName(link.dstTable)
	if dstTable == nil {
		return []error{errTableLinkByTableLink(td.Name, link, "dst table not found")}
	}

	if srcTableField := srcTable.GetFieldByName(link.srcField[0]); srcTableField == nil {
		return []error{errTableLinkByTableLink(td.Name, link, "src field not found")}
	} else {
		srcField = srcTableField.Field
		if len(link.srcField) > 1 {
			srcField = p.getNestedField(srcField, link.srcField, 1)
			if srcField == nil {
				return []error{errTableLinkByTableLink(td.Name, link, "src field not found")}
			}
		}
		if !srcField.Type.Primitive() && !(srcField.Type == gexcels.FTArray && srcField.ElementType.Primitive()) {
			return []error{errTableLinkByTableLink(td.Name, link, "src field type not primitive")}
		}
	}
	if dstTableField := dstTable.GetFieldByName(link.dstField); dstTableField == nil {
		return []error{errTableLinkByTableLink(td.Name, link, "dst field not found")}
	} else if !dstTableField.Unique() {
		return []error{errTableLinkByTableLink(td.Name, link, "dst field not Unique or not export method")}
	} else if !dstTableField.Type.Primitive() {
		return []error{errTableLinkByTableLink(td.Name, link, "dst field type not primitive")}
	} else {
		dstField = dstTableField.Field
	}
	if !checkFieldTypeOnLink(srcField, dstField) {
		return []error{errTableLinkByTableLink(td.Name, link, "field type not match")}
	}

	if srcTable.IsGlobal {
		fieldValue := srcTable.GetEntryByName(link.srcField[0])
		if fieldValue == nil {
			return []error{errTableLinkByTableLink(td.Name, link, "src field value not found")}
		}
		if err := checkTableLinkValue(srcTable, fieldValue, dstTable, dstField, link.srcField, 1); err != nil {
			errs = append(errs, err...)
		}
	} else {
		for _, srcEntry := range srcTable.Entries {
			fieldValue := srcEntry[link.srcField[0]]
			if fieldValue == nil {
				continue
			}
			if err := checkTableLinkValue(srcTable, fieldValue, dstTable, dstField, link.srcField, 1); err != nil {
				errs = append(errs, err...)
			}
		}
	}

	return
}

// checkFieldTypeOnLink 检查两个链接的字段类型是否一致
func checkFieldTypeOnLink(srcField, dstField *gexcels.Field) bool {
	srcFieldType := srcField.Type
	if srcFieldType == gexcels.FTArray {
		srcFieldType = srcField.ElementType
	}
	return srcFieldType == dstField.Type
}

// checkLinksBetweenTable 检查配置表之间的链接
func (p *Parser) checkLinksBetweenTable() (errs []error) {
	if p.options.OnlyFields {
		return nil
	}

	for _, table := range p.Tables {
		if err := p.checkTableLinks(table); err != nil {
			errs = append(errs, err...)
		}
	}
	return
}

// checkTableLinkValue 字段值链接检查
// 检查value根据filePath指定的字段路径，是否能攻能成功链接到dstTable中的目标字段
func checkTableLinkValue(srcTable *Table, srcValue interface{}, dstTable *Table, dstField *gexcels.Field, fieldPath []string, depth int) (errs []error) {
	v := reflect.ValueOf(srcValue)
	if v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			vv := v.Index(i)
			if err := checkTableLinkValue(srcTable, vv.Interface(), dstTable, dstField, fieldPath, depth); err != nil {
				errs = append(errs, err...)
			}
		}
	} else if v.Kind() == reflect.Pointer {
		vv := v.Elem()
		if vv.Kind() != reflect.Struct {
			return []error{errTableLink(srcTable.Name, fieldPath[:depth], dstTable.Name, dstField.Name, "value (%+v) not struct", vv.Interface())}
		}
		fieldName := utils.CamelCase(fieldPath[depth], true)
		vv = vv.FieldByName(fieldName)
		if !vv.IsValid() {
			return nil
		}
		errs = checkTableLinkValue(srcTable, vv.Interface(), dstTable, dstField, fieldPath, depth+1)
	} else {
		if depth < len(fieldPath) {
			return []error{errTableLink(srcTable.Name, fieldPath[:depth], dstTable.Name, dstField.Name, "primitive value (%v) on middle path", srcValue)}
		}

		vv, err := convertValue2PrimitiveFieldType(srcValue, dstField.Type)
		if err != nil {
			errs = []error{err}
		} else if dstField.Name == gexcels.TableFieldIDName {
			if !dstTable.hasEntry(vv) {
				errs = []error{errTableLink(srcTable.Name, fieldPath[:depth], dstTable.Name, dstField.Name, "dst.%s=%v not found", dstField.Name, vv)}
			}
		} else {
			if !dstTable.hasUniqueValue(dstField.Name, vv) {
				errs = []error{errTableLink(srcTable.Name, fieldPath[:depth], dstTable.Name, dstField.Name, "dst.%s=%v not found", dstField.Name, vv)}
			}
		}
	}

	return
}

// 尝试将val转换为ft指定的类型
func convertValue2PrimitiveFieldType(val interface{}, ft gexcels.FieldType) (interface{}, error) {
	switch ft {
	case gexcels.FTInt32:
		switch val.(type) {
		case int8, int16, int32, int64, uint8, uint16, uint32, uint64:
			return reflect.ValueOf(val).Convert(reflect.TypeOf(int32(0))).Interface(), nil
		case float32:
			f := float64(val.(float32))
			if math.Trunc(f) == f {
				return int32(f), nil
			}
		case float64:
			f := val.(float64)
			if math.Trunc(f) == f {
				return int32(f), nil
			}
		}

	case gexcels.FTInt64:
		switch val.(type) {
		case int8, int16, int32, int64, uint8, uint16, uint32, uint64:
			return reflect.ValueOf(val).Convert(reflect.TypeOf(int64(0))).Interface(), nil
		case float32:
			f := float64(val.(float32))
			if math.Trunc(f) == f {
				return int64(f), nil
			}
		case float64:
			f := val.(float64)
			if math.Trunc(f) == f {
				return int64(f), nil
			}
		}

	case gexcels.FTFloat32:
		v := reflect.ValueOf(val)
		t := reflect.TypeOf(float32(0))
		if v.CanConvert(t) {
			return v.Convert(t).Interface(), nil
		}
	case gexcels.FTFloat64:
		v := reflect.ValueOf(val)
		t := reflect.TypeOf(float64(0))
		if v.CanConvert(t) {
			return v.Convert(t).Interface(), nil
		}

	case gexcels.FTBool:
		if v, ok := val.(bool); ok {
			return v, nil
		}

	case gexcels.FTString:
		if v, ok := val.(string); ok {
			return v, nil
		}

	default:
		return nil, errFieldTypeInvalid(ft)
	}

	return nil, fmt.Errorf("value (%v) cant convert to type %s", val, ft.PrimitiveString())
}
