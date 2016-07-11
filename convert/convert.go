package convert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/chiepomme/chienote/config"
	"github.com/dreampuf/evernote-sdk-golang/types"
	"github.com/yosssi/gohtml"
)

type noteForTemplate struct {
	GUID              *types.GUID
	Title             *string
	Content           *string
	RawContent        template.HTML
	ContentHash       []byte
	ContentLength     *int32
	Created           *types.Timestamp
	Updated           *types.Timestamp
	Deleted           *types.Timestamp
	Active            *bool
	UpdateSequenceNum *int32
	NotebookGuid      *string
	TagGuids          []string
	Resources         []*types.Resource
	Attributes        *types.NoteAttributes
	TagNames          []string
}

// Convert local cache to static files
func Convert() {
	os.Remove("index.html")

	fileinfos, err := ioutil.ReadDir(config.NoteCachePath)
	if err != nil {
		fmt.Println(err)
	}

	for _, fileinfo := range fileinfos {
		cachedNote := &types.Note{}
		jsonBytes, err := ioutil.ReadFile(config.NoteCachePath + fileinfo.Name())
		if err != nil {
			fmt.Println(err)
		}

		jsonErr := json.Unmarshal(jsonBytes, cachedNote)
		if jsonErr != nil {
			fmt.Println(jsonErr)
		}
		htmlTmpl, _ := template.ParseFiles("template/home.html")
		str := *cachedNote.Content
		reader := bytes.NewReader([]byte(str))
		doc, _ := goquery.NewDocumentFromReader(reader)
		doc.Find("en-todo[checked='true']").ReplaceWithHtml(`<input type="checkbox" checked="true" />`)
		doc.Find("en-todo").ReplaceWithHtml(`<input type="checkbox" />`)
		doc.Find("en-media").Each(func(i int, selection *goquery.Selection) {
			hash, _ := selection.Attr("hash")
			fis, _ := ioutil.ReadDir(config.ResourceCachePath)
			for _, fi := range fis {
				if strings.HasPrefix(fi.Name(), hash) {
					selection.ReplaceWithHtml(`<img src="` + config.ResourceCachePath + fi.Name() + `" />`)
					break
				}
			}
		})
		inNote, _ := doc.Find("en-note").Html()
		formatted := gohtml.Format(inNote)

		cachedNote.Content = &formatted

		note := noteForTemplate{
			Active:            cachedNote.Active,
			Attributes:        cachedNote.Attributes,
			Content:           cachedNote.Content,
			ContentHash:       cachedNote.ContentHash,
			ContentLength:     cachedNote.ContentLength,
			Created:           cachedNote.Created,
			Deleted:           cachedNote.Deleted,
			GUID:              cachedNote.GUID,
			NotebookGuid:      cachedNote.NotebookGuid,
			RawContent:        template.HTML(formatted),
			Resources:         cachedNote.Resources,
			TagGuids:          cachedNote.TagGuids,
			TagNames:          cachedNote.TagNames,
			Title:             cachedNote.Title,
			Updated:           cachedNote.Updated,
			UpdateSequenceNum: cachedNote.UpdateSequenceNum}

		var url string
		if cachedNote.Attributes.SourceURL != nil && *cachedNote.Attributes.SourceURL != "" {
			url = *cachedNote.Attributes.SourceURL
		} else {
			url = *cachedNote.Title
		}

		os.MkdirAll("article", os.ModePerm)
		file, err := os.OpenFile("article/"+url+".html", os.O_CREATE, os.ModePerm)
		defer file.Close()
		templateErr := htmlTmpl.ExecuteTemplate(file, "home", note)
		if templateErr != nil {
			fmt.Println("tempalte file error")
			fmt.Println(templateErr)
		}
	}
}
