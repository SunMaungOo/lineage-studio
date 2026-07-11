package repo

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SunMaungOo/lineage-studio/internal/database"
)

type MetadataType string

const (
	TypeMssql MetadataType = "mssql"
)

type Metadata struct {
	Type MetadataType `json:"type"`
	Host string       `json:"host"`
	DB   string       `json:"db"`
}

type ObjectHistory struct {
	Hash  string `json:"hash"`
	Order int    `json:"order"`
}

type LinkType string

const (
	TypeView      LinkType = "view"
	TypeProcedure LinkType = "procedure"
)

type Link struct {
	Name    string          `json:"name"`
	Type    LinkType        `json:"type"`
	Current string          `json:"current"`
	History []ObjectHistory `json:"history"`
}

type ObjectInfo struct {
	Name        string     `json:"name"`
	Detail      string     `json:"detail"`
	DetailHash  string     `json:"detailHash"`
	Lineage     string     `json:"lineage"`
	LineageHash string     `json:"lineageHash"`
	CreatedAt   time.Time  `json:"createdAt"`
	Verified    bool       `json:"verified"`
	VerifiedAt  *time.Time `json:"verifiedAt,omitempty"`
	Hash        string     `json:"hash"`
}

type RepoDetail struct {
	Meta    Metadata
	Links   []Link
	Objects []ObjectInfo
}

type Repo struct {
	Name   string
	Detail RepoDetail
}

type ObjectDetail struct {
	Object ObjectInfo
	Type   LinkType
}

// objectName to add
// objectHash = hash of the object we wanted to add
// objectType is only used if it is the new object. Use nil to ignore it if link of the object already exist
// return new link with changed data
func AddObjectToLink(links []Link, objectName string, objectHash string, objectType *database.ObjectType) []Link {

	newLinks := make([]Link, len(links))

	copy(newLinks, links)

	isObjectExist := false

	for linkIndex, link := range newLinks {

		if link.Name == objectName {

			isObjectExist = true

			previousCurrent := link.Current

			newLinks[linkIndex].Current = objectHash

			oldHistory := link.History

			newHistory := make([]ObjectHistory, len(oldHistory)+1)

			// sort the history again

			for index, objectHistory := range oldHistory {

				newHistory[index] = ObjectHistory{
					Hash:  objectHistory.Hash,
					Order: objectHistory.Order + 1,
				}
			}

			newHistory[len(oldHistory)] = ObjectHistory{
				Hash:  previousCurrent,
				Order: 1,
			}

			newLinks[linkIndex].History = newHistory

		}

	}

	// if the object does not already exist in links , we have to add themn

	if !isObjectExist && objectType != nil {

		linkType := LinkType("view")

		if *objectType == "procedure" {
			linkType = LinkType("procedure")
		}

		newLinks = append(newLinks, Link{
			Name:    objectName,
			Type:    linkType,
			Current: objectHash,
			History: []ObjectHistory{},
		})

	}

	return newLinks

}

// Get Object which have the same hash

func GetObjectInfo(repoLocation string, repoName string, objectName string, objectHash string) (ObjectDetail, error) {

	linkLocation := filepath.Join(repoLocation, repoName, "links", objectName+".json")

	link, err := LoadLink(linkLocation)

	if err != nil {
		return ObjectDetail{}, err
	}

	objLocation := filepath.Join(repoLocation, repoName, "obj", objectName+"-"+objectHash+".json")

	objectInfo, err := LoadObject(objLocation)

	if err != nil {
		return ObjectDetail{}, err
	}

	return ObjectDetail{
		Object: objectInfo,
		Type:   link.Type,
	}, nil
}

func (initRepo Repo) GetCurrentObjectInfo(objectName string) (ObjectDetail, error) {

	var objectLink Link

	isFoundObject := false

	for _, link := range initRepo.Detail.Links {

		if link.Name == objectName {

			objectLink = link

			isFoundObject = true

			break
		}

	}

	if !isFoundObject {
		return ObjectDetail{}, errors.New("No object is found")
	}

	for _, object := range initRepo.Detail.Objects {

		if !(object.Name == objectName && object.Hash == objectLink.Current) {
			continue
		}

		return ObjectDetail{
			Object: object,
			Type:   objectLink.Type,
		}, nil
	}

	return ObjectDetail{}, fmt.Errorf("repo have missing object. Cannot find %v-%v", objectName, objectLink.Current)
}

func GetCurrentObjectInfo(repoLocation string, repoName string, objectName string) (ObjectDetail, error) {

	linkLocation := filepath.Join(repoLocation, repoName, "links", objectName+".json")

	link, err := LoadLink(linkLocation)

	if err != nil {
		return ObjectDetail{}, err
	}

	return GetObjectInfo(repoLocation, repoName, objectName, link.Current)

}

// get all the object which have the object name

func GetObjectInfoByName(repoLocation string, repoName string, objectName string) ([]ObjectDetail, error) {

	linkLocation := filepath.Join(repoLocation, repoName, "links", objectName+".json")

	link, err := LoadLink(linkLocation)

	if err != nil {
		return []ObjectDetail{}, err
	}

	objectHashes := make([]string, len(link.History)+1)
	objectHashes[0] = link.Current

	for index, history := range link.History {
		objectHashes[index+1] = history.Hash
	}

	objectDetails := make([]ObjectDetail, len(objectHashes))

	for index, hash := range objectHashes {

		objectDetail, err := GetObjectInfo(repoLocation, repoName, objectName, hash)

		if err != nil {
			return []ObjectDetail{}, err
		}

		objectDetails[index] = objectDetail

	}

	return objectDetails, nil

}

func GetCurrentObjectInfos(initRepo *Repo) []ObjectDetail {

	currentObjects := []ObjectDetail{}

	if initRepo == nil {
		return []ObjectDetail{}
	}

	currentHash := map[string]struct {
		hash     string
		linkType LinkType
	}{}

	for _, link := range initRepo.Detail.Links {

		currentHash[link.Name] = struct {
			hash     string
			linkType LinkType
		}{
			link.Current,
			link.Type,
		}

	}

	for _, object := range initRepo.Detail.Objects {

		linkValue, isExist := currentHash[object.Name]

		if !isExist {
			continue
		}

		if object.Hash != linkValue.hash {
			continue
		}

		currentObjects = append(currentObjects, ObjectDetail{
			Object: object,
			Type:   linkValue.linkType,
		})
	}

	return currentObjects
}

func SaveRepo(repoRootPath string, initRepo Repo, isOverwrite bool) error {

	repoRootPath = filepath.Clean(repoRootPath)

	_, err := os.Stat(repoRootPath)

	if err != nil {
		return err
	}

	repoLocation := filepath.Join(repoRootPath, initRepo.Name)

	_, err = os.Stat(repoLocation)

	if err == nil && !isOverwrite {
		return fmt.Errorf("%v folder already exist.Cannot save the repo without overwrite flag turn on", repoLocation)
	}

	err = saveMetadata(repoLocation, initRepo.Detail.Meta)

	if err != nil {
		return err
	}

	err = saveObjects(repoLocation, initRepo.Detail.Objects)

	if err != nil {
		return err
	}

	err = saveLinks(repoLocation, initRepo.Detail.Links)

	if err != nil {
		return err
	}

	return nil
}

func saveFile(fileLocation string, data []byte) error {

	dir := filepath.Dir(fileLocation)

	err := os.MkdirAll(dir, 0777)

	if err != nil {
		return err
	}

	err = os.WriteFile(fileLocation, data, 0666)

	if err != nil {
		return err
	}

	return nil
}

func saveMetadata(repoDirPath string, meta Metadata) error {

	marshal, err := json.Marshal(meta)

	if err != nil {
		return err
	}

	return saveFile(filepath.Join(repoDirPath, "meta.json"), marshal)
}

func saveObjects(repoDirPath string, objects []ObjectInfo) error {

	for _, object := range objects {

		marshal, err := json.Marshal(object)

		if err != nil {
			return err
		}

		fileName := fmt.Sprintf("%v-%v.json", object.Name, object.Hash)

		err = saveFile(filepath.Join(repoDirPath, "obj", fileName), marshal)

		if err != nil {
			return err
		}

	}
	return nil
}

func saveLinks(repoDirPath string, links []Link) error {

	for _, link := range links {

		marshal, err := json.Marshal(link)

		if err != nil {
			return err
		}

		fileName := fmt.Sprintf("%v.json", link.Name)

		err = saveFile(filepath.Join(repoDirPath, "links", fileName), marshal)

		if err != nil {
			return err
		}

	}
	return nil
}

func LoadRepos(repoRootPath string) ([]Repo, error) {

	repos := []Repo{}

	folderNames, err := GetFolderNames(repoRootPath)

	if err != nil {
		return nil, err
	}

	for _, folderName := range folderNames {

		repo, err := LoadRepo(repoRootPath, folderName)

		if err != nil {
			return nil, err
		}

		repos = append(repos, repo)
	}

	return repos, nil
}

func LoadRepo(repoRootPath string, folderName string) (Repo, error) {

	repoDirPath := filepath.Join(repoRootPath, folderName)

	meta, err := LoadMetadata(repoDirPath)

	if err != nil {
		return Repo{}, err
	}

	links, err := loadLinks(repoDirPath)

	if err != nil {
		return Repo{}, err
	}

	objects, err := loadObjects(repoDirPath)

	if err != nil {
		return Repo{}, err
	}

	return Repo{
		Name: folderName,
		Detail: RepoDetail{
			Meta:    meta,
			Links:   links,
			Objects: objects,
		},
	}, nil
}

func LoadMetadata(repoDirPath string) (Metadata, error) {

	var meta Metadata

	metaPath := filepath.Join(repoDirPath, "meta.json")
	metaPath = filepath.Clean(metaPath)

	_, err := os.Stat(metaPath)

	if err != nil {
		return Metadata{}, err
	}

	metaData, err := os.ReadFile(metaPath)

	if err != nil {
		return Metadata{}, err
	}

	err = json.Unmarshal(metaData, &meta)

	if err != nil {
		return Metadata{}, err
	}

	return meta, nil

}

func loadObjects(repoDirPath string) ([]ObjectInfo, error) {

	objPath := filepath.Join(repoDirPath, "obj")
	objPath = filepath.Clean(objPath)

	_, err := os.Stat(objPath)

	if err != nil {
		return []ObjectInfo{}, err
	}

	entries, err := os.ReadDir(objPath)

	if err != nil {
		return []ObjectInfo{}, err
	}

	objects := []ObjectInfo{}

	for _, entry := range entries {

		if entry.IsDir() {
			continue
		}

		objFileLocation := filepath.Clean(filepath.Join(objPath, entry.Name()))

		object, err := LoadObject(objFileLocation)

		if err != nil {
			return []ObjectInfo{}, err
		}

		objects = append(objects, object)

	}

	return objects, nil

}

func LoadObject(objFileLocation string) (ObjectInfo, error) {

	objData, err := os.ReadFile(objFileLocation)

	if err != nil {
		return ObjectInfo{}, err
	}

	var object ObjectInfo

	err = json.Unmarshal(objData, &object)

	if err != nil {
		return ObjectInfo{}, err
	}

	return object, nil
}

func loadLinks(repoDirPath string) ([]Link, error) {

	linkPath := filepath.Join(repoDirPath, "links")
	linkPath = filepath.Clean(linkPath)

	_, err := os.Stat(linkPath)

	if err != nil {
		return []Link{}, err
	}

	entries, err := os.ReadDir(linkPath)

	if err != nil {
		return []Link{}, err
	}

	links := []Link{}

	for _, entry := range entries {

		if entry.IsDir() {
			continue
		}

		linkFileLocation := filepath.Clean(filepath.Join(linkPath, entry.Name()))

		link, err := LoadLink(linkFileLocation)

		if err != nil {
			return []Link{}, err
		}

		links = append(links, link)

	}

	return links, nil

}

func LoadLink(linkFileLocation string) (Link, error) {

	linkData, err := os.ReadFile(linkFileLocation)

	if err != nil {
		return Link{}, err
	}

	var link Link

	err = json.Unmarshal(linkData, &link)

	if err != nil {
		return Link{}, err
	}

	return link, nil
}

func GetFolderNames(dirPath string) ([]string, error) {

	folderNames := []string{}

	cleanDirPath := filepath.Clean(dirPath)

	info, err := os.Stat(cleanDirPath)

	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("The provided path:%v is not a folder", cleanDirPath)
	}

	entries, err := os.ReadDir(cleanDirPath)

	if err != nil {
		return nil, err
	}

	for _, entry := range entries {

		if entry.IsDir() {
			folderNames = append(folderNames, entry.Name())
		}
	}

	return folderNames, nil
}
