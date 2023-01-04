package xlsx

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/midlang/mid/src/mid/build"
)

type FileInfo struct {
	Name     string `json:"name"`
	Checksum string `json:"checksum"`
	Filename string `json:"filename"`
}

func GenerateJSON(plugin build.Plugin, config build.PluginRuntimeConfig, pkg *build.Package) error {
	var exportedFiles = make(map[string][]*FileInfo)
	var names = make(map[string]bool)
	var errorsTable = config.Getenv("errors-table")
	var stringsTable = config.Getenv("strings-table")
	for _, file := range pkg.Files {
		dir := filepath.Join(config.Getenv("xlsxdir"), trimFilenameSuffix(filepath.Base(file.Filename)))
		for _, bean := range file.Beans {
			if bean.Kind != "protocol" {
				continue
			}
			if bean.GetTag("excel") == "false" {
				continue
			}
			if names[bean.Name] {
				return fmt.Errorf("protocol %s duplicated", bean.Name)
			}
			names[bean.Name] = true
			filename := filepath.Join(dir, bean.Name+excelSuffix)
			sheetName := defaultSheetName
			file, err := excelize.OpenFile(filename)
			if err != nil {
				if os.IsNotExist(err) {
					log.Printf("excel file '%s' not found", filename)
					continue
				} else {
					return err
				}
			}

			tagExports := bean.GetTag("export")
			exports := strings.Split(tagExports, ",")
			if tagExports == "" {
				exports = defaultExports
			}
			rows := file.GetRows(sheetName)
			if len(rows) < 2 {
				log.Printf("empty excel file '%s'", filename)
				continue
			}

			comments := getComments(file, sheetName)

			values := make([]interface{}, 0, len(rows)-2)
			indexes := make(map[string]int)
			nodes := buildJSONNodes(nil, pkg, bean, comments, rows[0])
			nodes[0].sort(pkg)
			for i := 2; i < len(rows); i++ {
				var row = rows[i]
				for j := 0; j+1 < len(nodes); j++ {
					if j < len(row) {
						nodes[j+1].data = row[j]
					} else {
						nodes[j+1].data = ""
					}
				}
				key := strings.TrimSpace(nodes[0].Key(pkg))
				value := nodes[0].Value(pkg, false)
				if key != "" && value != nil {
					if _, dup := indexes[key]; dup {
						return fmt.Errorf("id %q duplicated in table %s", key, bean.Name)
					}
					indexes[key] = len(values)
					values = append(values, value)
				}
			}
			var singleton bool
			var result interface{}
			if tagSingleton := bean.GetTag("singleton"); tagSingleton != "" {
				var err error
				singleton, err = strconv.ParseBool(tagSingleton)
				if err != nil {
					return fmt.Errorf("invalid singleton tag of %s: %w", bean.Name, err)
				}
			}
			if singleton {
				if len(values) == 0 {
					result = map[string]interface{}{
						"row": map[string]interface{}{},
					}
				} else if len(values) > 1 {
					return fmt.Errorf("singleton %s has more than one values", bean.Name)
				} else {
					result = map[string]interface{}{
						"row": values[0],
					}
				}
			} else {
				result = map[string]interface{}{
					"rows": values,
				}
				if errorsTable == bean.Name {
					for _, export := range exports {
						var templateFilename = config.Getenv("errors-" + export + "-template")
						var outputFilename = config.Getenv("errors-" + export + "-output")
						if templateFilename != "" && outputFilename != "" {
							if err := generateFileByRows(values, templateFilename, outputFilename); err != nil {
								return fmt.Errorf("generate errors error: %v", err)
							}
						}
					}
				}
				if stringsTable == bean.Name {
					for _, export := range exports {
						var templateFilename = config.Getenv("strings-" + export + "-template")
						var outputFilename = config.Getenv("strings-" + export + "-output")
						if templateFilename != "" && outputFilename != "" {
							if err := generateFileByRows(values, templateFilename, outputFilename); err != nil {
								return fmt.Errorf("generate strings error: %v", err)
							}
						}
					}
				}
			}

			var data []byte
			prefix := config.Getenv("jsonpreifx")
			indent := config.Getenv("jsonindent")
			if prefix == "" && indent == "" {
				data, err = json.Marshal(result)
			} else {
				data, err = json.MarshalIndent(result, prefix, indent)
			}
			if err != nil {
				log.Printf("converting excel file '%s' to json error: %v", filename, err)
				continue
			}
			exported := map[string]bool{}
			for _, export := range exports {
				if exported[export] {
					continue
				}
				exported[export] = true
				if export == "-" || export == "" {
					continue
				}
				dir := filepath.Join(config.Outdir, export)
				if exportedDir := config.Getenv("exported-" + export + "-dir"); exportedDir != "" {
					dir = exportedDir
				}
				if err := os.MkdirAll(dir, 0755); err != nil {
					return fmt.Errorf("mkdirall %s error: %w", dir, err)
				}
				var manifest = config.Getenv("manifest-" + export)
				filename = bean.Name
				if manifest != "" {
					checksum := fmt.Sprintf("%02x", sha256.Sum256(data))
					filename += "." + checksum
					file := &FileInfo{
						Name:     bean.Name,
						Checksum: checksum,
						Filename: filename + ".json",
					}
					exportedFiles[export] = append(exportedFiles[export], file)
				}
				filename += ".json"
				filename = filepath.Join(dir, filename)
				if err := os.WriteFile(filename, data, 0666); err != nil {
					return fmt.Errorf("write file %s error: %w", filename, err)
				}
			}
		}
	}
	for export, files := range exportedFiles {
		var manifest = config.Getenv("manifest-" + export)
		var manifestData struct {
			Files []*FileInfo `json:"files"`
		}
		manifestData.Files = files
		var content, err = json.MarshalIndent(manifestData, "", "    ")
		if err != nil {
			return fmt.Errorf("marshal manifest error: %w", err)
		}
		if err := createDirForFile(manifest); err != nil {
			return err
		}
		if err := os.WriteFile(manifest, content, 0666); err != nil {
			return fmt.Errorf("write file %s error: %w", manifest, err)
		}
	}
	return nil
}

func createDirForFile(filename string) error {
	dir, _ := filepath.Split(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdirall %s error: %w", dir, err)
	}
	return nil
}

func generateFileByRows(rows []interface{}, templateFilename, outputFilename string) error {
	t, err := template.ParseFiles(templateFilename)
	if err != nil {
		return err
	}
	if err := createDirForFile(outputFilename); err != nil {
		return err
	}
	out, err := os.Create(outputFilename)
	if err != nil {
		return err
	}
	defer out.Close()
	return t.Execute(out, rows)
}
