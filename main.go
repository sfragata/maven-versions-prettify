package main

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/html"
)

const arrow = "->"
const buildingText = "Building"
const infoTag = "[INFO]"
const emptyString = ""
const blankString = " "
const threePoints = "..."
const pipe = "|"
const mediatype = "text/html"

const htmlTemplate = `
<html>
<head>
	<style>
		.datagrid table { border-collapse: collapse; text-align: left; width: 100%; } 
		.datagrid {font: normal 12px/150% Arial, Helvetica, sans-serif; background: #fff; overflow: hidden; border: 1px solid #36752D; -webkit-border-radius: 3px; -moz-border-radius: 3px; border-radius: 3px; }
		.datagrid table td, 
		.datagrid table th { padding: 3px 10px; }
		.datagrid table thead th {background:-webkit-gradient( linear, left top, left bottom, color-stop(0.05, #36752D), color-stop(1, #275420) );background:-moz-linear-gradient( center top, #36752D 5%, #275420 100% );filter:progid:DXImageTransform.Microsoft.gradient(startColorstr='#36752D', endColorstr='#275420');background-color:#36752D; color:#FFFFFF; font-size: 15px; font-weight: bold; border-left: 1px solid #36752D; } 
		.datagrid table thead th:first-child { border: none; }
		.datagrid table tbody td { color: #275420; border-left: 1px solid #C6FFC2;font-size: 12px;font-weight: normal; }
		.datagrid table tbody .alt td { background: #DFFFDE; color: #275420; }
		.datagrid table tbody td:first-child { border-left: none; }
		.datagrid table tbody tr:last-child td { border-bottom: none; }
		.datagrid table tfoot td div { border-top: 1px solid #36752D;background: #DFFFDE;} 
		.datagrid table tfoot td { padding: 0; font-size: 12px } 
		.datagrid table tfoot td div{ padding: 2px; }
		.datagrid table tfoot td ul { margin: 0; padding:0; list-style: none; text-align: right; }
		.datagrid table tfoot  li { display: inline; }
		.datagrid table tfoot li a { text-decoration: none; display: inline-block;  padding: 2px 8px; margin: 1px;color: #FFFFFF;border: 1px solid #36752D;-webkit-border-radius: 3px; -moz-border-radius: 3px; border-radius: 3px; background:-webkit-gradient( linear, left top, left bottom, color-stop(0.05, #36752D), color-stop(1, #275420) );background:-moz-linear-gradient( center top, #36752D 5%, #275420 100% );filter:progid:DXImageTransform.Microsoft.gradient(startColorstr='#36752D', endColorstr='#275420');background-color:#36752D; }
		.datagrid table tfoot ul.active, .datagrid table tfoot ul a:hover { text-decoration: none;border-color: #275420; color: #FFFFFF; background: none; background-color:#36752D;}
		div.dhtmlx_window_active, div.dhx_modal_cover_dv { position: fixed !important; }
	</style>
</head>
<body>
	<h1>"Version Report"</h1>
	<h3>Dependencies that are not up-to-date:</h3>
	<div class="datagrid">
		<table>
			{{ range $key, $value := .}}
				<tr class="module">
					<th>{{ $key }}</th>
					<th>Current</th>
					<th>Latest</th>
				</tr>
				{{ range $value }}
					{{ $currentVersion := split .Version "."}}
					{{ $latestVersion := split .NewVersion "."}}
					
					{{ $currentMajorVersion := index $currentVersion 0 }}
					{{ $latestMajorVersion := index $latestVersion 0 }}
					
					{{ $lengthLatestVersion := len $latestVersion }} 
					{{ $latestPatchVersion :=  "0" }}
					{{ if eq $lengthLatestVersion 2 }}
						{{ $latestPatchVersion = index $latestVersion 1}}
					{{ else if ge $lengthLatestVersion 3 }}
						{{ $latestPatchVersion = index $latestVersion 2}}
					{{end}}

					{{ $latestMinorVersion :=  "0" }}
					{{ if ge $lengthLatestVersion 2 }}
						{{ $latestMinorVersion = index $latestVersion 1}}
					{{end}}

					{{ $lengthCurrentVersion := len $currentVersion }} 
					{{ $currentPatchVersion :=  "0" }}
					{{ if eq $lengthCurrentVersion 2 }}
						{{ $currentPatchVersion = index $currentVersion 1}}
					{{ else if ge $lengthCurrentVersion 3 }}
						{{ $currentPatchVersion = print (index $currentVersion 2)}}
					{{end}}

					{{ $currentMinorVersion :=  "0" }}
					{{ if ge $lengthCurrentVersion 2 }}
						{{ $currentMinorVersion = index $currentVersion 1}}
					{{end}}

					<tr>
						{{ $artifactSlice := split .Artifact ":" }}
						<td class="artifact"><a href="https://mvnrepository.com/artifact/{{index $artifactSlice 0}}/{{index $artifactSlice 1}}/{{.Version}}">{{ .Artifact }}</a></td>
						{{ $severity := "auto" }}
						{{ if ne $currentMajorVersion $latestMajorVersion }}
							{{ $severity = "red" }}
						{{ else if ne $currentMinorVersion $latestMinorVersion }}
							{{ $severity = "yellow" }}
						{{ else if ne $currentPatchVersion $latestPatchVersion }}
							{{ $severity = "powderblue" }}
						{{ end }}
						<td class="current" style="background-color:{{ $severity }}">{{ .Version }}</td>
						<td class="latest">{{ .NewVersion }}</td>
					</tr>
				{{ end }}
			{{ end }}
			</table>
	</div>
</body>
</html>
`

var dots = regexp.MustCompile("(\\.){2,}")

type MavenVersion struct {
	Artifact   string
	Version    string
	NewVersion string
}

func main() {
	var lines = make([]string, 0, 50)

	scanner := bufio.NewScanner(os.Stdin)
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	for scanner.Scan() {
		line := scanner.Text()
		// Only get INFO lines and the module header
		if strings.HasPrefix(line /*"[INFO]   "*/, infoTag) || strings.Contains(line /*"Building "*/, buildingText) {
			// Remove the INFO tag
			lines = append(lines, strings.Replace(line, infoTag, emptyString, 1))
		}
	}
	// Fix lines that has wrapped
	for index, line := range lines {
		if !strings.Contains(line, threePoints) && strings.Contains(line, arrow) {
			lines[index-1] = lines[index-1] + line
			lines[index] = emptyString
		}
	}

	maven := split(lines)
	template := template.Must(template.New("maven-version").Funcs(template.FuncMap{"split": func(string string, sep string) []string {
		return strings.Split(string, sep)
	}}).Parse(htmlTemplate))
	var resp bytes.Buffer
	err := template.Execute(&resp, maven)
	if err != nil {
		log.Println("executing template:", err)
	}

	htmlMinify := minify.New()
	htmlMinify.Add(mediatype, &html.Minifier{
		KeepEndTags:             false,
		KeepDefaultAttrVals:     false,
		KeepWhitespace:          false,
		KeepDocumentTags:        false,
		KeepConditionalComments: false,
	})
	minifiedString, err := htmlMinify.String(mediatype, resp.String())
	if err != nil {
		log.Println("executing minify:", err)
	} else {
		os.Stdout.WriteString(minifiedString)
	}
}

func split(oldSlice []string) map[string][]MavenVersion {
	var maven = make(map[string][]MavenVersion)
	var key string
	for _, line := range oldSlice {
		if strings.Contains(line, arrow) || strings.Contains(line, buildingText) {
			newLine := dots.ReplaceAllString(line, pipe)
			newLine = strings.ReplaceAll(newLine, arrow, pipe)
			newLine = strings.ReplaceAll(newLine, buildingText, emptyString)
			values := strings.Split(newLine, pipe)
			if len(values) == 1 {
				entry := strings.Split(strings.TrimSpace(values[0]), blankString)
				key = strings.TrimSpace(entry[0])
			} else {
				maven[key] = append(maven[key], MavenVersion{
					Artifact:   strings.TrimSpace(values[0]),
					Version:    strings.TrimSpace(values[1]),
					NewVersion: strings.TrimSpace(values[2]),
				})
			}

		}

	}
	return maven
}
