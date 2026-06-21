package editor

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/SunMaungOo/lineage-studio/internal/repo"
)

func VerifyLineage(initRepo *repo.Repo, objectName string) (repo.Repo, error) {

	changeRepo := repo.Repo{}

	jsonData, _ := json.Marshal(initRepo)

	json.Unmarshal(jsonData, &changeRepo)

	var currenctObjectHash string

	var isFoundObject bool = false

	for _, link := range changeRepo.Detail.Links {

		if link.Name == objectName {

			currenctObjectHash = link.Current

			isFoundObject = true

			break
		}
	}

	if !isFoundObject {
		return repo.Repo{}, errors.New("No object found")
	}

	for index, obj := range changeRepo.Detail.Objects {

		if !(obj.Name == objectName && obj.Hash == currenctObjectHash) {
			continue
		}

		verifiedAt := time.Now().UTC()

		changeRepo.Detail.Objects[index] = repo.ObjectInfo{
			Name:        obj.Name,
			Detail:      obj.Detail,
			DetailHash:  obj.DetailHash,
			Lineage:     obj.Lineage,
			LineageHash: obj.LineageHash,
			CreatedAt:   obj.CreatedAt,
			Verified:    true,
			VerifiedAt:  &verifiedAt,
			Hash:        obj.Hash,
		}

		return changeRepo, nil
	}

	return repo.Repo{}, nil

}

func ChangeLineage(initRepo *repo.Repo, objectName string, lineage string) (repo.Repo, error) {

	changeRepo := repo.Repo{}

	jsonData, _ := json.Marshal(initRepo)

	json.Unmarshal(jsonData, &changeRepo)

	var currenctObjectHash string

	var isFoundObject bool = false

	for _, link := range changeRepo.Detail.Links {

		if link.Name == objectName {

			currenctObjectHash = link.Current

			isFoundObject = true

			break
		}
	}

	if !isFoundObject {
		return repo.Repo{}, errors.New("No object found")
	}

	for _, obj := range initRepo.Detail.Objects {

		if !(obj.Name == objectName && obj.Hash == currenctObjectHash) {
			continue
		}

		lineageHash := getMd5Hash(lineage)

		hash := getMd5Hash(obj.Detail + lineageHash)

		verifiedAt := time.Now().UTC()

		changeRepo.Detail.Objects = append(changeRepo.Detail.Objects, repo.ObjectInfo{
			Name:        obj.Name,
			Detail:      obj.Detail,
			DetailHash:  obj.DetailHash,
			Lineage:     lineage,
			LineageHash: lineageHash,
			CreatedAt:   obj.CreatedAt,
			Verified:    true,
			VerifiedAt:  &verifiedAt,
			Hash:        hash,
		})

		changeRepo.Detail.Links = repo.AddObjectToLink(changeRepo.Detail.Links, obj.Name, hash, nil)

		return changeRepo, nil
	}

	return repo.Repo{}, nil

}

func getMd5Hash(value string) string {

	byteData := md5.Sum([]byte(value))

	return hex.EncodeToString(byteData[:])
}
