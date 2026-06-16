package differ

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/SunMaungOo/lineage-studio/internal/database"
	"github.com/SunMaungOo/lineage-studio/internal/repo"
)

type ObjectName struct {
	Schema string
	Name   string
}

// check if there is differences in the repo
// initRepo = old repo to check
// objects = new object to check with
// check definition but not lineage
func IsRepoDiff(initRepo *repo.Repo, objects []database.ObjectInfo) bool {

	if len(objects) == 0 {
		fmt.Println("AAAAA")
		return false
	}

	if len(initRepo.Detail.Links) == 0 {
		return true
	}

	// key = object name

	links := make(map[string]repo.Link)

	for _, link := range initRepo.Detail.Links {
		links[link.Name] = link
	}

	//check if there is link changes

	for _, newObject := range objects {

		newObjectName := newObject.Schema + "." + newObject.Name

		link, isExist := links[newObjectName]

		if !isExist {
			return true
		}

		repoCurrentObjHash := link.Current

		// if there is same object , check if their definition is same

		if isExist, objectInfo := getObject(initRepo.Detail.Objects, newObjectName, repoCurrentObjHash); isExist {

			if objectInfo.Detail != newObject.Definition {
				return true
			}

		}

	}
	return false
}

// (isRepoDiff,object info which have changes)
// check definition but not lineage
func GetRepoDiff(initRepo *repo.Repo, databaseObjects []database.ObjectInfo) (bool, []database.ObjectInfo) {

	diffObjects := []database.ObjectInfo{}

	isRepoDiff := IsRepoDiff(initRepo, databaseObjects)

	if !isRepoDiff {
		return false, []database.ObjectInfo{}
	}

	// key = object name

	links := make(map[string]repo.Link)

	for _, link := range initRepo.Detail.Links {
		links[link.Name] = link
	}

	for _, databaseObject := range databaseObjects {

		newObjectName := databaseObject.Schema + "." + databaseObject.Name

		link, isExist := links[newObjectName]

		if !isExist {

			diffObjects = append(diffObjects, databaseObject)

			continue
		}

		repoCurrentObjHash := link.Current

		// if there is same object , check if their definition is same

		if isExist, objectInfo := getObject(initRepo.Detail.Objects, newObjectName, repoCurrentObjHash); isExist {

			if objectInfo.Detail != databaseObject.Definition {
				diffObjects = append(diffObjects, databaseObject)
			}

		}

	}

	return isRepoDiff, diffObjects
}

// Apply a lineage to the repo
// lineage = map[objectName,lineageOfThatObject]. objectName in schema.name format
func ApplyLineage(initRepo *repo.Repo, lineage map[ObjectName]string) repo.Repo {

	changeRepo := repo.Repo{}

	jsonData, _ := json.Marshal(initRepo)

	json.Unmarshal(jsonData, &changeRepo)

	for index, obj := range changeRepo.Detail.Objects {

		objNameBlocks := strings.Split(obj.Name, ".")

		detailHash := obj.DetailHash

		hash := detailHash

		var lineageHash string

		lineageValue, lineageExist := lineage[ObjectName{Schema: objNameBlocks[0], Name: strings.Join(objNameBlocks[1:], "")}]

		if lineageExist {

			lineageHash = getMd5Hash(lineageValue)

			hash = getMd5Hash(detailHash + lineageHash)

			changeRepo.Detail.Objects[index].Lineage = lineageValue
			changeRepo.Detail.Objects[index].LineageHash = lineageHash

		}

		changeRepo.Detail.Objects[index].Hash = hash

	}

	return changeRepo

}

// changedObjects = both and modifed object
// lineage = map[objectName,lineageOfThatObject]. objectName in schema.name format
func ApplyRepoChanges(initRepo *repo.Repo, changedObjects []database.ObjectInfo, lineage map[ObjectName]string) repo.Repo {

	changeRepo := repo.Repo{}

	//perform deep copy

	jsonData, _ := json.Marshal(initRepo)

	json.Unmarshal(jsonData, &changeRepo)

	for _, obj := range changedObjects {

		objName := obj.Schema + "." + obj.Name

		detailHash := getMd5Hash(obj.Definition)

		hash := detailHash

		var lineageHash string

		lineageValue, lineageExist := lineage[ObjectName{Schema: obj.Schema, Name: obj.Name}]

		if lineageExist {

			lineageHash = getMd5Hash(lineageValue)

			hash = getMd5Hash(detailHash + lineageHash)

		}

		changeRepo.Detail.Objects = append(changeRepo.Detail.Objects, repo.ObjectInfo{
			Name:        objName,
			Detail:      obj.Definition,
			DetailHash:  detailHash,
			Lineage:     lineageValue,
			LineageHash: lineageHash,
			CreatedAt:   time.Now().UTC(),
			Verfied:     false,
			Hash:        hash,
		})

		changeRepo.Detail.Links = addObjectToLink(changeRepo.Detail.Links,
			objName,
			hash,
			obj.ObjectType)
	}

	return changeRepo
}

// return new link with changed data
func addObjectToLink(links []repo.Link, objectName string, objectHash string, objectType database.ObjectType) []repo.Link {

	newLinks := make([]repo.Link, len(links))

	copy(newLinks, links)

	isObjectExist := false

	for linkIndex, link := range newLinks {

		if link.Name == objectName {

			isObjectExist = true

			previousCurrent := link.Current

			newLinks[linkIndex].Current = objectHash

			oldHistory := link.History

			newHistory := make([]repo.ObjectHistory, len(oldHistory)+1)

			// sort the history again

			for index, objectHistory := range oldHistory {

				newHistory[index] = repo.ObjectHistory{
					Hash:  objectHistory.Hash,
					Order: objectHistory.Order + 1,
				}
			}

			newHistory[len(oldHistory)] = repo.ObjectHistory{
				Hash:  previousCurrent,
				Order: 1,
			}

			newLinks[linkIndex].History = newHistory

		}

	}

	// if the object does not already exist in links , we have to add themn

	if !isObjectExist {

		linkType := repo.LinkType("view")

		if objectType == "procedure" {
			linkType = repo.LinkType("procedure")
		}

		newLinks = append(newLinks, repo.Link{
			Name:    objectName,
			Type:    linkType,
			Current: objectHash,
			History: []repo.ObjectHistory{},
		})

	}

	return newLinks

}

// get object which have the same name and hash
// (isExist,found object)
func getObject(objects []repo.ObjectInfo, name string, hash string) (bool, repo.ObjectInfo) {

	for _, repoObject := range objects {

		if repoObject.Name == name && repoObject.Hash == hash {
			return true, repoObject
		}

	}

	return false, repo.ObjectInfo{}
}

func getMd5Hash(value string) string {

	byteData := md5.Sum([]byte(value))

	return hex.EncodeToString(byteData[:])
}
