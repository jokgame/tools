package xlsx

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gopherd/log"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/midlang/mid/src/mid/build"
)

func GenerateXlsx(plugin build.Plugin, config build.PluginRuntimeConfig, pkg *build.Package) error {
	for _, file := range pkg.Files {
		dir := filepath.Join(config.Outdir, trimFilenameSuffix(filepath.Base(file.Filename)))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		for _, bean := range file.Beans {
			if bean.Kind != "protocol" {
				continue
			}
			if bean.GetTag("excel") == "false" {
				continue
			}
			// 打开 excel 文件，如果文件不存在则新建一个
			filename := filepath.Join(dir, bean.Name+excelSuffix)
			sheetName := defaultSheetName
			isNew := false
			file, err := excelize.OpenFile(filename)
			if err != nil {
				if os.IsNotExist(err) {
					file = excelize.NewFile()
					file.Path = filename
					isNew = true
				} else {
					return err
				}
			}
			modified := true

			headers, err := makeHeader(pkg, bean, "", "", "")
			if err != nil {
				return err
			}
			rows := file.GetRows(sheetName)
			if len(rows) < 2 {
				for i := 0; i < len(rows); i++ {
					file.RemoveRow(sheetName, i)
				}
				file.InsertRow(sheetName, 0)
				file.InsertRow(sheetName, 1)
				for col, header := range headers {
					setHeader(file, sheetName, header, col)
				}
			} else {
				// 调整表结构

				// 取出当前的所有 comments
				comments := getComments(file, sheetName)

				headerMap := make(map[string]xlsxHeader)
				for _, header := range headers {
					headerMap[header.Comment] = header
				}

				// 根据已有的表头建立节点数
				nodes := buildJSONNodes(headerMap, pkg, bean, comments, rows[0])
				for i := 1; i < len(nodes); i++ {
					// nodes 从 1 开始的都标记为叶子节点,这些节点直接对应 excel 的各列
					nodes[i].userdata.s = columnName(i - 1)
					nodes[i].userdata.i = int64(i)
				}

				// 根据最新表头加入节点并记录下哪些是新的表头
				var newHeaders = map[*Node]int{}
				for i, header := range headers {
					contents := strings.Split(header.Comment, ".")
					next := nodes[0]
					for j := 0; j < len(contents); j++ {
						next = next.addChild(pkg, contents[j])
					}
					// 旧的节点都被标记了非 0 值
					// userdata.i 还等于 0 的是新节点
					if next.userdata.i == 0 {
						newHeaders[next] = i
						next.header = &header
						log.Debug().Printf("new header %v", header)
					}
				}
				nodes[0].sort(pkg)

				modified = isAnyHeaderChanged(file, sheetName, newHeaders, headers, nodes)
				if modified {

					moved := make(map[int]int)

					// 清空原来的表头
					file.RemoveRow(sheetName, 0)
					file.InsertRow(sheetName, 0)
					file.RemoveRow(sheetName, 1)
					file.InsertRow(sheetName, 1)
					rows = file.GetRows(sheetName)

					// 遍历节点,添加新表头到合适的位置
					nodes[0].visit(func(index int, node *Node) {
						if i, ok := newHeaders[node]; ok {
							log.Debug().Printf("insert new column for new header %v before %s", headers[i], columnName(index))
							setHeader(file, sheetName, headers[i], index)
						} else if node.header != nil {
							moved[int(node.userdata.i)-1] = index
							setHeader(file, sheetName, *(node.header), index)
						} else {
							log.Debug().Printf("header of node '%s' is nil", node.text)
						}
					})

					// 移动原来的数据
					for i := 2; i < len(rows); i++ {
						for j := len(rows[i]) - 1; j >= 0; j-- {
							if targetIndex, ok := moved[j]; ok && j != targetIndex {
								if n, err := strconv.ParseInt(rows[i][j], 10, 64); err == nil {
									file.SetCellValue(sheetName, cellName(i, targetIndex), n)
								} else if f, err := strconv.ParseFloat(rows[i][j], 64); err == nil {
									file.SetCellValue(sheetName, cellName(i, targetIndex), f)
								} else {
									file.SetCellStr(sheetName, cellName(i, targetIndex), rows[i][j])
								}
								file.SetCellStr(sheetName, cellName(i, j), "")
							}
						}
					}
				}
			}
			if modified {
				if isNew {
					log.Debug().Printf("create new excel file '%s'", filename)
				} else {
					log.Debug().Printf("excel file '%s' modified", filename)
				}
				if err := file.Save(); err != nil {
					log.Error().Printf("save excel file '%s' error: %v", filename, err)
					return err
				}
			}
		}
	}
	return nil
}

func isAnyHeaderChanged(file *excelize.File, sheetName string, newHeaders map[*Node]int, headers []xlsxHeader, nodes []*Node) bool {
	moved := make(map[int]int)
	modified := len(newHeaders) > 0
	// 遍历节点,添加新表头到合适的位置
	nodes[0].visit(func(index int, node *Node) {
		if i, ok := newHeaders[node]; ok {
			if isHeaderChanged(file, sheetName, headers[i], index) {
				modified = true
			}
		} else if node.header != nil {
			moved[int(node.userdata.i)-1] = index
			if isHeaderChanged(file, sheetName, *(node.header), index) {
				modified = true
			}
		} else {
			modified = true
		}
	})

	if modified == false {
		for from, to := range moved {
			if from != to {
				modified = true
				break
			}
		}
	}
	return modified
}

func isHeaderChanged(file *excelize.File, sheetName string, header xlsxHeader, index int) bool {
	return file.GetCellValue(sheetName, cellName(0, index)) != header.Comment ||
		file.GetCellValue(sheetName, cellName(1, index)) != header.Name
}

func setHeader(file *excelize.File, sheetName string, header xlsxHeader, index int) {
	// 首行用作标注
	file.SetCellStr(sheetName, cellName(0, index), header.Comment)

	// 第二行开始做标题行
	file.SetCellStr(sheetName, cellName(1, index), header.Name)

	return

	//FIXME
	if len(header.Enums) > 0 {
		dv := excelize.NewDataValidation(true)
		cname := columnName(index)
		dv.Sqref = fmt.Sprintf("%s%d:%s%d", cname, 3, cname, (1<<15)-2)
		var keys []string
		for _, e := range header.Enums {
			keys = append(keys, e.Desc)
		}
		dv.SetDropList(keys)
		file.AddDataValidation(sheetName, dv)
	}
}
