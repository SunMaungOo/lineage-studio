package initer

import (
	"crypto/md5"
	"encoding/hex"
	"time"

	"github.com/SunMaungOo/lineage-studio/internal/database"
	"github.com/SunMaungOo/lineage-studio/internal/repo"
)

func GenerateRepo(repoName string, metadata repo.Metadata, views []database.ObjectInfo, procedures []database.ObjectInfo) repo.Repo {

	viewObjects := getRepoObjects(views)

	procedureObjects := getRepoObjects(procedures)

	objects := append(viewObjects, procedureObjects...)

	viewLinks := getLink(viewObjects, repo.LinkType("view"))

	procedureLinks := getLink(procedureObjects, repo.LinkType("procedure"))

	links := append(viewLinks, procedureLinks...)

	return repo.Repo{
		Name: repoName,
		Detail: repo.RepoDetail{
			Meta:    metadata,
			Links:   links,
			Objects: objects,
		},
	}
}

func getLink(repoObjects []repo.ObjectInfo, linkType repo.LinkType) []repo.Link {

	links := make([]repo.Link, len(repoObjects))

	for index, object := range repoObjects {

		links[index] = repo.Link{
			Name:    object.Name,
			Type:    linkType,
			Current: object.Hash,
			History: []repo.ObjectHistory{},
		}
	}

	return links
}

func getRepoObjects(sqlObjects []database.ObjectInfo) []repo.ObjectInfo {

	repoObjects := make([]repo.ObjectInfo, len(sqlObjects))

	for index, object := range sqlObjects {

		objName := object.Schema + "." + object.Name

		detailHash := getMd5Hash(object.Definition)

		hash := detailHash

		repoObjects[index] = repo.ObjectInfo{
			Name:        objName,
			Detail:      object.Definition,
			DetailHash:  hash,
			Lineage:     "",
			LineageHash: "",
			CreatedAt:   time.Now().UTC(),
			Verfied:     false,
			Hash:        hash,
		}

	}

	return repoObjects

}

func getMd5Hash(value string) string {

	byteData := md5.Sum([]byte(value))

	return hex.EncodeToString(byteData[:])
}
