package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s\n", os.Args[0])
		flag.PrintDefaults()
	}

	addr := flag.String("addr", ":9090", "listen address of goast viewer")
	flag.Parse()

	http.HandleFunc("/", rootHandler)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		fmt.Println("err = ", err)
	}
}

var index string = `
<html>
<head>
	<title>goast viewer</title>
	<style>	p {	font-family:monospace;	}</style>
</head>
<body>
<form action="/">
	<pre><textarea name="gocode" rows="20" cols="80">%v</textarea></pre>
	<br>
	<input type="checkbox" name="expression">Is Go expression
	<br>
	<br>
	<input type="submit">
</form>
<br><pre>%v</pre><br>
</body>
</html>
`

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.Write([]byte(fmt.Sprintf("Error parse form = %v", err)))
		return
	}

	var gocode string
	gg := r.Form["gocode"]
	if len(gg) > 0 {
		gocode = gg[0]
	}
	if len(gocode) > 2 {
		if gocode[0] == '[' {
			gocode = gocode[1:]
		}
		if gocode[len(gocode)-1] == ']' {
			gocode = gocode[:len(gocode)-1]
		}
	}
	gocode = strings.TrimSpace(gocode)
	if gocode == "" {
		gocode = `
	package main

	func main(){
	}`
	}

	result := "Undefined"

	if st := r.Form["expression"]; len(st) == 1 && st[0] == "on" {
		a, err := parser.ParseExpr(gocode)
		if err != nil {
			w.Write([]byte(fmt.Sprintln(err)))
			return
		}
		var buf bytes.Buffer
		ast.Fprint(&buf, token.NewFileSet(), a, ast.NotNilFilter)
		w.Write(buf.Bytes())
		return
	}

	// gofmt gocode
	var dat []byte
	var filename string

	file, err := ioutil.TempFile("", "goast")
	if err != nil {
		goto NextStep
	}
	if _, err = file.WriteString(gocode); err != nil {
		goto NextStep
	}
	filename = file.Name()
	if err = file.Close(); err != nil {
		goto NextStep
	}

	if _, err = exec.Command("gofmt", "-w", filename).Output(); err != nil {
		goto NextStep
	}

	if dat, err = ioutil.ReadFile(filename); err != nil {
		goto NextStep
	}
	log.Printf("gofmt for Go code\n")
	gocode = string(dat)

NextStep:
	err = nil
	// add ast to result var
	log.Printf("parse ast tree\n")

	var buf bytes.Buffer
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", gocode, 0)
	if err != nil {
		fmt.Println("err = ", err)
		goto END
	}
	ast.Fprint(&buf, fset, f, ast.NotNilFilter)
	result = strings.Replace(buf.String(), "\n", "<br>", -1)

END:
	out := fmt.Sprintf(index, gocode, result)
	w.Write([]byte(out))
}
