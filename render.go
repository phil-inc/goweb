package goweb

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	logger "github.com/phil-inc/plog-ng/pkg/core"
)

// Render reads a template files, applies data, and writes the output to an http.ResponseWriter.
func Render(r *http.Request, w http.ResponseWriter, templateFiles []string, data map[string]interface{}) {
	w.Header().Set("Content-Type", "text/html")

	// nil is passed from handlers that do not need to pass data to the template
	if data == nil {
		data = make(map[string]interface{})
	}

	layoutFiles := []string{"index.html", "navbar.html"}
	layoutFiles = append(layoutFiles, templateFiles...)

	// Render nested templates
	if err := renderTemplates(w, data, layoutFiles...); err != nil {
		logErrorAndRespond(w, "error executing template", err)
	}
}

// renderTemplates executes templates and writes the output to an http.ResponseWriter.
func renderTemplates(w http.ResponseWriter, data map[string]interface{}, files ...string) error {
	tmpl := parseTemplates(files...)
	return tmpl.Execute(w, data)
}

// parseTemplates parses files, adds functions to the template, and returns a template.
func parseTemplates(files ...string) *template.Template {
	viewsDirPath := fmt.Sprintf("%s/templates", DirectoryPath())
	paths := make([]string, len(files))
	for i, file := range files {
		paths[i] = filepath.Join(viewsDirPath, file)
	}
	return template.Must(template.ParseFiles(paths...))
}

func logErrorAndRespond(w http.ResponseWriter, message string, err error) {
	logger.Errorf("%s: %v", message, err)
	w.WriteHeader(http.StatusInternalServerError)
}
