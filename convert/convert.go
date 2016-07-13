package convert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
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
					lowerName := strings.ToLower(fi.Name())
					relativeResourcePath := "../" + strings.Replace(config.PublicResourcePath, config.PublicPath, "", 1)
					if strings.HasSuffix(lowerName, ".png") || strings.HasSuffix(lowerName, ".jpeg") {
						selection.ReplaceWithHtml(`<img src="` + relativeResourcePath + fi.Name() + `" />`)

					} else if strings.HasSuffix(lowerName, ".mp3") {
						selection.ReplaceWithHtml(`<audio src="` + relativeResourcePath + fi.Name() + `" />`)
					} else {
						selection.ReplaceWithHtml(`<a href="` + relativeResourcePath + fi.Name() + `">` + strings.Split(fi.Name(), "-")[1] + "</a>")
					}
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

		os.MkdirAll(config.PublicArticlePath, os.ModePerm)
		file, err := os.OpenFile(config.PublicArticlePath+url+".html", os.O_CREATE, os.ModePerm)
		defer file.Close()
		templateErr := htmlTmpl.ExecuteTemplate(file, "home", note)
		if templateErr != nil {
			fmt.Println("tempalte file error")
			fmt.Println(templateErr)
		}

		os.RemoveAll(config.PublicResourcePath)
		os.MkdirAll(config.PublicResourcePath, os.ModePerm)

		resourceInfos, err := ioutil.ReadDir(config.ResourceCachePath)
		if err != nil {
			fmt.Println(err)
		}

		for _, resourceInfo := range resourceInfos {
			sourcePath := config.ResourceCachePath + resourceInfo.Name()
			source, err := os.OpenFile(sourcePath, os.O_RDONLY, os.ModePerm)
			if err != nil {
				fmt.Println(err)
			}

			destPath := config.PublicResourcePath + resourceInfo.Name()
			dest, err := os.OpenFile(destPath, os.O_CREATE, os.ModePerm)
			if err != nil {
				fmt.Println(err)
			}

			_, copyErr := io.Copy(dest, source)
			if copyErr != nil {
				fmt.Println(copyErr)
			}
		}
	}
}
