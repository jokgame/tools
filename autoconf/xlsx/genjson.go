package xlsx

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
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
	var files = make(map[string]*FileInfo)
	var exportedFiles = make(map[string][]*FileInfo)
	for _, file := range pkg.Files {
		dir := filepath.Join(config.Getenv("xlsxdir"), trimFilenameSuffix(filepath.Base(file.Filename)))
		for _, bean := range file.Beans {
			if bean.Kind != "protocol" {
				continue
			}
			if bean.GetTag("excel") == "false" {
				continue
			}
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
				key := nodes[0].Key(pkg)
				value := nodes[0].Value(pkg)
				if key != "" && value != nil {
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
						"value": map[string]interface{}{},
					}
				} else if len(values) > 1 {
					return fmt.Errorf("singleton %s has more than one values", bean.Name)
				} else {
					result = map[string]interface{}{
						"value": values[0],
					}
				}
			} else {
				result = map[string]interface{}{
					"indexes": indexes,
					"values":  values,
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
			tagExports := bean.GetTag("export")
			exports := strings.Split(tagExports, ",")
			if tagExports == "" {
				exports = defaultExports
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
					return err
				}
				var manifest = config.Getenv("manifest-" + export)
				filename = bean.Name
				if manifest != "" {
					checksum := fmt.Sprintf("%02x", sha256.Sum256(data))
					if _, ok := files[bean.Name]; ok {
						return fmt.Errorf("bean %s duplicated", bean.Name)
					}
					filename += "." + checksum
					file := &FileInfo{
						Name:     bean.Name,
						Checksum: checksum,
						Filename: filename,
					}
					files[bean.Name] = file
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
		var content, err = json.Marshal(manifestData)
		if err != nil {
			return fmt.Errorf("marshal manifest error: %w", err)
		}
		dir := filepath.Join(config.Outdir, export)
		if exportedDir := config.Getenv("exported-" + export + "-dir"); exportedDir != "" {
			dir = exportedDir
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		filename := filepath.Join(dir, manifest)
		if err := os.WriteFile(filename, content, 0666); err != nil {
			return fmt.Errorf("write file %s error: %w", manifest, err)
		}
	}
	return nil
}
