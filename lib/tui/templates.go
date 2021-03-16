package tui

import (
	"github.com/manifoldco/promptui"
)

var listTemplates = &promptui.SelectTemplates{
	Label:    "{{ . }}",
	Active:   "\U0000261E {{ if .Mounted }}{{ .ShortId | blue | bold }} {{ .Name | blue | bold }} (mounted){{ else }}{{ .ShortId | bold }} {{ .Name | bold }}{{ end }}",
	Inactive: "  {{ if .Mounted }}{{ .ShortId | blue }} {{ .Name | blue }} (mounted){{else}}{{ .ShortId }} {{ .Name }}{{ end }}",
	Selected: "\U0000261E {{ .Id | red | cyan }} (selected)",
	Details: `
------ Container ------
Id: {{ .Id }}
Name:  {{ .Names }}
Image: {{ .Image }}
Command: {{ .Command }}
MountPoint: {{ .MountPoint }}`,
}
