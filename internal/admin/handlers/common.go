package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/rs/zerolog/log"

	"publika-auction/internal/admin/templates"
)

func render(w http.ResponseWriter, r *http.Request, page string, data interface{}) {
	tmpl, err := template.New("").Funcs(templateFuncs()).ParseFS(templates.FS,
		"views/layout.html",
		"views/partials/*.html",
		"views/"+page,
	)
	if err != nil {
		log.Err(err).Str("page", page).Msg("template parse error")
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Err(err).Str("page", page).Msg("template execute error")
	}
}

func renderPartial(w http.ResponseWriter, r *http.Request, partial string, data interface{}) {
	tmpl, err := template.New("").Funcs(templateFuncs()).ParseFS(templates.FS,
		"views/partials/*.html",
	)
	if err != nil {
		log.Err(err).Str("partial", partial).Msg("partial parse error")
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, partial, data); err != nil {
		log.Err(err).Str("partial", partial).Msg("partial execute error")
	}
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"str": func(v interface{}) string { return fmt.Sprintf("%s", v) },
	}
}
