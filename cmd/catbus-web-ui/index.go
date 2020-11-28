// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"html/template"
	"log"
	"strings"

	"go.eth.moe/catbus-web-ui/home"
)

var (
	funcs = map[string]interface{}{
		"controlTmpl": controlTmpl,
		"title": func(s string) string {
			return strings.Title(strings.Replace(s, "-", " ", -1))
		},
	}

	indexTmpl = template.Must(template.New("index.html").
			Funcs(funcs).
			Parse(`<!DOCTYPE html>
<html lang='en'>
<head>
  <meta charset='utf-8'>
  <meta name='viewport' content='width=device-width, initial-scale=1.0'>

  <link rel='icon' href='./icon.svg'>
  <!-- add to home screen for Safari on iOS. -->
  <meta name='apple-mobile-web-app-capable' content='yes'>
  <meta name='apple-mobile-web-app-status-bar-style' content='white'>
  <meta name='apple-mobile-web-app-title' content='Home'>
  <link rel='apple-touch-icon' href='./ios-icon.png'>

  <title>Home</title>
  <style>
    @media (prefers-color-scheme: dark) {
        body {
            background: #1f1f1f;
            color: #ddd;
        }
    }
    tr.device > td {
        padding-bottom: 1em;
    }
  </style>
</head>
<body>
  <h1>Home</h1>
  {{ range .Zones }}
  <section>
    <h2>{{ title .Name }}</h2>
    <table>
    {{ range .Devices }}
      <tr class='device'>
        <td>{{ title .Name }}</td>
        <td>
          <table>
          {{ range .Controls }}
            <tr>
              <td>{{ .Name }}</td>
              <td>{{ controlTmpl . }}</td>
            </tr>
          {{ end }}
          </table>
        </td>
      </tr>
    {{ end }}
    </table>
  </section>
  {{ end }}

  <script type='module'>
    import { addDefaultHooks, loadPage } from '/turbolinks.js';
    addDefaultHooks();

    const refresh = () =>
        loadPage( document.location )
	    .then( () => { console.log( 'refreshed page' ); } );

    document.addEventListener( 'focus', refresh );

    const handleInput = e => {
	const topic = e.target.id;
	let fd = new FormData();
	if ( e.target.tagName === 'INPUT' && e.target.type === 'checkbox' ) {
            if ( e.target.id.endsWith( '/power' ) ) {
                fd.append( 'value', e.target.checked ? 'on' : 'off' );
            } else {
                fd.append( 'value', e.target.checked ? 'yes' : 'no' );
            }
	} else {
            fd.append( 'value', e.target.value );
	}
	console.log( 'pushing ' + e.target.value + ' to ' + topic );
	fetch( '/' + topic, { method: 'POST', body: fd } ).then( () => console.log( 'updated' ) );
    };
    document.addEventListener( 'change', e => { handleInput( e ); refresh(); } );
    document.addEventListener( 'input', handleInput );
  </script>
</body>
</html>`))

	enumTmpl = template.Must(template.New("enum").Parse(`
{{ $value := .Value }}
<select id='{{ .Topic }}'>
{{ range .Values }}
  <option {{ if eq $value . }}selected{{ end}}>{{ . }}</option>
{{ end }}
</select>`))

	rangeTmpl = template.Must(template.New("range").Parse(
		"<input id='{{ .Topic }}' type='range' min='{{ .Min }}' max='{{ .Max }}' value='{{ .Value }}'>",
	))

	toggleTmpl = template.Must(template.New("toggle").Parse(
		"<input id='{{ .Topic }}' type='checkbox' {{ if .Value }}checked{{ end }}>",
	))
)

func controlTmpl(control home.Control) template.HTML {
	var w bytes.Buffer
	switch control := control.(type) {
	case *home.Enum:
		if err := enumTmpl.Execute(&w, control); err != nil {
			log.Printf("could not fill template: %v", err)
		}
	case *home.Range:
		if err := rangeTmpl.Execute(&w, control); err != nil {
			log.Printf("could not fill template: %v", err)
		}
	case *home.Toggle:
		if err := toggleTmpl.Execute(&w, control); err != nil {
			log.Printf("could not fill template: %v", err)
		}
	default:
		panic("unknown control type")
	}
	return template.HTML(w.String())
}
