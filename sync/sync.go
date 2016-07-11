package sync

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/chiepomme/chienote/config"
	"github.com/deckarep/golang-set"
	"github.com/dreampuf/evernote-sdk-golang/client"
	"github.com/dreampuf/evernote-sdk-golang/notestore"
	"github.com/dreampuf/evernote-sdk-golang/types"
	"github.com/dreampuf/evernote-sdk-golang/userstore"
)

// Sync local cache from evernote server
func Sync() {
	cfg, err := config.GetConfig()
	if err != nil {
		fmt.Println("failed to load config.json: " + err.Error())
		return
	}

	c := client.NewClient(cfg.ClientKey, cfg.ClientSecret, cfg.GetEnvironment())
	us, err := c.GetUserStore()
	if err != nil {
		fmt.Println(err)
		return
	}

	versionOk, err := us.CheckVersion("chienote", userstore.EDAM_VERSION_MAJOR, userstore.EDAM_VERSION_MINOR)
	if !versionOk {
		fmt.Println("not correct version")
		return
	}
	if err != nil {
		fmt.Println("error occured on checking version")
		return
	}

	url, err := us.GetNoteStoreUrl(cfg.DeveloperToken)
	if err != nil {
		fmt.Println(err)
	}
	if len(url) < 1 {
		fmt.Println("invalid url")
	}

	ns, err := c.GetNoteStoreWithURL(url)
	if err != nil {
		fmt.Println(err)
	}

	notebook, err := ns.GetDefaultNotebook(cfg.DeveloperToken)
	if err != nil {
		fmt.Println(err)
	}
	if notebook == nil {
		fmt.Println("invalid note")
	}

	ascending := false
	filter := &notestore.NoteFilter{NotebookGuid: notebook.GUID, Ascending: &ascending}
	notes, err := ns.FindNotes(cfg.DeveloperToken, filter, 0, 100)

	os.MkdirAll(config.NoteCachePath, os.ModePerm)
	os.MkdirAll(config.ResourceCachePath, os.ModePerm)

	existingIds := mapset.NewSet()
	fileInfos, err := ioutil.ReadDir(config.NoteCachePath)
	if err != nil {
		fmt.Println("can't read cache directory " + config.NoteCachePath)
	}

	for _, fileInfo := range fileInfos {
		if strings.HasSuffix(fileInfo.Name(), ".json") {
			existingIds.Add(strings.Replace(fileInfo.Name(), ".json", "", 1))
		}
	}

	for _, note := range notes.GetNotes() {
		fmt.Printf("processing %v[%v]\n", *note.Title, *note.GUID)

		cachePath := config.NoteCachePath + string(*note.GUID) + ".json"
		cacheReader, err := os.OpenFile(cachePath, os.O_RDONLY, os.ModePerm)
		if err != nil {
			fmt.Println("cant open file")
		}
		cachedJSON, err := ioutil.ReadAll(cacheReader)
		if err != nil {
			fmt.Println("cant read file")
		}
		cachedNote := &types.Note{}
		jsonErr := json.Unmarshal(cachedJSON, cachedNote)
		if jsonErr != nil {
			fmt.Println("corrupted file")
		}

		titleUpdated := cachedNote.Title == nil || *cachedNote.Title != *note.Title
		contentUpdated := cachedNote.Content == nil || !bytes.Equal(cachedNote.ContentHash, note.ContentHash)

		if titleUpdated || contentUpdated {
			note, err := ns.GetNote(cfg.DeveloperToken, *note.GUID, true, false, false, false)
			if err == nil {
				jsonBytes, err := json.Marshal(note)
				if err != nil {
					fmt.Println(err)
					continue
				}

				ioErr := ioutil.WriteFile(cachePath, jsonBytes, os.ModePerm)
				if ioErr != nil {
					fmt.Println(ioErr)
				}

				localResourceMap := map[types.GUID][]byte{}
				for _, cachedResource := range cachedNote.Resources {
					localResourceMap[*cachedResource.GUID] = cachedResource.Data.BodyHash
				}

				for _, resource := range note.Resources {
					fetchingNeeded := false
					cachedHash, exists := localResourceMap[*resource.GUID]
					if exists {
						if !bytes.Equal(cachedHash, resource.Data.BodyHash) {
							fetchingNeeded = true
						}
						delete(localResourceMap, *resource.GUID)
					} else {
						fetchingNeeded = true
					}

					if fetchingNeeded {
						resourceWithBytes, err := ns.GetResource(cfg.DeveloperToken, *resource.GUID, true, true, true, false)
						if err != nil {
							fmt.Println(err)
						}

						path := config.ResourceCachePath + hex.EncodeToString(resource.Data.BodyHash) + "-" + *resourceWithBytes.Attributes.FileName
						ioutil.WriteFile(path, resourceWithBytes.Data.Body, os.ModePerm)
						fmt.Println("write resource to " + path)
					}
				}

				/*
					for guid := range localResourceMap {
						fileInfos2, err := ioutil.ReadDir(config.ResourceCachePath)
						if err != nil {
							fmt.Println(err)
						}

						for _, fileInfo := range fileInfos2 {
							if strings.HasPrefix(fileInfo.Name(), string(guid)) {
								removeErr := os.Remove(config.ResourceCachePath + fileInfo.Name())
								if removeErr != nil {
									fmt.Println(removeErr)
								}
							}
						}
					}
				*/
			}
		}

		existingIds.Remove(*note.GUID)
	}

	for _, id := range existingIds.ToSlice() {
		guid := id.(string)
		os.Remove(config.NoteCachePath + guid + ".json")
	}
}
