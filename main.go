package main

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
	yaml "gopkg.in/yaml.v2"
)

const delim = "---"

type post struct {
	Title       string
	Published   bool
	Description string
	Tags        []string
	CoverImage  string
	Series      string
	PostBody    template.HTML
}

type index struct {
	Pages []Page
}

type Page struct {
	FileName string
	Title    string
}

// 读取文件
func getContents(s io.Reader) (b []byte, err error) {
	f, err := ioutil.ReadAll(s)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func parseFrontMatter(b []byte) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	err := yaml.Unmarshal(b, &m)
	if err != nil {
		msg := fmt.Sprintf("error:%v\ninput:\n%s", err, b)
		return nil, fmt.Errorf(msg)
	}
	return m, nil
}

// 切割头部，取出主要内容
// bytes.Split(s,sep) 按照sep切割，结果不包含sep
func splitData(f []byte) ([][]byte, error) {
	b := bytes.Split(f, []byte(delim))
	fmt.Println("b0:"+string(b[0]), "b1:"+string(b[1]), "b2:"+string(b[2]))
	if len(b) < 3 || len(b[0]) != 0 {
		return nil, fmt.Errorf("Front matter is damaged")
	}
	return b, nil
}

func isNil(i interface{}) bool {
	if i != nil {
		return false
	}
	return true
}

func makePost(fm map[string]interface{}, contents []byte, s [][]byte) (*template.Template, *post) {
	post := &post{}
	titleIntf, ok := fm["title"]
	if ok {
		title, ok := titleIntf.(string)
		if ok {
			post.Title = title
		} else {
			post.Title = ""
		}
	} else {
		post.Title = ""
	}
	pubIntf, ok := fm["published"]
	if ok {
		published, ok := pubIntf.(bool)
		if ok {
			post.Published = published
		} else {
			post.Published = false
		}
	} else {
		post.Published = false
	}
	descIntf, ok := fm["description"]
	if ok {
		description, ok := descIntf.(string)
		if ok {
			post.Description = description
		} else {
			post.Description = ""
		}
	} else {
		post.Description = ""
	}

	tagsIntf, ok := fm["tags"]
	if ok {
		tags, ok := tagsIntf.(string)
		if ok {
			post.Tags = strings.Split(tags, ", ")
		} else {
			post.Tags = []string{}
		}
	} else {
		post.Tags = []string{}
	}

	covIntf, ok := fm["cover_image"]
	if ok {
		coverImage, ok := covIntf.(string)
		if ok {
			post.CoverImage = coverImage
		} else {
			post.CoverImage = ""
		}
	} else {
		post.CoverImage = ""
	}

	seriesIntf, ok := fm["series"]
	if ok {
		series, ok := seriesIntf.(string)
		if ok {
			post.Series = series
		} else {
			post.Series = ""
		}
	} else {
		post.Series = ""
	}
	pBody := contents[len(s[1])+(len(delim)*2):]

	out := blackfriday.Run(pBody)

	bm := bluemonday.UGCPolicy()
	bm.AllowAttrs("class").Matching(regexp.MustCompile("^language-[a-zA-Z0-9]+$")).OnElements("code")
	post.PostBody = template.HTML(bm.SanitizeBytes(out))

	tm := template.Must(template.New("post").Parse(postTempl))
	return tm, post
}

func writeIndex(idx index) {
	indexFile, err := os.Create("index.html")
	if err != nil {
		panic(err)
	}
	defer indexFile.Close()
	buffer := bufio.NewWriter(indexFile)
	tm := template.Must(template.New("index").Parse(indexTempl))
	err = tm.Execute(buffer, idx)
	if err != nil {
		panic(err)
	}
	buffer.Flush()
}

var indexTempl = `<!DOCTYPE html>
<html lang="en">
  <head>
    <title>shindakun's dev site</title>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="HandheldFriendly" content="True">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="referrer" content="no-referrer-when-downgrade" />
    <meta name="description" content="shindakun's dev site" />
  </head>
  <body>
    <div class="index">
    {{ range $key, $value := .Pages }}
      <a href="./{{ $value.FileName }}">{{ $value.Title }}</a>
    {{ end }}
    </div>
  </body>
  </html>
`

var postTempl = `<!DOCTYPE html>
<html lang="en">
  <head>
    <title>{{.Title}}</title>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="HandheldFriendly" content="True">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="referrer" content="no-referrer-when-downgrade" />
    <meta name="description" content="{{.Description}}" />
  </head>
  <body>
    <div class="post">
      <h1>{{.Title}}</h1>
      {{.PostBody}}
    </div>
  </body>
  </html>
`

func main() {
	var idx index

	dir, err := ioutil.ReadDir(".")
	if err != nil {
		panic(err)
	}

	for _, file := range dir {
		if fileName := file.Name(); strings.HasSuffix(fileName, ".md") {

			openedFile, err := os.Open(fileName)
			if err != nil {
				panic(err)
			}

			contents, err := getContents(openedFile)
			if err != nil {
				panic(err)
			}
			s, err := splitData(contents)
			if err != nil {
				panic(err)
			}

			fm, err := parseFrontMatter(s[1])
			if err != nil {
				msg := fmt.Sprintf("error: %v\ninput:\n%s", err, s[1])
				panic(msg)
			}

			template, post := makePost(fm, contents, s)

			trimmedName := strings.TrimSuffix(fileName, ".md")
			outputFile, err := os.Create(trimmedName + ".html")
			if err != nil {
				panic(err)
			}
			defer outputFile.Close()

			buffer := bufio.NewWriter(outputFile)

			err = template.Execute(buffer, post)
			if err != nil {
				panic(err)
			}
			buffer.Flush()

			indexLinks := Page{
				FileName: trimmedName + ".html",
				Title:    post.Title,
			}
			idx.Pages = append(idx.Pages, indexLinks)
		}
	}
	writeIndex(idx)
}
