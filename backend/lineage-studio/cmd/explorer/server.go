package main

import (
	"context"
	"path/filepath"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/SunMaungOo/lineage-studio/internal/git"
	lineage_explorer "github.com/SunMaungOo/lineage-studio/internal/lineage-explorer"
	"github.com/SunMaungOo/lineage-studio/internal/repo"
)

const STATEMENT_SEPARATOR string = ";"

const SOURCE_TARGET_SEPARATOR string = "->"

const VALUE_SEPARATOR string = ","

type LineageExplorerServer struct {
	RepoLocation string
	RemoteName   string
}

func (server LineageExplorerServer) getRepoSummary() ([]*lineage_explorer.RepoSummary, error) {

	repos := []*lineage_explorer.RepoSummary{}

	folderNames, err := repo.GetFolderNames(server.RepoLocation)

	if err != nil {
		return nil, err
	}

	for _, folderName := range folderNames {

		repoDirPath := filepath.Join(server.RepoLocation, folderName)

		metadata, err := repo.LoadMetadata(repoDirPath)

		if err != nil {
			return nil, err
		}

		repos = append(repos, &lineage_explorer.RepoSummary{
			Name:         folderName,
			DatabaseType: string(metadata.Type),
			Host:         metadata.Host,
			DbName:       metadata.DB,
		})

	}

	return repos, nil
}

func (server LineageExplorerServer) GetRepos(context context.Context, request *connect.Request[lineage_explorer.GetReposRequest]) (*connect.Response[lineage_explorer.GetReposResponse], error) {

	summaries, err := server.getRepoSummary()

	if err != nil {
		return nil, err
	}

	response := lineage_explorer.GetReposResponse{
		Repos: summaries,
	}

	return connect.NewResponse(&response), nil
}

func (server LineageExplorerServer) getLineageDetail(initRepo *repo.Repo) []*lineage_explorer.LineageDetail {

	lineageDetail := []*lineage_explorer.LineageDetail{}

	for _, objectDetail := range repo.GetCurrentObjectInfos(initRepo) {

		if len(strings.TrimSpace(objectDetail.Object.LineageHash)) == 0 {
			continue
		}

		lineage := objectDetail.Object.Lineage

		statements := strings.Split(lineage, STATEMENT_SEPARATOR)

		lineageStatements := make([]*lineage_explorer.Statement, len(statements))

		for index, statement := range statements {

			cleanStatement := strings.TrimSpace(statement)

			blocks := strings.Split(cleanStatement, SOURCE_TARGET_SEPARATOR)

			sources := strings.Split(blocks[0], VALUE_SEPARATOR)

			slices.Sort(sources)

			uniqueSources := slices.Compact(sources)

			var target string = ""

			if len(blocks) > 1 {
				target = blocks[1]
			}

			lineageStatements[index] = &lineage_explorer.Statement{
				Sources: uniqueSources,
				Target:  target,
			}
		}

		lineageDetail = append(lineageDetail, &lineage_explorer.LineageDetail{
			Repo:       initRepo.Name,
			Name:       objectDetail.Object.Name,
			Type:       string(objectDetail.Type),
			Statements: lineageStatements,
		})

	}

	return lineageDetail
}

func (server LineageExplorerServer) GetLineage(context context.Context, request *connect.Request[lineage_explorer.LineageRequest]) (*connect.Response[lineage_explorer.LineageResponse], error) {

	repoLocation := server.RepoLocation

	repoName := request.Msg.RepoName

	remoteName := server.RemoteName

	requestVersion := request.Msg.RepoVersion

	responseVersion := requestVersion

	repoDirPath := filepath.Join(repoLocation, repoName)

	currentVersion, err := git.GitHeadCommit(repoDirPath)

	if err != nil {
		return nil, err
	}

	remoteVersion, err := git.GitRemoteHeadCommit(repoDirPath, remoteName)

	if err != nil {
		return nil, err
	}

	if currentVersion != remoteVersion {

		err = git.GitPull(repoLocation)

		if err != nil {
			return nil, err
		}
	}

	responseVersion, err = git.GitHeadCommit(repoDirPath)

	if err != nil {
		return nil, err
	}

	if requestVersion == responseVersion {

		return connect.NewResponse(&lineage_explorer.LineageResponse{
			RepoVersion: responseVersion,
			Status:      lineage_explorer.LineageStatus_LINEAGE_STATUS_NO_CHANGES,
		}), nil

	}

	initRepo, err := repo.LoadRepo(repoLocation, repoName)

	if err != nil {
		return nil, err
	}

	lineageDetail := server.getLineageDetail(&initRepo)

	response := lineage_explorer.LineageResponse{
		RepoVersion: responseVersion,
		Details:     lineageDetail,
		Status:      lineage_explorer.LineageStatus_LINEAGE_STATUS_OK,
	}

	return connect.NewResponse(&response), nil
}
