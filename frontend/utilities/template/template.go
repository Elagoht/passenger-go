package template

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type TemplateManager struct {
	cache map[string]*template.Template
}

func NewTemplateManager() *TemplateManager {
	templateManager := &TemplateManager{
		cache: make(map[string]*template.Template),
	}

	err := templateManager.init("frontend/templates")
	if err != nil {
		panic(err)
	}

	return templateManager
}

func (templateManager *TemplateManager) init(root string) error {
	pagesRoot := filepath.Join(root, "pages")
	err := filepath.Walk(pagesRoot, func(
		path string,
		info os.FileInfo,
		err error,
	) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go.tmpl") {
			return nil
		}

		relPath, _ := filepath.Rel(pagesRoot, path) // auth/register.go.tmpl
		parts := strings.Split(relPath, string(os.PathSeparator))
		if len(parts) < 2 {
			return nil
		}

		layout := parts[0]
		name := strings.TrimSuffix(parts[1], ".go.tmpl")
		cacheKey := layout + "/" + name

		base := filepath.Join(root, "base/index.go.tmpl")
		layoutFile := filepath.Join(root, "layouts", layout+".go.tmpl")

		tmpl := template.Must(template.ParseFiles(base, layoutFile, path))
		templateManager.cache[cacheKey] = tmpl

		return nil
	})

	return err
}

func (templateManager *TemplateManager) Render(
	writer http.ResponseWriter,
	layout string,
	page string,
	data any,
) {
	key := layout + "/" + page
	tmpl, ok := templateManager.cache[key]
	if !ok {
		http.Error(writer, "template not found: "+key, http.StatusInternalServerError)
		return
	}

	err := tmpl.ExecuteTemplate(writer, "base", data)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}
