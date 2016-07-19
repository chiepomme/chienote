package convert

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/PuerkitoBio/goquery"
	"github.com/dreampuf/evernote-sdk-golang/types"
	"github.com/pkg/errors"
	"github.com/yosssi/gohtml"
)

type frontMatter struct {
	Title     string   `yaml:"title,omitempty"`
	Layout    string   `yaml:"layout,omitempty"`
	Published bool     `yaml:"published"`
	Date      string   `yaml:"date,omitempty"`
	Tags      []string `yaml:"tags,omitempty"`
}

const cacheExtension = ".yml"

// Convert local cache to static files
func Convert(cacheRoot string, noteCacheDirName string, resourceCacheDirName string, jekyllRoot string, postsDirName string, resourcesDirName string, cleanNeeded bool) error {
	jekyllPostsDir := path.Join(jekyllRoot, postsDirName)
	jekyllResourcesDir := path.Join(jekyllRoot, resourcesDirName)
	noteCacheDir := path.Join(cacheRoot, noteCacheDirName)
	resourceCacheDir := path.Join(cacheRoot, resourceCacheDirName)

	notefiles, err := ioutil.ReadDir(noteCacheDir)
	if err != nil {
		return errors.Wrapf(err, "can't get cached notes", noteCacheDir)
	}

	resourceFiles, err := ioutil.ReadDir(resourceCacheDir)
	if err != nil {
		return errors.Wrapf(err, "can't get cached resources %v", resourceCacheDir)
	}

	createDestinations(cleanNeeded, &jekyllPostsDir, &jekyllResourcesDir)

	for _, notefile := range notefiles {
		cachedNote := &types.Note{}
		cachedNotePath := path.Join(noteCacheDir, notefile.Name())
		yamlBytes, err := ioutil.ReadFile(cachedNotePath)
		if err != nil {
			return errors.Wrapf(err, "can't read cached note file %v", cachedNotePath)
		}
		if err := yaml.Unmarshal(yamlBytes, cachedNote); err != nil {
			return errors.Wrapf(err, "can't unmarshal cached note file %v", cachedNotePath)
		}

		created := time.Unix(int64(*cachedNote.Created)/1000, 0)
		created = created.In(time.Local)

		html, err := replaceEvernoteTags(cachedNote.Content, &resourceFiles, &resourcesDirName)
		if err != nil {
			return errors.Wrapf(err, "can't replace evernote tags %v", cachedNotePath)
		}
		*html = strings.Replace(*html, "\u00a0", " ", -1)
		*html = gohtml.Format(*html)

		fm := frontMatter{
			Title:     *cachedNote.Title,
			Layout:    "post",
			Published: false,
			Date:      created.Format("2006-01-02 15:04:05 -0700"),
			Tags:      cachedNote.TagNames,
		}

		for i, tag := range fm.Tags {
			if tag == "published" {
				fm.Published = true
				fm.Tags = append(fm.Tags[:i], fm.Tags[i+1:]...)
				break
			}
		}

		for i, tag := range fm.Tags {
			if tag == "page" {
				fm.Layout = "page"
				fm.Tags = append(fm.Tags[:i], fm.Tags[i+1:]...)
				break
			}
		}

		fmyaml, err := yaml.Marshal(fm)
		if err != nil {
			return errors.Wrap(err, "can't create front matter")
		}

		*html = "---\n" + string(fmyaml) + "---\n" + *html

		var noteFileName string
		if cachedNote.Attributes.SourceURL != nil && *cachedNote.Attributes.SourceURL != "" {
			noteFileName = *cachedNote.Attributes.SourceURL
		} else {
			// TODO: need to sanitize title
			noteFileName = *cachedNote.Title
		}

		var notePath string
		if fm.Layout == "page" {
			notePath = path.Join(jekyllRoot, noteFileName+".html")
		} else {
			notePath = path.Join(jekyllPostsDir, created.Format("2006-01-02")+"-"+noteFileName+".html")
		}

		if err := ioutil.WriteFile(notePath, []byte(*html), os.ModePerm); err != nil {
			return errors.Wrapf(err, "can't create note file %v", notePath)
		}

		for _, resourceFile := range resourceFiles {
			copyResourceFile(resourceCacheDir, jekyllResourcesDir, resourceFile.Name())
		}
	}

	return nil
}

func createDestinations(needClean bool, jekyllPostsDir *string, jekyllResourcesDir *string) {
	if needClean {
		os.RemoveAll(*jekyllPostsDir)
		os.RemoveAll(*jekyllResourcesDir)
	}

	os.MkdirAll(*jekyllPostsDir, os.ModePerm)
	os.MkdirAll(*jekyllResourcesDir, os.ModePerm)
}

func replaceEvernoteTags(enml *string, resourceFiles *[]os.FileInfo, jekyllResourcesDirName *string) (*string, error) {
	// FIXME
	// standard library's html parser can't handle unknown self closing tags
	// https://github.com/golang/net/blob/master/html/parse.go#L727-L980
	// so replace en-medias to imgs to parse them
	*enml = strings.Replace(*enml, "<en-media", `<img en-media="true"`, -1)

	reader := bytes.NewReader([]byte(*enml))
	doc, _ := goquery.NewDocumentFromReader(reader)

	doc.Find("en-todo").Each(func(_ int, todo *goquery.Selection) {
		if _, exists := todo.Attr("checked"); exists {
			todo.ReplaceWithHtml(`<input type="checkbox" checked="checked"/>`)
		} else {
			todo.ReplaceWithHtml(`<input type="checkbox"/>`)
		}
	})

	for {
		codeOpen := doc.Find("div:contains(\\`\\`\\`)").First()
		if len(codeOpen.Nodes) == 0 {
			break
		}
		codeClose := codeOpen.NextAllFiltered("div:contains(\\`\\`\\`)").First()

		if len(codeClose.Nodes) == 0 {
			return nil, errors.Errorf("can't find code block end")
		}

		language := strings.Replace(codeOpen.Text(), "```", "", 1)
		codeLines := make([]string, 0, 10)
		codeLines = append(codeLines, `<div>{% highlight `+language+` %}`)

		codeOpen.NextUntilSelection(codeClose).Each(func(_ int, line *goquery.Selection) {
			codeLines = append(codeLines, line.Text())
			line.Remove()
		})

		codeLines = append(codeLines, `{% endhighlight %}</div>`)
		codeOpen.ReplaceWithHtml(strings.Join(codeLines, "\n"))
		codeClose.Remove()
	}

	doc.Find(`div:contains("#")`).Each(func(_ int, div *goquery.Selection) {
		line := strings.Replace(div.Text(), "\u00a0", " ", -1)
		if strings.HasPrefix(line, "##### ") {
			div.ReplaceWithHtml("<h5>" + strings.Replace(line, "##### ", "", 1))
		} else if strings.HasPrefix(line, "#### ") {
			div.ReplaceWithHtml("<h4>" + strings.Replace(line, "#### ", "", 1))
		} else if strings.HasPrefix(line, "### ") {
			div.ReplaceWithHtml("<h3>" + strings.Replace(line, "### ", "", 1))
		} else if strings.HasPrefix(line, "## ") {
			div.ReplaceWithHtml("<h2>" + strings.Replace(line, "## ", "", 1))
		} else if strings.HasPrefix(line, "# ") {
			div.ReplaceWithHtml("<h1>" + strings.Replace(line, "# ", "", 1))
		}
	})

	doc.Find("img[en-media]").Each(func(i int, selection *goquery.Selection) {
		hash, _ := selection.Attr("hash")
		found := false
		for _, resourceFile := range *resourceFiles {
			if !strings.HasPrefix(resourceFile.Name(), hash) {
				continue
			}

			found = true
			lowerName := strings.ToLower(resourceFile.Name())
			url := "{{ site.baseurl }}/" + path.Join(*jekyllResourcesDirName, resourceFile.Name())

			if strings.HasSuffix(lowerName, ".png") || strings.HasSuffix(lowerName, ".jpg") || strings.HasSuffix(lowerName, ".gif") {
				selection.ReplaceWithHtml(fmt.Sprintf(`<img src="%v" />`, url))
			} else if strings.HasSuffix(lowerName, ".mp3") {
				selection.ReplaceWithHtml(fmt.Sprintf(`<audio src="%v" controls="true"/>`, url))
			} else if strings.HasSuffix(lowerName, ".mp4") {
				selection.ReplaceWithHtml(fmt.Sprintf(`<video src="%v" />`, url))
			} else {
				selection.ReplaceWithHtml(fmt.Sprintf(`<a src="%v" />`, url))
			}
		}

		if !found {
			fmt.Printf("can't find resource %v\n", hash)
		}
	})

	innerNoteHTML, _ := doc.Find("en-note").Html()
	return &innerNoteHTML, nil
}

func copyResourceFile(from string, to string, fileName string) error {
	sourcePath := path.Join(from, fileName)
	source, err := os.OpenFile(sourcePath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "can't open resource cache file %v", sourcePath)
	}
	defer source.Close()

	destPath := path.Join(to, fileName)
	dest, err := os.OpenFile(destPath, os.O_CREATE, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "can't create resource file %v", sourcePath)
	}
	defer dest.Close()

	if _, err := io.Copy(dest, source); err != nil {
		return errors.Wrapf(err, "can't copy resource file %v", sourcePath)
	}

	return nil
}
