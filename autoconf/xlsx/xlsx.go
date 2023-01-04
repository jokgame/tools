package xlsx

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gopherd/log"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/midlang/mid/src/mid/build"
	"github.com/midlang/mid/src/mid/lexer"
)

// 默认的excel表单名称
const defaultSheetName = "Sheet1"

// excel 文件后缀
const excelSuffix = ".xlsx"

// 默认导出目标
var defaultExports = []string{"client", "server"}

// excel 标注的作者
var commentAuthor = "auto:"

// bool 型数据在 excel 选择中的显示选项
const yes = "是"
const no = "否"

var boolEnums = []enumValue{
	{
		Desc:  yes,
		Value: 1,
	},
	{
		Desc:  no,
		Value: 0,
	},
}

// 枚举值
type enumValue struct {
	Desc  string
	Value int
}

// excel 标题行单元
type xlsxHeader struct {
	Name    string
	Type    string
	Comment string
	Enums   []enumValue
}

func trimFilenameSuffix(filename string) string {
	dotIndex := strings.LastIndex(filename, ".")
	if dotIndex >= 0 {
		return filename[:dotIndex]
	}
	return filename
}

// NOTE: index 从 0 开始,只支持到 ZZ 这一列,即 index >= 0 && index < (26+26*26)
func columnName(index int) string {
	if index < 26 {
		return string('A' + index)
	}
	index -= 26
	return string('A'+index/26) + string('A'+index%26)
}

// NOTE: row, col 均从 0 开始
func cellName(row, col int) string {
	return fmt.Sprintf("%s%d", columnName(col), row+1)
}

func fieldName(field *build.Field) string {
	name, _ := field.Name()
	return name
}

func buildExcelComment(comment string) string {
	data, err := json.Marshal(struct {
		Author string `json:"author"`
		Text   string `json:"text"`
	}{
		Author: commentAuthor + " ",
		Text:   comment,
	})
	if err != nil {
		return ""
	}
	return string(data)
}

func isBasicType(t build.Type) bool {
	return t.IsString() || t.IsInt() || t.IsBool() || t.IsFloat()
}

func makeSuffix(i int) string {
	if i == 0 {
		return ""
	}
	return fmt.Sprintf("_%d", i)
}

func getCommentContent(comment string) string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(comment), "//"))
}

func buildBasicType(t build.Type) string {
	t2 := t.(*build.BasicType)
	return t2.Name
}

func allFieldsOfBean(pkg *build.Package, bean *build.Bean) []*build.Field {
	var fields []*build.Field
	for _, ext := range bean.Extends {
		t := ext.(*build.StructType)
		bean2 := pkg.FindBean(t.Name)
		if bean2 == nil {
			log.Fatal().Printf("extended type '%s' not found", t.Name)
		}
		fields = append(fields, allFieldsOfBean(pkg, bean2)...)
	}
	fields = append(fields, bean.Fields...)
	return fields
}

func keyFieldOfBean(pkg *build.Package, bean *build.Bean) *build.Field {
	fields := allFieldsOfBean(pkg, bean)
	var idField *build.Field
	for i := len(fields) - 1; i >= 0; i-- {
		if fields[i].GetTag("key") == "true" {
			return fields[i]
		}
		if idField == nil {
			name := fieldName(fields[i])
			if name == "id" || name == "ID" || name == "Id" {
				idField = fields[i]
			}
		}
	}
	if idField != nil {
		return idField
	}
	if len(fields) > 0 {
		return fields[0]
	}
	return nil
}

func makeHeader(pkg *build.Package, bean *build.Bean, context string, prefix, suffix string) ([]xlsxHeader, error) {
	var ret []xlsxHeader
	var fields = allFieldsOfBean(pkg, bean)
	for _, field := range fields {
		tag := field.GetTag("name")
		if tag == "" {
			tag = fieldName(field)
		}
		name := tag
		log.Debug().Printf("field.Comment: %s", field.Comment)
		if field.Comment != "" {
			if comment := getCommentContent(field.Comment); comment != "" {
				log.Debug().Printf("getCommentContent: %s", comment)
				name = comment
			}
		}

		t := field.Type
		if t.IsArray() {
			// 数组字段
			array := t.(*build.ArrayType)
			size, ok := build.ParseIntFromExpr(array.Size)
			if !ok || size < 1 || size >= 128 {
				return nil, fmt.Errorf("invalid size of array field '%s.%s::%s'", pkg.Name, bean.Name, fieldName(field))
			}

			if isBasicType(array.T) {
				elemType := buildBasicType(array.T)
				tmpContext := context + fmt.Sprintf("%s(%s[])", fieldName(field), elemType)
				var enums []enumValue
				if array.T.IsBool() {
					enums = boolEnums
				}
				// 基础类型的数组
				for i := 0; i < size; i++ {
					ret = append(ret, xlsxHeader{
						Name:    fmt.Sprintf("%s%s%s_%d", prefix, name, suffix, i+1),
						Comment: fmt.Sprintf("%s.%d", tmpContext, i),
						Enums:   enums,
					})
				}
			} else if array.T.IsStruct() {
				// 结构体字段或枚举的数组
				t2 := array.T.(*build.StructType)
				b2 := pkg.FindBean(t2.Name)
				if b2 == nil {
					return nil, fmt.Errorf("element type of array field '%s.%s::%s' not found", pkg.Name, bean.Name, fieldName(field))
				}
				_prefix := prefix + name
				tmpContext := context + fmt.Sprintf("%s(%s[])", fieldName(field), t2.Name)

				if b2.Kind == "enum" {
					// 枚举数组
					enums, err := buildEnumHeader(b2)
					if err != nil {
						return nil, err
					}
					for i := 0; i < size; i++ {
						_suffix := suffix + makeSuffix(i+1)
						ret = append(ret, xlsxHeader{
							Name:    _prefix + _suffix,
							Comment: fmt.Sprintf("%s.%d", tmpContext, i),
							Enums:   enums,
						})
					}
				} else if b2.Kind == "protocol" || b2.Kind == "struct" {
					// 结构体数组
					for i := 0; i < size; i++ {
						_suffix := suffix + makeSuffix(i+1)
						tmpHeaders, err := makeHeader(pkg, b2, fmt.Sprintf("%s.%d.", tmpContext, i), _prefix, _suffix)
						if err != nil {
							return nil, err
						}
						ret = append(ret, tmpHeaders...)
					}
				} else {
					return nil, fmt.Errorf("invalid element type of field '%s.%s::%s'", pkg.Name, bean.Name, fieldName(field))
				}
			}
		} else if t.IsStruct() {
			// 结构体字段或枚举
			t2 := t.(*build.StructType)
			b2 := pkg.FindBean(t2.Name)
			if b2 == nil {
				return nil, fmt.Errorf("type of field '%s.%s::%s' not found", pkg.Name, bean.Name, fieldName(field))
			}
			if b2.Kind == "enum" {
				tmpContext := context + fmt.Sprintf("%s(%s)", fieldName(field), t2.Name)
				// 枚举类型
				enums, err := buildEnumHeader(b2)
				if err != nil {
					return nil, err
				}
				ret = append(ret, xlsxHeader{
					Name:    prefix + name + suffix,
					Comment: tmpContext,
					Enums:   enums,
				})
			} else if b2.Kind == "protocol" || b2.Kind == "struct" {
				// 结构体
				tmpContext := context + fmt.Sprintf("%s(%s).", fieldName(field), t2.Name)
				tmpHeaders, err := makeHeader(pkg, b2, tmpContext, prefix+name, suffix)
				if err != nil {
					return nil, err
				}
				ret = append(ret, tmpHeaders...)
			} else {
				return nil, fmt.Errorf("invalid type of field '%s.%s::%s'", pkg.Name, bean.Name, fieldName(field))
			}
		} else if isBasicType(t) {
			tmpContext := context + fmt.Sprintf("%s(%s)", fieldName(field), buildBasicType(t))
			var enums []enumValue
			if t.IsBool() {
				enums = boolEnums
			}
			ret = append(ret, xlsxHeader{
				Name:    prefix + name + suffix,
				Comment: tmpContext,
				Enums:   enums,
			})
		} else {
			return nil, fmt.Errorf("unsupported type of field '%s.%s::%s'", pkg.Name, bean.Name, fieldName(field))
		}
	}
	return ret, nil
}

func descOfEnum(field *build.Field) string {
	desc, _ := field.Name()
	if field.Comment != "" {
		desc = getCommentContent(field.Comment)
	}
	return desc
}

func buildEnumHeader(bean *build.Bean) ([]enumValue, error) {
	var enums []enumValue
	for _, f := range bean.Fields {
		s := f.Value()
		v, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("invalid enum value '%s'", s)
		}
		desc := descOfEnum(f)
		enums = append(enums, enumValue{
			Value: v,
			Desc:  desc,
		})
	}
	return enums, nil
}

func nameOfComment(s string) string {
	index := strings.Index(s, "(")
	if index >= 0 {
		return s[:index]
	}
	return s
}

func typeOfComment(s string) string {
	start := strings.Index(s, "(") + 1
	end := strings.Index(s, ")")
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = len(s)
	}
	return s[start:end]
}

type NodeList []*Node

func (nl NodeList) getByKey(key string) *Node {
	for _, node := range nl {
		if node.text == key {
			return node
		}
	}
	return nil
}

func (nl *NodeList) setByKey(key string, node *Node) {
	for i, n := range *nl {
		if n.text == key {
			(*nl)[i] = node
			return
		}
	}
	node.text = key
	(*nl) = append((*nl), node)
}

func (nl NodeList) get(index int) *Node {
	return nl[index]
}

func (nl *NodeList) set(index int, node *Node) {
	(*nl)[index] = node
}

func (nl *NodeList) push(node *Node) {
	(*nl) = append((*nl), node)
}

func (nl NodeList) list() []*Node {
	return ([]*Node)(nl)
}

func (nl NodeList) Len() int { return len(nl) }

func (nl NodeList) findNodeByField(field *build.Field) *Node {
	for _, node := range nl {
		if nameOfComment(node.text) == fieldName(field) {
			return node
		}
	}
	return nil
}

type Node struct {
	// excel 标注文本内容
	text string
	// excel 单元格数据
	data string
	// 节点数据类型
	nodeType string
	// 节点数据结构
	// 对于 struct/protocol/enum 为该节点的Bean结构
	// 对于以上3者之一的 array 则为数组元素类型的 Bean 结构
	// 其他则 bean 为 nil
	bean *build.Bean

	// 父节点
	parent *Node
	// 子节点
	children *NodeList

	// 自定义数据
	userdata struct {
		s string
		i int64
	}
	// excel 头部
	header *xlsxHeader
}

func (node *Node) isInteger() bool {
	bt, ok := lexer.LookupType(node.nodeType)
	return ok && bt.IsInt()
}

func (node *Node) isFloat() bool {
	bt, ok := lexer.LookupType(node.nodeType)
	return ok && bt.IsFloat()
}

func (node *Node) isBool() bool {
	return node.nodeType == "bool"
}

func (node *Node) isString() bool {
	return node.nodeType == "string"
}

func (node *Node) isArray() bool {
	return strings.HasSuffix(node.text, "[])")
}

func (node *Node) isEnum() bool {
	return !node.isArray() && node.bean != nil && node.bean.Kind == "enum"
}

func (node *Node) addChild(pkg *build.Package, text string) *Node {
	var child *Node

	if node.isArray() {
		index, _ := strconv.Atoi(nameOfComment(text))
		if node.children == nil {
			node.children = new(NodeList)
		}
		for node.children.Len() <= index {
			node.children.push(nil)
		}
		child = node.children.get(index)
		if child == nil {
			child = new(Node)
			child.text = text
			child.parent = node
			child.nodeType = strings.TrimSuffix(node.nodeType, "[]")
			child.bean = node.bean
			node.children.set(index, child)
			log.Debug().Printf("add child '%s', nodeType=%s", text, child.nodeType)
		}
	} else {
		if node.children == nil {
			node.children = new(NodeList)
		}
		child = node.children.getByKey(text)
		if child == nil {
			child = new(Node)
			node.children.setByKey(text, child)
			child.parent = node
			child.nodeType = typeOfComment(text)
			child.bean = pkg.FindBean(strings.TrimSuffix(child.nodeType, "[]"))
			log.Debug().Printf("add child '%s', nodeType=%s", text, child.nodeType)
		}
	}

	return child
}

type Visitor func(index int, node *Node)

func (node *Node) visit(visitor Visitor) {
	if node.children != nil {
		index := 0
		for _, child := range node.children.list() {
			child.recVisit(&index, visitor)
		}
	}
}

func (node *Node) recVisit(index *int, visitor Visitor) {
	// 对于基础数据类型都直接对应 excel 中的列
	if node.isString() || node.isBool() || node.isFloat() || node.isInteger() || node.isEnum() {
		visitor(*index, node)
		*index = *index + 1
	}
	if node.children != nil {
		for _, child := range node.children.list() {
			child.recVisit(index, visitor)
		}
	}
}

func (node *Node) sort(pkg *build.Package) {
	if node.bean != nil && !node.isArray() && (node.bean.Kind == "protocol" || node.bean.Kind == "struct") {
		if node.children != nil && node.children.Len() > 0 {
			fields := allFieldsOfBean(pkg, node.bean)
			if node.parent == nil {
				log.Debug().Printf("fields.length=%d", len(fields))
			}
			orders := make(map[*Node]int)
			for i, child := range node.children.list() {
				found := false
				for j, field := range fields {
					if nameOfComment(child.text) == fieldName(field) {
						if node.parent == nil {
							log.Debug().Printf("order of node %s: %d", nameOfComment(child.text), j)
						}
						orders[child] = j
						found = true
						break
					}
				}
				if !found {
					orders[child] = len(fields) + i
				}
			}
			sort.Slice(*node.children, func(i, j int) bool {
				return orders[(*node.children)[i]] < orders[(*node.children)[j]]
			})
		}
	}
	if node.children != nil {
		for _, child := range node.children.list() {
			child.sort(pkg)
		}
	}
}

func (node *Node) Key(pkg *build.Package) string {
	if node.bean == nil || node.children == nil || node.children.Len() == 0 {
		return ""
	}
	keyField := keyFieldOfBean(pkg, node.bean)
	n := node.children.findNodeByField(keyField)
	if n != nil {
		var value = n.Value(pkg, true)
		if value != nil {
			return fmt.Sprintf("%v", value)
		}
	}
	return ""
}

func (node *Node) Value(pkg *build.Package, required bool) interface{} {
	log.Debug().Printf("data of node '%s': '%s'", node.text, node.data)
	if required && node.data == "" && !node.isString() {
		return nil
	}
	if node.isInteger() {
		i, _ := strconv.ParseInt(node.data, 10, 64)
		return i
	}
	if node.isFloat() {
		f, _ := strconv.ParseFloat(node.data, 64)
		return f
	}
	if node.isBool() {
		var str = strings.TrimSpace(node.data)
		if str == yes {
			return true
		} else {
			if b, err := strconv.ParseBool(str); err == nil {
				return b
			} else {
				if i, err := strconv.Atoi(str); err == nil {
					return i != 0
				}
				return false
			}
		}
	}
	if node.isString() {
		return node.data
	}
	if node.isArray() {
		values := make([]interface{}, 0)
		for i := 0; i < node.children.Len(); i++ {
			value := node.children.get(i).Value(pkg, required)
			if value == nil {
				log.Debug().Printf("%dth value of array is nil", i)
			} else {
				values = append(values, value)
			}
		}
		return values
	}
	if node.bean != nil {
		if node.bean.Kind == "protocol" || node.bean.Kind == "struct" {
			values := make(map[string]interface{})
			if node.children != nil {
				for _, v := range node.children.list() {
					value := v.Value(pkg, required)
					if value == nil {
						log.Debug().Printf("field '%s' is nil", v.text)
					} else {
						field := node.bean.FindFieldByName(nameOfComment(v.text))
						jsonKey := nameOfComment(v.text)
						if field == nil {
							values[jsonKey] = value
						} else {
							if tag := field.GetTag("name"); tag == "-" {
								jsonKey = ""
							} else if tag != "" {
								jsonKey = tag
							}
							if jsonKey != "" {
								values[jsonKey] = value
							}
						}
					}
				}
			}
			return values
		} else if node.bean.Kind == "enum" {
			data := strings.TrimSpace(node.data)
			if data == "" {
				return 0
			}
			if n, err := strconv.ParseInt(data, 10, 64); err == nil {
				return n
			}
			for _, f := range node.bean.Fields {
				desc := descOfEnum(f)
				if desc == strings.TrimSpace(data) {
					i, ok := build.ParseIntFromExpr(f.Default)
					if ok {
						return i
					}
					return nil
				}
			}
			return 0
		}
	}

	return nil
}

func buildJSONNodes(headers map[string]xlsxHeader, pkg *build.Package, bean *build.Bean, comments map[string]string, columns []string) []*Node {
	root := new(Node)
	root.bean = bean
	root.nodeType = bean.Name

	nodes := make([]*Node, 0, len(columns)+1)
	nodes = append(nodes, root)

	for i := 0; i < len(columns); i++ {
		ref := cellName(0, i)
		comment, ok := comments[ref]
		if !ok {
			log.Debug().Printf("comment of cell '%s' in bean '%s' not found", ref, bean.Name)
			break
		}

		commentText := strings.TrimSpace(comment)
		contents := strings.Split(commentText, ".")
		next := root
		for j := 0; j < len(contents); j++ {
			next = next.addChild(pkg, contents[j])
		}
		next.data = columns[i]
		if headers != nil {
			if header, ok := headers[commentText]; ok {
				next.header = &header
			}
		}
		nodes = append(nodes, next)
	}
	return nodes
}

func getComments(file *excelize.File, sheetName string) map[string]string {
	var comments = make(map[string]string)
	rows, err := file.Rows(sheetName)
	if err == nil && rows.Next() {
		columns := rows.Columns()
		for i := 0; i < len(columns); i++ {
			comments[cellName(0, i)] = columns[i]
		}
	}
	return comments
}
