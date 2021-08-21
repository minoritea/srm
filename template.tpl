package {{ .PkgName }}

import (
	"database/sql"
)
{{ range .Models }}

func (row *{{ .Name }}) bind(rows *sql.Rows, columns []string) error {
	var (
		dest []interface{}
		{{- range $name, $_ := .Fields }}
		counter_of_{{ $name }} int
		{{- end }}
	)
	for _, name := range columns {
		switch name {
		{{- range $name, $fields := .Fields }}
		case "{{ $name }}":
			switch counter_of_{{ $name }} {
			{{- range $index, $field := $fields }}
			case {{ $index }}:
				dest = append(dest, &row{{ $field }})
				counter_of_{{ $name }}++
				continue
			{{- end }}
			}
			counter_of_{{ $name }}++
		{{- end }}
		}
		var i interface{}
		dest = append(dest, &i)
	}
	return rows.Scan(dest...)
}

type {{ .Name }}s []{{ .Name }}
func (srm *{{ .Name }}s) Bind(rows *sql.Rows, err error) error {
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	for rows.Next() {
		var srmRow {{ .Name }}
		err := srmRow.bind(rows, columns)
		if err != nil {
			return err
		}
		*srm = append(*srm, srmRow)
	}

	err = rows.Err()
	if err != nil {
		return err
	}
	return nil
}
{{ end -}}
