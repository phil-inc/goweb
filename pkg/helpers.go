package goweb

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var helperFuncs = template.FuncMap{
	"assetPath": assetPath,
	"stylesheetTag": func(file string) template.HTML {
		return css(file)
	},
	"javascriptTag": func(file string) template.HTML {
		return js(file)
	},
}

type stringMap struct {
	data sync.Map
}

var assetMap = stringMap{}

func (m *stringMap) Load(key string) (string, bool) {
	i, ok := m.data.Load(key)
	if !ok {
		return ``, false
	}
	s, ok := i.(string)
	return s, ok
}

// Store a string in the map
func (m *stringMap) Store(key string, value string) {
	m.data.Store(key, value)
}

func assetPath(file string) (string, error) {
	return assetPathFor(file), nil
}

func assetPathFor(file string) string {
	filePath, ok := assetMap.Load(file)
	if filePath == "" || !ok {
		filePath = file
	}
	return filepath.ToSlash(filepath.Join("/public/assets", filePath))
}

func loadManifest(manifest io.Reader) error {
	m := map[string]string{}

	if err := json.NewDecoder(manifest).Decode(&m); err != nil {
		return err
	}
	for k, v := range m {
		assetMap.Store(k, v)
	}
	return nil
}

func css(file string) template.HTML {
	filePath, ok := assetMap.Load(file)
	if filePath == "" || !ok {
		filePath = file
	}
	path := filepath.ToSlash(filepath.Join("views/assets/css", filePath))
	return template.HTML(fmt.Sprintf(`<link rel="stylesheet" href="/%s">`, path))
}

func js(file string) template.HTML {
	filePath, ok := assetMap.Load(file)
	if filePath == "" || !ok {
		filePath = file
	}
	path := filepath.ToSlash(filepath.Join("view/assets/js", filePath))
	return template.HTML(fmt.Sprintf(`<script type="text/javascript" src="/%s"></script>`, path))
}

func workingDirectory() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	basePath := ""
	switch runtime.GOOS {
	case "windows":
		parts := strings.Split(dir, "\\")
		basePath = parts[len(parts)-1]
	default:
		parts := strings.Split(dir, "/")
		basePath = parts[len(parts)-1]
	}

	fmt.Printf("Current working directory: %q\n", dir)
	fmt.Printf("Base directory name:      %q\n", basePath)

	return basePath
}

func DirectoryPath() string {
	viewsDirPath := workingDirectory()
	if viewsDirPath == "phil-pay" {
		viewsDirPath = "./web/views"
	} else {
		viewsDirPath = fmt.Sprintf("./views")
	}
	return viewsDirPath
}
