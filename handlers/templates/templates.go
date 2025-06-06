package templates

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

// Template cache
var Templates = make(map[string]*template.Template)

// Load templates on init
func InitTemplates() {
	templatesDir := "templates"
	log.Printf("Loading templates from: %s", templatesDir)

	layouts, err := filepath.Glob(filepath.Join(templatesDir, "base.html"))
	if err != nil || len(layouts) == 0 {
		log.Fatalf("Error loading base template: %v (found: %d)", err, len(layouts))
	}
	log.Printf("Found base template: %v", layouts)

	includes, err := filepath.Glob(filepath.Join(templatesDir, "*.html"))
	if err != nil {
		log.Fatalf("Error finding template includes: %v", err)
	}
	log.Printf("Found templates: %v", includes)

	funcMap := template.FuncMap{
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
	}

	for _, include := range includes {
		if filepath.Base(include) == "base.html" {
			continue
		}

		files := append([]string{include}, layouts...)
		fileName := filepath.Base(include)
		log.Printf("Loading template: %s with files: %v", fileName, files)
		Templates[fileName] = template.Must(template.New(fileName).Funcs(funcMap).ParseFiles(files...))
	}

	log.Printf("Templates loaded successfully. Count: %d", len(Templates))
	if _, ok := Templates["login.html"]; !ok {
		log.Fatalf("FATAL: login.html template not loaded correctly.")
	}
}

// Render a template given a model
func RenderTemplate(c echo.Context, tmpl string, data interface{}) error {
	log.Printf("Attempting to render template: %s", tmpl)
	
	t, ok := Templates[tmpl]
	if !ok {
		log.Printf("Template %s does not exist in map. Available templates: %v", tmpl, Templates)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Template %s not found.", tmpl))
	}

	var buf strings.Builder
	err := t.ExecuteTemplate(&buf, "base", data)
	if err != nil {
		log.Printf("Error executing template %s: %v", tmpl, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error rendering page.")
	}

	return c.HTML(http.StatusOK, buf.String())
} 