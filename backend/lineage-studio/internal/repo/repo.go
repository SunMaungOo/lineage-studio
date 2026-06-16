package repo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
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
	Name        string    `json:"name"`
	Detail      string    `json:"detail"`
	DetailHash  string    `json:"detailHash"`
	Lineage     string    `json:"lineage"`
	LineageHash string    `json:"lineageHash"`
	CreatedAt   time.Time `json:"createdAt"`
	Verfied     bool      `json:"verified"`
	Hash        string    `json:"hash"`
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

		objectValue, isExist := currentHash[object.Name]

		if !isExist {
			continue
		}

		if object.Hash != objectValue.hash {
			continue
		}

		currentObjects = append(currentObjects, ObjectDetail{
			Object: object,
			Type:   objectValue.linkType,
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

	folderNames, err := getFolderNames(repoRootPath)

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

	meta, err := loadMetadata(repoDirPath)

	if err != nil {
		return Repo{}, nil
	}

	links, err := loadLinks(repoDirPath)

	if err != nil {
		return Repo{}, nil
	}

	objects, err := loadObjects(repoDirPath)

	if err != nil {
		return Repo{}, nil
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

func loadMetadata(repoDirPath string) (Metadata, error) {

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

		objData, err := os.ReadFile(objFileLocation)

		if err != nil {
			return []ObjectInfo{}, err
		}

		var object ObjectInfo

		err = json.Unmarshal(objData, &object)

		if err != nil {
			return []ObjectInfo{}, err
		}

		objects = append(objects, object)

	}

	return objects, nil

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

		linkData, err := os.ReadFile(linkFileLocation)

		if err != nil {
			return []Link{}, err
		}

		var link Link

		err = json.Unmarshal(linkData, &link)

		if err != nil {
			return []Link{}, err
		}

		links = append(links, link)

	}

	return links, nil

}

func getFolderNames(dirPath string) ([]string, error) {

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
