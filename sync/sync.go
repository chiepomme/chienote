package sync

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/deckarep/golang-set"
	"github.com/dreampuf/evernote-sdk-golang/client"
	"github.com/dreampuf/evernote-sdk-golang/notestore"
	"github.com/dreampuf/evernote-sdk-golang/types"
	"github.com/dreampuf/evernote-sdk-golang/userstore"
	"github.com/pkg/errors"
)

const minimumFetchIntervalSeconds = 15 * 60
const cacheExtension = ".yml"

// Sync local cache from evernote server
func Sync(cacheRoot string, noteCacheDirName string, resourceCacheDirName string, clientKey string, clientSecret string, developerToken string, isSandbox bool, notebookName string) error {
	cacheRoot = path.Clean(cacheRoot)
	noteCacheDir := path.Join(cacheRoot, noteCacheDirName)
	resourceCacheDir := path.Join(cacheRoot, resourceCacheDirName)

	if err := os.MkdirAll(noteCacheDir, os.ModePerm); err != nil {
		return errors.Wrapf(err, "couldn't create note cache path %v", noteCacheDir)
	}

	if err := os.MkdirAll(resourceCacheDir, os.ModePerm); err != nil {
		return errors.Wrapf(err, "couldn't create resource cache path %v", resourceCacheDir)
	}

	cli := client.NewClient(clientKey, clientSecret, getEnvironment(isSandbox))

	us, err := getUserStore(cli, isSandbox)
	if err != nil {
		return err
	}

	ns, err := getNoteStore(cli, us, &developerToken)
	if err != nil {
		return err
	}

	notUpdated, err := checkUpdate(ns, &cacheRoot, &developerToken)
	if err != nil {
		return err
	}
	if notUpdated {
		return errors.Errorf("note is not updated")
	}

	bookGUID, err := findNotebookGUID(&notebookName, cli, ns, &developerToken)
	if err != nil {
		return err
	}

	metadatas, err := findMetadatas(ns, bookGUID, &developerToken)
	if err != nil {
		return err
	}

	cachedIds, err := createCachedNoteIDMap(&noteCacheDir)
	if err != nil {
		return err
	}

	for _, note := range metadatas.GetNotes() {
		fmt.Printf("processing %v\n", note.GUID)

		noteCachePath := path.Join(noteCacheDir, string(note.GUID)+cacheExtension)
		cachedNote, err := readCachedNote(&noteCachePath, &note.GUID)
		if err != nil {
			return errors.Wrapf(err, "can't read cached note")
		}

		if cachedNote == nil || *cachedNote.UpdateSequenceNum != *note.UpdateSequenceNum {
			note, err := ns.GetNote(developerToken, note.GUID, true, false, false, false)
			if err != nil {
				return errors.Wrapf(err, "can't get note %v", *note.GUID)
			}

			tags, err := ns.GetNoteTagNames(developerToken, *note.GUID)
			if err != nil {
				return errors.Wrapf(err, "can't get note tags %v", *note.GUID)
			}
			note.TagNames = tags

			fmt.Printf("downloaded %v[%v]\n", *note.Title, *note.GUID)

			writeCachedNoteToFile(&noteCachePath, note)
			saveResources(&resourceCacheDir, cachedNote, note, ns, &developerToken)
		}

		cachedIds.Remove(note.GUID)
	}

	for _, id := range cachedIds.ToSlice() {
		guid := id.(string)
		os.Remove(noteCacheDir + guid + cacheExtension)
	}

	return nil
}

func getEnvironment(isSandbox bool) client.EnvironmentType {
	if isSandbox {
		return client.SANDBOX
	}
	return client.PRODUCTION
}

func getUserStore(cli *client.EvernoteClient, isSandbox bool) (*userstore.UserStoreClient, error) {
	us, err := cli.GetUserStore()
	if err != nil {
		return nil, errors.Wrapf(err, "can't get user store (environment: %v)", getEnvironment(isSandbox))
	}

	versionOk, err := us.CheckVersion("chienote", userstore.EDAM_VERSION_MAJOR, userstore.EDAM_VERSION_MINOR)
	if !versionOk {
		return nil, errors.New("user store isn't correct version")
	}
	if err != nil {
		return nil, errors.Wrap(err, "error occured on checking user store version")
	}

	return us, nil
}

func getNoteStore(cli *client.EvernoteClient, us *userstore.UserStoreClient, developerToken *string) (*notestore.NoteStoreClient, error) {
	url, err := us.GetNoteStoreUrl(*developerToken)
	if err != nil {
		return nil, errors.Wrap(err, "can't get note store url")
	}
	if url == "" {
		return nil, errors.New("empty notestore url received")
	}

	ns, err := cli.GetNoteStoreWithURL(url)
	if err != nil {
		return nil, errors.Wrap(err, "can't get note store")
	}

	return ns, nil
}

func findNotebookGUID(notebookName *string, cli *client.EvernoteClient, ns *notestore.NoteStoreClient, developerToken *string) (notebookGUID *types.GUID, err error) {
	notebooks, err := ns.ListNotebooks(*developerToken)
	if err != nil {
		return nil, errors.Wrap(err, "can't get notebook list")
	}

	for _, book := range notebooks {
		if *book.Name == *notebookName {
			notebookGUID = book.GUID
			break
		}
	}

	if notebookGUID == nil || *notebookGUID == "" {
		return nil, errors.Errorf("can't get notebook %v", *notebookName)
	}

	return notebookGUID, nil
}

func findMetadatas(ns *notestore.NoteStoreClient, bookGUID *types.GUID, developerToken *string) (metadatas *notestore.NotesMetadataList, err error) {
	ascending := false
	filter := &notestore.NoteFilter{NotebookGuid: bookGUID, Ascending: &ascending}

	var resultSpec notestore.NotesMetadataResultSpec
	metadatas, err = ns.FindNotesMetadata(*developerToken, filter, 0, 1000, &resultSpec)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't find notes")
	}
	return metadatas, err
}

func createCachedNoteIDMap(noteCachePath *string) (mapset.Set, error) {
	cachedIDs := mapset.NewSet()
	cacheFileInfos, err := ioutil.ReadDir(*noteCachePath)
	if err != nil {
		return nil, errors.Wrapf(err, "can't read cache directory %v", *noteCachePath)
	}

	for _, cacheFileInfo := range cacheFileInfos {
		if strings.HasSuffix(cacheFileInfo.Name(), cacheExtension) {
			cachedIDs.Add(strings.Replace(cacheFileInfo.Name(), cacheExtension, "", 1))
		}
	}

	return cachedIDs, nil
}

func readCachedNoteFile(noteCachePath *string) (*types.Note, error) {
	cacheReader, err := os.OpenFile(*noteCachePath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, errors.Wrapf(err, "can't open cached note file %v", *noteCachePath)
	}
	defer cacheReader.Close()

	cachedYAML, err := ioutil.ReadAll(cacheReader)
	if err != nil {
		return nil, errors.Wrapf(err, "can't read cached note file %v", *noteCachePath)
	}

	cachedNote := &types.Note{}
	if err := yaml.Unmarshal(cachedYAML, cachedNote); err != nil {
		return nil, errors.Wrapf(err, "can't parse cached yaml file %v", *noteCachePath)
	}

	return cachedNote, nil
}

func readCachedNote(noteCachePath *string, noteGUID *types.GUID) (cachedNote *types.Note, err error) {
	if _, err := os.Stat(*noteCachePath); err == nil {
		cachedNote, err = readCachedNoteFile(noteCachePath)
		if err != nil {
			return nil, errors.Wrapf(err, "error occurs when reading cached note file %v", *noteCachePath)
		}
	}

	return cachedNote, nil
}

func writeCachedNoteToFile(cachePath *string, note *types.Note) error {
	yamlBytes, err := yaml.Marshal(note)
	if err != nil {
		return errors.Wrapf(err, "can't marshal note as YAML %v", *cachePath)
	}

	if err := ioutil.WriteFile(*cachePath, yamlBytes, os.ModePerm); err != nil {
		return errors.Wrapf(err, "can't write marshaled note to %v", *cachePath)
	}

	return nil
}

func saveResources(resourceCacheDir *string, cachedNote *types.Note, receivedNote *types.Note, ns *notestore.NoteStoreClient, developerToken *string) error {
	localResourceMap := map[types.GUID]int32{}
	if cachedNote != nil {
		for _, cachedResource := range cachedNote.Resources {
			localResourceMap[*cachedResource.GUID] = *cachedResource.UpdateSequenceNum
		}
	}

	for _, resource := range receivedNote.Resources {
		fetchingNeeded := false
		cachedUpdateNum, exists := localResourceMap[*resource.GUID]
		if exists {

			if cachedUpdateNum != *resource.UpdateSequenceNum {
				fetchingNeeded = true
			}
			delete(localResourceMap, *resource.GUID)
		} else {
			fetchingNeeded = true
		}

		if fetchingNeeded {
			resourceWithBytes, err := ns.GetResource(*developerToken, *resource.GUID, true, true, true, false)
			if err != nil {
				return errors.Wrapf(err, "can't get resource %v", *resource.Attributes.FileName)
			}

			p := path.Join(*resourceCacheDir, hex.EncodeToString(resource.Data.BodyHash)+"-"+*resourceWithBytes.Attributes.FileName)
			ioutil.WriteFile(p, resourceWithBytes.Data.Body, os.ModePerm)
			fmt.Println("write resource to " + p)
		} else {
			fmt.Printf("not updated %v\n", *resource.Attributes.FileName)
		}
	}

	return nil
}

func checkUpdate(ns *notestore.NoteStoreClient, cacheDir *string, developerToken *string) (notUpdated bool, err error) {
	syncState, err := ns.GetSyncState(*developerToken)
	if err != nil {
		return false, errors.Wrap(err, "can't get sync state")
	}

	syncStatePath := path.Join(*cacheDir, "sync_state.yml")

	prevStateBytes, err := ioutil.ReadFile(syncStatePath)
	if err == nil {
		prevState := &notestore.SyncState{Uploaded: new(int64)}
		if err := yaml.Unmarshal(prevStateBytes, prevState); err == nil {
			if prevState.UpdateCount == syncState.UpdateCount {
				return true, nil
			}

			if syncState.CurrentTime-prevState.CurrentTime < minimumFetchIntervalSeconds*1000 {
				return false, errors.Errorf("you should wait at least %v seconds from last update", minimumFetchIntervalSeconds)
			}
		}
	}

	stateBytes, err := yaml.Marshal(syncState)
	if err != nil {
		return false, errors.Wrapf(err, "can't marshal sync state")
	}

	if err := ioutil.WriteFile(syncStatePath, stateBytes, os.ModePerm); err != nil {
		return false, errors.Wrapf(err, "can't write sync state %v", syncStatePath)
	}

	return false, nil
}
