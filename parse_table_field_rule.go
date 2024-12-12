package gexcels

import (
	"fmt"
	pkg_errors "github.com/pkg/errors"
	"reflect"
	"strconv"
	"strings"
)

// parseTableFieldRules 解析配置表字段规则
func (p *Parser) parseTableFieldRules(td *Table, fd *TableField, s string) error {
	if s == "" {
		return nil
	}

	rules := strings.Split(s, ruleSep)
	for _, rule := range rules {
		matches := ruleRegexp.FindStringSubmatch(rule)
		if len(matches) != 3 {
			return errRuleDefineInvalid(rule)
		}

		ruleType := strings.ToLower(matches[1])
		switch ruleType {
		case fieldRuleUnique:
			if err := p.parseTableFieldRuleUnique(td, fd, matches[2]); err != nil {
				return err
			}
		case fieldRuleLink:
			if err := p.parseTableFieldRuleLink(td, fd, matches[2]); err != nil {
				return err
			}
		case fieldRuleCompositeKey:
			if td.IsGlobal {
				return errRuleCompositeKeyOnGlobalTable
			}
			if err := p.parseTableFieldRuleCompositeKey(td, fd, matches[2]); err != nil {
				return err
			}
		default:
			return errRuleTypeInvalid(matches[1])
		}
	}

	return nil
}

// parseTableFieldRuleUnique 解析配置表字段Unique规则
func (p *Parser) parseTableFieldRuleUnique(td *Table, fd *TableField, value string) error {
	if !fd.Type.Primitive() {
		return errRuleUniqueOnNonPrimitiveField
	}
	exportMethod := false
	if value != "" {
		b, err := strconv.ParseBool(value)
		if err != nil {
			return pkg_errors.WithMessagef(err, "unique value \"%s\" invalid", value)
		}
		exportMethod = b
	}
	if !fd.addRule(&RuleUnique{ExportMethod: exportMethod}) {
		return errRuleMultiple(fieldRuleUnique)
	}
	return nil
}

// parseTableFieldRuleLink 解析配置表字段链接规则
func (p *Parser) parseTableFieldRuleLink(td *Table, fd *TableField, value string) error {
	if !fd.Type.Primitive() && !(fd.Type == FTArray && fd.ElementType.Primitive()) {
		return errRuleLinkOnNonPrimitiveField
	}

	ss := tableFieldRuleLinkRegexp.FindStringSubmatch(value)
	if len(ss) != 3 {
		return fmt.Errorf("link value \"%s\" invalid", value)
	}

	if !fd.addRule(&RuleLink{
		DstTable: ss[1],
		DstField: ss[2],
	}) {
		return errRuleMultiple(fieldRuleLink)
	}

	td.addLink(newTableLink([]string{fd.Name}, ss[1], ss[2]))

	return nil
}

func (p *Parser) parseTableFieldRuleCompositeKey(td *Table, fd *TableField, value string) error {
	if !fd.Type.Primitive() {
		return errRuleCompositeKeyOnNonPrimitiveField
	}

	ss := tableFieldRuleCompositeKey.FindStringSubmatch(value)
	if len(ss) != 3 {
		return fmt.Errorf("composite-key value \"%s\" invalid", value)
	}

	keyName := ss[1]
	keyIndex, _ := strconv.Atoi(ss[2])

	rule := convertRule[*RuleCompositeKey](getCreateFieldRule(fd.Field, fieldRuleCompositeKey))
	if !rule.addKeyIndex(keyName, keyIndex) {
		return fmt.Errorf("composite-key %s keyIndex %d duplicate on field", keyName, keyIndex)
	}

	if !td.addCompositeKey(keyName, keyIndex, fd.Name) {
		return fmt.Errorf("composite-key %s keyIndex %d duplicate", keyName, keyIndex)
	}

	return nil
}

// getFieldTableLinks 从字段fd中获取TableLink
func (p *Parser) getFieldTableLinks(fd *Field, fieldPath *[]string, links *[]*TableLink) {
	if fd.Type.Primitive() || (fd.Type == FTArray && fd.ElementType.Primitive()) {
		if ruleLink, ok := fd.getRule(fieldRuleLink).(*RuleLink); ok && ruleLink != nil {
			srcField := make([]string, len(*fieldPath)+1)
			copy(srcField, *fieldPath)
			srcField[len(srcField)-1] = fd.Name
			*links = append(*links, newTableLink(srcField, ruleLink.DstTable, ruleLink.DstField))
		}
	} else if fd.Type == FTStruct || fd.ElementType == FTStruct {
		sd := p.getStruct(fd.StructName)
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
	for _, link := range td.Links {
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
		srcField, dstField *Field
	)

	srcTable = td
	dstTable = p.getTableByName(link.DstTable)
	if dstTable == nil {
		return []error{makeErrTableLinkByTableLink(td.Name, link, "dst table not found")}
	}

	if srcTableField := srcTable.GetFieldByName(link.SrcField[0]); srcTableField == nil {
		return []error{makeErrTableLinkByTableLink(td.Name, link, "src field not found")}
	} else {
		srcField = srcTableField.Field
		if len(link.SrcField) > 1 {
			srcField = p.getNestedField(srcField, link.SrcField, 1)
			if srcField == nil {
				return []error{makeErrTableLinkByTableLink(td.Name, link, "src field not found")}
			}
		}
		if !srcField.Type.Primitive() && !(srcField.Type == FTArray && srcField.ElementType.Primitive()) {
			return []error{makeErrTableLinkByTableLink(td.Name, link, "src field type not primitive")}
		}
	}
	if dstTableField := dstTable.GetFieldByName(link.DstField); dstTableField == nil {
		return []error{makeErrTableLinkByTableLink(td.Name, link, "dst field not found")}
	} else if !dstTableField.ExportUniqueMethod() {
		return []error{makeErrTableLinkByTableLink(td.Name, link, "dst field not Unique or not export method")}
	} else if !dstTableField.Type.Primitive() {
		return []error{makeErrTableLinkByTableLink(td.Name, link, "dst field type not primitive")}
	} else {
		dstField = dstTableField.Field
	}
	if !checkFieldLinkType(srcField, dstField) {
		return []error{makeErrTableLinkByTableLink(td.Name, link, "field type not match")}
	}

	if srcTable.IsGlobal {
		fieldValue := srcTable.getEntryByName(link.SrcField[0])
		if fieldValue == nil {
			return []error{makeErrTableLinkByTableLink(td.Name, link, "src field value not found")}
		}
		if err := checkTableLinkValue(srcTable, fieldValue, dstTable, dstField, link.SrcField, 1); err != nil {
			errs = append(errs, err...)
		}
	} else {
		for _, srcEntry := range srcTable.Entries {
			fieldValue := srcEntry[link.SrcField[0]]
			if fieldValue == nil {
				continue
			}
			if err := checkTableLinkValue(srcTable, fieldValue, dstTable, dstField, link.SrcField, 1); err != nil {
				errs = append(errs, err...)
			}
		}
	}

	return
}

// checkLinksBetweenTable 检查配置表之间的链接
func (p *Parser) checkLinksBetweenTable() (errs []error) {
	for _, table := range p.tables {
		if err := p.checkTableLinks(table); err != nil {
			errs = append(errs, err...)
		}
	}
	return
}

// checkTableLinkValue 字段值链接检查
// 检查value根据filePath指定的字段路径，是否能攻能成功链接到dstTable中的目标字段
func checkTableLinkValue(srcTable *Table, srcValue interface{}, dstTable *Table, dstField *Field, fieldPath []string, depth int) (errs []error) {
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
			return []error{makeErrTableLink(srcTable.Name, fieldPath[:depth], dstTable.Name, dstField.Name, "value (%+v) not struct", vv.Interface())}
		}
		fieldName := exportFieldName(fieldPath[depth])
		vv = vv.FieldByName(fieldName)
		if !vv.IsValid() {
			return nil
		}
		errs = checkTableLinkValue(srcTable, vv.Interface(), dstTable, dstField, fieldPath, depth+1)
	} else {
		if depth < len(fieldPath) {
			return []error{makeErrTableLink(srcTable.Name, fieldPath[:depth], dstTable.Name, dstField.Name, "primitive value (%v) on middle path", srcValue)}
		}

		vv, err := convertValue2PrimitiveFieldType(srcValue, dstField.Type)
		if err != nil {
			errs = []error{err}
		} else if dstField.Name == idExportName {
			if !dstTable.hasEntry(vv) {
				errs = []error{makeErrTableLink(srcTable.Name, fieldPath[:depth], dstTable.Name, dstField.Name, "dst.ID=%v not found", vv)}
			}
		} else {
			if !dstTable.hasUniqueValue(dstField.Name, vv) {
				errs = []error{makeErrTableLink(srcTable.Name, fieldPath[:depth], dstTable.Name, dstField.Name, "dst.%s=%v not found", dstField.Name, vv)}
			}
		}
	}

	return
}
