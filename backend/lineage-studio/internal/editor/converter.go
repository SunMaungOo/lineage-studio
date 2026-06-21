package editor

import (
	lineage_studio "github.com/SunMaungOo/lineage-studio/internal/lineage-studio"
	"github.com/SunMaungOo/lineage-studio/internal/repo"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func MetadataTypeToDatabaseType(metadataType repo.MetadataType) lineage_studio.DatabaseType {

	switch metadataType {
	case repo.TypeMssql:
		return lineage_studio.DatabaseType_DATABASE_TYPE_UNSPECIFIED

	default:
		return lineage_studio.DatabaseType_DATABASE_TYPE_UNSPECIFIED
	}
}

func LinkTypeToObjectType(linkType repo.LinkType) lineage_studio.ObjectType {

	switch linkType {
	case repo.TypeView:
		return lineage_studio.ObjectType_OBJECT_TYPE_VIEW

	case repo.TypeProcedure:
		return lineage_studio.ObjectType_OBJECT_TYPE_PROCEDURE

	default:
		return lineage_studio.ObjectType_OBJECT_TYPE_UNSPECIFIED
	}
}

func ObjectDetailToLsObjectDetail(objectDetail repo.ObjectDetail, repoVersion string) lineage_studio.ObjectDetail {

	var verifiedAt *timestamppb.Timestamp = nil

	if objectDetail.Object.VerifiedAt != nil {
		verifiedAt = timestamppb.New(*objectDetail.Object.VerifiedAt)
	}

	return lineage_studio.ObjectDetail{
		Name:        objectDetail.Object.Name,
		ObjectType:  LinkTypeToObjectType(objectDetail.Type),
		SourceCode:  objectDetail.Object.Detail,
		Lineage:     objectDetail.Object.Lineage,
		LineageHash: objectDetail.Object.LineageHash,
		ObjectHash:  objectDetail.Object.Hash,
		Verified:    objectDetail.Object.Verified,
		VerifiedAt:  verifiedAt,
		CreatedAt:   timestamppb.New(objectDetail.Object.CreatedAt),
		RepoVersion: repoVersion,
	}
}
