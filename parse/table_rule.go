package parse

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"

	"github.com/godyy/gexcels"
)

// TableLink 配置表链接
type TableLink struct {
	srcField []string // 源字段，可以指向结构体内部字段
	dstTable string   // 目标配置表
	dstField string   // 目标字段，ID或者unique
	kind     tableLinkKind
	mapLevel int
}

type tableLinkKind uint8

const (
	tableLinkKindValue tableLinkKind = iota
	tableLinkKindMapKey
)

func newTableLink(srcField []string, dstTable, dstField string, kind tableLinkKind, mapLevel int) *TableLink {
	return &TableLink{
		srcField: srcField,
		dstTable: dstTable,
		dstField: dstField,
		kind:     kind,
		mapLevel: mapLevel,
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
	if ruleLink := fd.GetFRLink(); ruleLink != nil {
		srcField := make([]string, 0, len(*fieldPath)+1)
		srcField = append(srcField, *fieldPath...)
		srcField = append(srcField, fd.Name)

		if ruleLink.Value != nil {
			*links = append(*links, newTableLink(srcField, ruleLink.Value.TableName, ruleLink.Value.FieldName, tableLinkKindValue, 0))
		}
		for level, tgt := range ruleLink.Keys {
			*links = append(*links, newTableLink(srcField, tgt.TableName, tgt.FieldName, tableLinkKindMapKey, level))
		}
	}

	if fd.Type == gexcels.FTStruct || fd.Type == gexcels.FTArray || fd.Type == gexcels.FTMap {
		*fieldPath = append(*fieldPath, fd.Name)
		p.getTypeTableLinks(fd.FieldTypeInfo, fieldPath, links)
		*fieldPath = (*fieldPath)[:len(*fieldPath)-1]
	}
}

// getTypeTableLinks 从字段ti中获取TableLink
func (p *Parser) getTypeTableLinks(ti *gexcels.FieldTypeInfo, fieldPath *[]string, links *[]*TableLink) {
	switch ti.Type {
	case gexcels.FTStruct:
		sd := p.GetStructByName(ti.GetName())
		if sd == nil {
			return
		}
		for _, fd := range sd.Fields {
			p.getFieldTableLinks(fd, fieldPath, links)
		}
	case gexcels.FTArray:
		p.getTypeTableLinks(ti.GetElementType(), fieldPath, links)
	case gexcels.FTMap:
		p.getTypeTableLinks(ti.GetMapValueType(), fieldPath, links)
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
		srcRootField       *gexcels.Field
		dstField           *gexcels.Field
	)

	srcTable = td
	dstTable = p.getTableByName(link.dstTable)
	if dstTable == nil {
		return []error{errTableLink(td.Name, link, "dst table not found")}
	}

	if srcTableField := srcTable.GetFieldByName(link.srcField[0]); srcTableField == nil {
		return []error{errTableLink(td.Name, link, "src field not found")}
	} else {
		srcRootField = srcTableField.Field
	}
	if dstTableField := dstTable.GetFieldByName(link.dstField); dstTableField == nil {
		return []error{errTableLink(td.Name, link, "dst field not found")}
	} else if !dstTableField.Unique() {
		return []error{errTableLink(td.Name, link, "dst field not Unique or not export method")}
	} else if !(dstTableField.Type.Primitive() || dstTableField.Type == gexcels.FTEnum) {
		return []error{errTableLink(td.Name, link, "dst field type not primitive or enum")}
	} else {
		dstField = dstTableField.Field
	}

	path := link.srcField[1:]
	var validateErr error
	switch link.kind {
	case tableLinkKindValue:
		validateErr = p.validateValueLinkSource(srcRootField.FieldTypeInfo, path, dstField.FieldTypeInfo)
	case tableLinkKindMapKey:
		validateErr = p.validateMapKeyLinkSource(srcRootField.FieldTypeInfo, path, link.mapLevel, dstField.FieldTypeInfo)
	default:
		validateErr = fmt.Errorf("link kind invalid")
	}
	if validateErr != nil {
		return []error{errTableLink(td.Name, link, "%s", validateErr.Error())}
	}

	if srcTable.IsGlobal {
		fieldValue := srcTable.GetEntryByName(link.srcField[0])
		if fieldValue == nil {
			return []error{errTableLink(td.Name, link, "src field value not found")}
		}
		errs = append(errs, p.checkTableLinkValue(srcTable, fieldValue, srcRootField.FieldTypeInfo, path, link, dstTable, dstField)...)
	} else {
		for _, srcEntry := range srcTable.Entries {
			fieldValue := srcEntry[link.srcField[0]]
			if fieldValue == nil {
				continue
			}
			errs = append(errs, p.checkTableLinkValue(srcTable, fieldValue, srcRootField.FieldTypeInfo, path, link, dstTable, dstField)...)
		}
	}

	return
}

// validateValueLinkSource 检查配置表值链接值类型是否匹配
func (p *Parser) validateValueLinkSource(srcType *gexcels.FieldTypeInfo, path []string, dstType *gexcels.FieldTypeInfo) error {
	leafType, err := p.getValueLeafType(srcType, path)
	if err != nil {
		return err
	}
	if leafType.Type != dstType.Type || (leafType.Type == gexcels.FTEnum && leafType.GetName() != dstType.GetName()) {
		return fmt.Errorf("field type not match")
	}
	return nil
}

// getValueLeafType 获取配置表字段的叶子类型
func (p *Parser) getValueLeafType(ti *gexcels.FieldTypeInfo, path []string) (*gexcels.FieldTypeInfo, error) {
	switch ti.Type {
	case gexcels.FTArray:
		return p.getValueLeafType(ti.GetElementType(), path)
	case gexcels.FTMap:
		return p.getValueLeafType(ti.GetMapValueType(), path)
	case gexcels.FTStruct:
		if len(path) == 0 {
			return nil, fmt.Errorf("LINK on struct must be defined on struct field")
		}
		sd := p.GetStructByName(ti.GetName())
		if sd == nil {
			return nil, fmt.Errorf("struct %s not define", ti.GetName())
		}
		fd := sd.GetFieldByName(path[0])
		if fd == nil {
			return nil, fmt.Errorf("src field not found")
		}
		return p.getValueLeafType(fd.FieldTypeInfo, path[1:])
	case gexcels.FTEnum:
		return ti, nil
	default:
		if !ti.Type.Primitive() {
			return nil, fmt.Errorf("src field type invalid")
		}
		if len(path) != 0 {
			return nil, fmt.Errorf("src field not found")
		}
		return ti, nil
	}
}

// validateMapKeyLinkSource 检查配置表映射键链接值类型是否匹配
func (p *Parser) validateMapKeyLinkSource(srcType *gexcels.FieldTypeInfo, path []string, mapLevel int, dstType *gexcels.FieldTypeInfo) error {
	if mapLevel <= 0 {
		return fmt.Errorf("map level invalid")
	}

	keyType, ok, err := p.findMapKeyTypeAtLevel(srcType, path, mapLevel, 0)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("map level %d not exist", mapLevel)
	}
	if keyType.Type != dstType.Type || (keyType.Type == gexcels.FTEnum && keyType.GetName() != dstType.GetName()) {
		return fmt.Errorf("field type not match")
	}
	return nil
}

// findMapKeyTypeAtLevel 按照层级查找映射键类型
func (p *Parser) findMapKeyTypeAtLevel(ti *gexcels.FieldTypeInfo, path []string, targetLevel int, curLevel int) (*gexcels.FieldTypeInfo, bool, error) {
	switch ti.Type {
	case gexcels.FTArray:
		return p.findMapKeyTypeAtLevel(ti.GetElementType(), path, targetLevel, curLevel)
	case gexcels.FTStruct:
		if len(path) == 0 {
			return nil, false, fmt.Errorf("src field not found")
		}
		sd := p.GetStructByName(ti.GetName())
		if sd == nil {
			return nil, false, fmt.Errorf("struct %s not define", ti.GetName())
		}
		fd := sd.GetFieldByName(path[0])
		if fd == nil {
			return nil, false, fmt.Errorf("src field not found")
		}
		return p.findMapKeyTypeAtLevel(fd.FieldTypeInfo, path[1:], targetLevel, curLevel)
	case gexcels.FTMap:
		curLevel++
		if curLevel == targetLevel {
			return ti.GetMapKeyType(), true, nil
		}

		vt := ti.GetMapValueType()
		if vt.Type == gexcels.FTArray || vt.Type == gexcels.FTMap {
			return p.findMapKeyTypeAtLevel(vt, path, targetLevel, curLevel)
		}
		return nil, false, nil
	default:
		return nil, false, nil
	}
}

// checkTableLinkValue 检查配置表链接值是否有效
func (p *Parser) checkTableLinkValue(srcTable *Table, srcValue any, srcType *gexcels.FieldTypeInfo, path []string, link *TableLink, dstTable *Table, dstField *gexcels.Field) (errs []error) {
	switch link.kind {
	case tableLinkKindValue:
		return p.checkValueLinkValue(srcTable, srcValue, srcType, path, link, dstTable, dstField)
	case tableLinkKindMapKey:
		return p.checkMapKeyLinkValue(srcTable, srcValue, srcType, path, link, 0, dstTable, dstField)
	default:
		return []error{errTableLink(srcTable.Name, link, "link kind invalid")}
	}
}

// checkValueLinkValue 检查配置表值链接值是否有效
func (p *Parser) checkValueLinkValue(srcTable *Table, srcValue any, srcType *gexcels.FieldTypeInfo, path []string, link *TableLink, dstTable *Table, dstField *gexcels.Field) (errs []error) {
	if srcValue == nil {
		return nil
	}

	switch srcType.Type {
	case gexcels.FTArray:
		v := reflect.ValueOf(srcValue)
		if v.Kind() != reflect.Slice {
			return []error{errTableLink(srcTable.Name, link, "src value not array")}
		}
		for i := 0; i < v.Len(); i++ {
			errs = append(errs, p.checkValueLinkValue(srcTable, v.Index(i).Interface(), srcType.GetElementType(), path, link, dstTable, dstField)...)
		}
		return errs
	case gexcels.FTMap:
		v := reflect.ValueOf(srcValue)
		if v.Kind() != reflect.Map {
			return []error{errTableLink(srcTable.Name, link, "src value not map")}
		}
		for _, k := range v.MapKeys() {
			vv := v.MapIndex(k)
			if !vv.IsValid() {
				continue
			}
			errs = append(errs, p.checkValueLinkValue(srcTable, vv.Interface(), srcType.GetMapValueType(), path, link, dstTable, dstField)...)
		}
		return errs
	case gexcels.FTStruct:
		if len(path) == 0 {
			return []error{errTableLink(srcTable.Name, link, "LINK on struct must be defined on struct field")}
		}
		obj, ok := srcValue.(map[string]any)
		if !ok {
			return []error{errTableLink(srcTable.Name, link, "src value not object")}
		}
		fieldName := path[0]
		subValue, ok := obj[fieldName]
		if !ok || subValue == nil {
			return nil
		}
		sd := p.GetStructByName(srcType.GetName())
		if sd == nil {
			return []error{errTableLink(srcTable.Name, link, "struct not found")}
		}
		subField := sd.GetFieldByName(fieldName)
		if subField == nil {
			return []error{errTableLink(srcTable.Name, link, "src field not found")}
		}
		return p.checkValueLinkValue(srcTable, subValue, subField.FieldTypeInfo, path[1:], link, dstTable, dstField)
	case gexcels.FTEnum:
		if len(path) != 0 {
			return []error{errTableLink(srcTable.Name, link, "src field not found")}
		}
		return p.checkTableLinkPrimitiveValue(srcTable, srcValue, link, dstTable, dstField)
	default:
		if !srcType.Type.Primitive() {
			return []error{errTableLink(srcTable.Name, link, "src field type invalid")}
		}
		if len(path) != 0 {
			return []error{errTableLink(srcTable.Name, link, "src field not found")}
		}
		return p.checkTableLinkPrimitiveValue(srcTable, srcValue, link, dstTable, dstField)
	}
}

// checkMapKeyLinkValue 检查配置表映射键链接值是否有效
func (p *Parser) checkMapKeyLinkValue(srcTable *Table, srcValue any, srcType *gexcels.FieldTypeInfo, path []string, link *TableLink, curMapLevel int, dstTable *Table, dstField *gexcels.Field) (errs []error) {
	if srcValue == nil {
		return nil
	}

	switch srcType.Type {
	case gexcels.FTArray:
		v := reflect.ValueOf(srcValue)
		if v.Kind() != reflect.Slice {
			return []error{errTableLink(srcTable.Name, link, "src value not array")}
		}
		for i := 0; i < v.Len(); i++ {
			errs = append(errs, p.checkMapKeyLinkValue(srcTable, v.Index(i).Interface(), srcType.GetElementType(), path, link, curMapLevel, dstTable, dstField)...)
		}
		return errs
	case gexcels.FTStruct:
		if len(path) == 0 {
			return []error{errTableLink(srcTable.Name, link, "src field not found")}
		}
		obj, ok := srcValue.(map[string]any)
		if !ok {
			return []error{errTableLink(srcTable.Name, link, "src value not object")}
		}
		fieldName := path[0]
		subValue, ok := obj[fieldName]
		if !ok || subValue == nil {
			return nil
		}
		sd := p.GetStructByName(srcType.GetName())
		if sd == nil {
			return []error{errTableLink(srcTable.Name, link, "struct not found")}
		}
		subField := sd.GetFieldByName(fieldName)
		if subField == nil {
			return []error{errTableLink(srcTable.Name, link, "src field not found")}
		}
		return p.checkMapKeyLinkValue(srcTable, subValue, subField.FieldTypeInfo, path[1:], link, curMapLevel, dstTable, dstField)
	case gexcels.FTMap:
		v := reflect.ValueOf(srcValue)
		if v.Kind() != reflect.Map {
			return []error{errTableLink(srcTable.Name, link, "src value not map")}
		}

		curMapLevel++
		if curMapLevel == link.mapLevel {
			for _, k := range v.MapKeys() {
				errs = append(errs, p.checkTableLinkPrimitiveValue(srcTable, k.Interface(), link, dstTable, dstField)...)
			}
			return errs
		}

		vt := srcType.GetMapValueType()
		if vt.Type != gexcels.FTArray && vt.Type != gexcels.FTMap {
			return nil
		}
		for _, k := range v.MapKeys() {
			vv := v.MapIndex(k)
			if !vv.IsValid() {
				continue
			}
			errs = append(errs, p.checkMapKeyLinkValue(srcTable, vv.Interface(), vt, path, link, curMapLevel, dstTable, dstField)...)
		}
		return errs
	default:
		return nil
	}
}

// checkTableLinkPrimitiveValue 检查配置表链接值是否有效
func (p *Parser) checkTableLinkPrimitiveValue(srcTable *Table, srcValue any, link *TableLink, dstTable *Table, dstField *gexcels.Field) (errs []error) {
	vv, err := convertValue2PrimitiveFieldType(srcValue, dstField.Type)
	if err != nil {
		return []error{err}
	}

	if dstField.Name == gexcels.TableFieldIDName {
		if !dstTable.hasEntry(vv) {
			errs = []error{errTableLink(srcTable.Name, link, "dst.%s=%v not found", dstField.Name, vv)}
		}
	} else {
		if !dstTable.hasUniqueValue(dstField.Name, vv) {
			errs = []error{errTableLink(srcTable.Name, link, "dst.%s=%v not found", dstField.Name, vv)}
		}
	}
	return errs
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

// 尝试将val转换为ft指定的类型
func convertValue2PrimitiveFieldType(val any, ft gexcels.FieldType) (any, error) {
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
