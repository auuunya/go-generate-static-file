package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const DELIM = "---"

var (
	conteneTemp = `<!DOCTYPE html>
		<html lang="en">
		<head>
			<title>{{.title}}</title>
			<meta charset="utf-8">
			<meta http-equiv="X-UA-Compatible" content="IE=edge">
			<meta name="HandheldFriendly" content="True">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<meta name="referrer" content="no-referrer-when-downgrade" />
			<meta name="description" content="{{.description}}" />
		</head>
		<body>
			<div class="post">
			<h1>{{.title}}</h1>
			{{.body}}
			</div>
		</body>
		</html>
		`

	SplitError = errors.New("front matter is damaged")
)

// 读取文件
func getContents(s io.Reader) (b []byte, err error) {
	f, err := io.ReadAll(s)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// 切割头部，取出主要内容
// bytes.Split(s,sep) 按照sep切割，结果不包含sep
func splitByte(f []byte) ([][]byte, error) {
	b := bytes.Split(f, []byte(DELIM))
	if len(b) < 3 || len(b[0]) != 0 {
		return nil, SplitError
	}
	return b, nil
}

// 处理头部信息
func parseHeader(b []byte) map[string]interface{} {
	m := make(map[string]interface{})
	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// fmt.Printf("line: %#v\n", line)
		before, after, found := strings.Cut(line, ":")
		if found {
			// parseStr(before)
			after = strings.Trim(after, " ")
			after = strings.ReplaceAll(after, `\n`, "")
			after = strings.ReplaceAll(after, `"`, "")
			m[before] = after
		}
	}
	return m
}

func generatePost(m map[string]interface{}, buffer *bufio.Writer) {
	t, err := template.New("article").Parse(conteneTemp)
	if err != nil {
		fmt.Printf("template parse error: %v\n", err)
		panic(err)
	}
	err = t.Execute(buffer, m)
	if err != nil {
		fmt.Printf("template execute error: %v\n", err)
	}
	buffer.Flush()
}

func main() {
	dir, err := os.ReadDir("./mds")
	if err != nil {
		panic(err)
	}
	arts := make(map[int]map[string]interface{})
	for i, file := range dir {
		fmt.Printf("file: %v\n", file)
		filename := file.Name()
		if strings.HasPrefix(filename, "README") {
			continue
		}
		if strings.HasSuffix(filename, ".md") {
			f, err := os.Open(filepath.Join("mds", filename))
			if err != nil {
				panic(err)
			}
			defer f.Close()
			contents, err := getContents(f)
			if err != nil {
				panic(err)
			}
			s, err := splitByte(contents)
			if err != nil {
				panic(err)
			}
			trimmedName := strings.TrimSuffix(filename, ".md")
			outputFile, err := os.Create(filepath.Join("tpl", trimmedName+".html"))
			if err != nil {
				panic(err)
			}
			defer outputFile.Close()
			buffer := bufio.NewWriter(outputFile)
			head := parseHeader(s[1])
			head["body"] = string(s[len(s)-1])
			generatePost(head, buffer)
			arts[i+1] = head
		}
	}
}
