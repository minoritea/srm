package srm

import (
	_ "embed"
	"text/template"
)

var (
	//go:embed template.tpl
	tmpl     string
	Template = template.Must(template.New("*SRM").Parse(tmpl))
)
