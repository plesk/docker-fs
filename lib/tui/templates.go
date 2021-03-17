package tui

import (
	"github.com/manifoldco/promptui"
)

var listTemplates = &promptui.SelectTemplates{
	Label:    "Select container to mount/unmount. {{ \"(use ^C to exit)\" | faint }}",
	Active:   "\U0000261E {{ if .Mounted }}{{ .ShortId | blue | bold }} {{ .Name | blue | bold }} (mounted){{ else }}{{ .ShortId | bold }} {{ .Name | bold }}{{ end }}",
	Inactive: "  {{ if .Mounted }}{{ .ShortId | blue }} {{ .Name | blue }} (mounted){{else}}{{ .ShortId }} {{ .Name }}{{ end }}",
	Details: `
------ Container ------
Id: {{ .Id }}
Name:  {{ .Names }}
Image: {{ .Image }}
Command: {{ .Command }}
{{ if .Mounted }}MountPoint: {{ .MountPoint }}{{ end }}`,
}

var confirmUnmountTemplates = &promptui.SelectTemplates{
	Label:    "{{ \"Unmount\" | red }} container {{ .Id | bold}} from {{ .Mp | bold }}",
	Active:   "\U0000261E {{ . | bold }}",
	Inactive: "  {{ . }}",
}
