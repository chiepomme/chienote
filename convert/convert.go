package convert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/chiepomme/chienote/config"
	"github.com/dreampuf/evernote-sdk-golang/types"
	"github.com/yosssi/gohtml"
)

type pageForTemplate struct {
	HasPrevious bool
	PreviousURL string
	PageNumber  int
	HasNext     bool
	NextURL     string
	Notes       []noteForTemplate
}

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
	RelativeURL       string
}

type noteList []noteForTemplate

func (notes noteList) Len() int {
	return len(notes)
}

func (notes noteList) Less(i, j int) bool {
	return *notes[i].Created < *notes[i].Created
}

func (notes noteList) Swap(i, j int) {
	notes[i], notes[j] = notes[j], notes[i]
}

// Convert local cache to static files
func Convert() {
	os.Remove("index.html")

	fileinfos, err := ioutil.ReadDir(config.NoteCachePath)
	if err != nil {
		fmt.Println(err)
	}

	os.RemoveAll(config.PublicResourcePath)
	os.MkdirAll(config.PublicArticlePath, os.ModePerm)

	var notes noteList

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
		htmlTmpl, _ := template.ParseFiles("template/note.html")
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
					relativeResourcePath := "/" + strings.Replace(config.PublicResourcePath, config.PublicPath, "", 1)
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

		notePath := config.PublicArticlePath + url + ".html"
		note.RelativeURL = strings.Replace(config.PublicArticlePath, config.PublicPath, "", 1) + url + ".html"
		notes = append(notes, note)

		file, err := os.OpenFile(notePath, os.O_CREATE, os.ModePerm)
		defer file.Close()
		templateErr := htmlTmpl.ExecuteTemplate(file, "note", note)
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

	sort.Sort(notes)

	for pageIdx := 0; pageIdx*config.NotesPerPage < len(notes); pageIdx++ {
		page := pageForTemplate{
			HasPrevious: pageIdx > 0,
			HasNext:     (pageIdx+1)*config.NotesPerPage < len(notes),
			PageNumber:  pageIdx + 1,
		}

		if page.HasPrevious {
			if page.PageNumber == 2 {
				page.PreviousURL = ""
			} else {
				page.PreviousURL = strconv.Itoa(page.PageNumber-1) + ".html"
			}
		}

		if page.HasNext {
			page.NextURL = strconv.Itoa(page.PageNumber+1) + ".html"
		}

		initialNoteIndexOnPage := pageIdx * config.NotesPerPage
		finalNoteIndexOnPage := (pageIdx + 1*config.NotesPerPage)
		if finalNoteIndexOnPage > len(notes) {
			finalNoteIndexOnPage = len(notes)
		}

		for _, note := range notes[initialNoteIndexOnPage:finalNoteIndexOnPage] {
			page.Notes = append(page.Notes, note)
		}

		htmlTmpl, _ := template.ParseFiles("template/home.html")
		pagePath := config.PublicPath
		if page.PageNumber == 1 {
			pagePath += "index.html"
		} else {
			pagePath += strconv.Itoa(page.PageNumber) + ".html"
		}
		file, err := os.OpenFile(pagePath, os.O_CREATE, os.ModePerm)
		if err != nil {
			fmt.Println(err)
		}
		defer file.Close()

		templateErr := htmlTmpl.ExecuteTemplate(file, "home", page)
		if templateErr != nil {
			fmt.Println("tempalte file error")
			fmt.Println(templateErr)
		}
	}
}
