package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/SunMaungOo/lineage-studio/internal/editor"
	"github.com/SunMaungOo/lineage-studio/internal/git"
	lineage_studio "github.com/SunMaungOo/lineage-studio/internal/lineage-studio"
	"github.com/SunMaungOo/lineage-studio/internal/repo"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const maxPushRetries = 3

type repoLock struct {
	mutex sync.Mutex
	locks map[string]*sync.Mutex
}

// get the mutex , creating it if it not exist and lock it
func (repoLock *repoLock) Accquire(repoName string) *sync.Mutex {

	repoLock.mutex.Lock()

	lock, isExist := repoLock.locks[repoName]

	if !isExist {
		lock = &sync.Mutex{}
		repoLock.locks[repoName] = lock
	}

	repoLock.mutex.Unlock()

	lock.Lock()

	return lock
}

func newRepoLocks() *repoLock {
	return &repoLock{
		locks: make(map[string]*sync.Mutex),
	}
}

type LineageStudioServer struct {
	RepoLocation string
	RemoteName   string
	WriteLocks   *repoLock
}

func refreshIfStale(repoDirPath string, remoteName string) error {

	currentVersion, err := git.GitHeadCommit(repoDirPath)

	if err != nil {
		return err
	}

	remoteVersion, err := git.GitRemoteHeadCommit(repoDirPath, remoteName)

	if err != nil {
		return err
	}

	if currentVersion == remoteVersion {
		return nil
	}

	return git.GitPull(repoDirPath)
}

func (server LineageStudioServer) getRepoSummary() ([]*lineage_studio.RepoSummary, error) {

	repos := []*lineage_studio.RepoSummary{}

	folderNames, err := repo.GetFolderNames(server.RepoLocation)

	if err != nil {
		return nil, err
	}

	for _, folderName := range folderNames {

		repoDirPath := filepath.Join(server.RepoLocation, folderName)

		if err := refreshIfStale(repoDirPath, server.RemoteName); err != nil {
			return nil, err
		}

		metadata, err := repo.LoadMetadata(repoDirPath)

		if err != nil {
			return nil, err
		}

		lastSyncedAt, err := git.GitHeadCommitDateTime(repoDirPath)

		if err != nil {
			return nil, err
		}

		repos = append(repos, &lineage_studio.RepoSummary{
			Name:         folderName,
			DatabaseType: editor.MetadataTypeToDatabaseType(metadata.Type),
			Host:         metadata.Host,
			DbName:       metadata.DB,
			LastSyncedAt: timestamppb.New(lastSyncedAt),
		})

	}

	return repos, nil
}

func (LineageStudioServer) getObjectSummaries(initRepo *repo.Repo) []*lineage_studio.ObjectSummary {

	objectDetails := repo.GetCurrentObjectInfos(initRepo)

	objectSummaries := make([]*lineage_studio.ObjectSummary, len(objectDetails))

	for index, objectDetail := range objectDetails {

		objectSummaries[index] = &lineage_studio.ObjectSummary{
			Name:       objectDetail.Object.Name,
			ObjectType: editor.LinkTypeToObjectType(objectDetail.Type),
			Verified:   objectDetail.Object.Verified,
		}

	}

	return objectSummaries

}

func (server LineageStudioServer) GetRepos(context.Context, *connect.Request[lineage_studio.GetReposRequest]) (*connect.Response[lineage_studio.GetReposResponse], error) {

	repos, err := server.getRepoSummary()

	if err != nil {
		return nil, err
	}

	response := lineage_studio.GetReposResponse{
		Repos: repos,
	}

	return connect.NewResponse(&response), err
}

// Get all the object in the repo
func (server LineageStudioServer) GetObjects(context context.Context, request *connect.Request[lineage_studio.GetObjectsRequest]) (*connect.Response[lineage_studio.GetObjectsResponse], error) {

	repoLocation := server.RepoLocation

	repoName := request.Msg.RepoName

	repoDirPath := filepath.Join(repoLocation, repoName)

	if err := refreshIfStale(repoDirPath, server.RemoteName); err != nil {
		return nil, err
	}

	initRepo, err := repo.LoadRepo(repoLocation, repoName)

	if err != nil {
		return nil, err
	}

	repoVersion, err := git.GitHeadCommit(repoDirPath)

	if err != nil {
		return nil, err
	}

	response := lineage_studio.GetObjectsResponse{
		RepoVersion: repoVersion,
		Objects:     server.getObjectSummaries(&initRepo),
	}

	return connect.NewResponse(&response), nil
}

// Get detail about single object in the repo
func (server LineageStudioServer) GetObject(context context.Context, request *connect.Request[lineage_studio.GetObjectRequest]) (*connect.Response[lineage_studio.GetObjectResponse], error) {

	repoLocation := server.RepoLocation

	repoName := request.Msg.RepoName

	objectName := request.Msg.ObjectName

	repoDirPath := filepath.Join(repoLocation, repoName)

	if err := refreshIfStale(repoDirPath, server.RemoteName); err != nil {
		return nil, err
	}

	var objectDetail repo.ObjectDetail

	var err error

	isHistorical := len(request.Msg.HistoryHash) > 0

	if isHistorical {

		objectDetail, err = repo.GetObjectInfo(repoLocation, repoName, objectName, request.Msg.HistoryHash)

	} else {

		objectDetail, err = repo.GetCurrentObjectInfo(repoLocation, repoName, objectName)
	}

	if err != nil {
		return nil, err
	}

	repoVersion, err := git.GitHeadCommit(repoDirPath)

	if err != nil {
		return nil, err
	}

	lsObjectDetail := editor.ObjectDetailToLsObjectDetail(objectDetail, repoVersion)

	response := lineage_studio.GetObjectResponse{
		Object:       &lsObjectDetail,
		IsHistorical: isHistorical,
	}

	return connect.NewResponse(&response), nil
}

// Get history timeline of single object in the repo
func (server LineageStudioServer) GetObjectHistory(context context.Context, request *connect.Request[lineage_studio.GetObjectHistoryRequest]) (*connect.Response[lineage_studio.GetObjectHistoryResponse], error) {

	repoDirPath := filepath.Join(server.RepoLocation, request.Msg.RepoName)

	if err := refreshIfStale(repoDirPath, server.RemoteName); err != nil {
		return nil, err
	}

	objectDetails, err := repo.GetObjectInfoByName(server.RepoLocation, request.Msg.RepoName, request.Msg.ObjectName)

	if err != nil {
		return nil, err
	}

	histories := []*lineage_studio.HistoryEntry{}

	for _, objectDetail := range objectDetails {

		histories = append(histories, &lineage_studio.HistoryEntry{
			ObjectHash: objectDetail.Object.Hash,
			Verified:   objectDetail.Object.Verified,
			CreatedAt:  timestamppb.New(objectDetail.Object.CreatedAt),
		})
	}

	response := lineage_studio.GetObjectHistoryResponse{
		Histories: histories,
	}

	return connect.NewResponse(&response), nil
}

func (server LineageStudioServer) RefreshRepo(context context.Context, request *connect.Request[lineage_studio.RefreshRepoRequest]) (*connect.Response[lineage_studio.RefreshRepoResponse], error) {

	repoDirPath := filepath.Join(server.RepoLocation, request.Msg.RepoName)

	repoVersion, err := git.GitHeadCommit(repoDirPath)

	if err != nil {
		return nil, err
	}

	err = git.GitPull(repoDirPath)

	if err != nil {
		return nil, err
	}

	newRepoVersion, err := git.GitHeadCommit(repoDirPath)

	hadChanges := repoVersion != newRepoVersion

	response := lineage_studio.RefreshRepoResponse{
		NewRepoVersion: newRepoVersion,
		HadChanges:     hadChanges,
	}

	return connect.NewResponse(&response), nil
}

func getConflictVerifyLineageResponse(repoLocation string, repoName string, currentRepoVersion string, objectName string) (*lineage_studio.VerifyObjectLineageResponse, error) {

	currentObj, err := repo.GetCurrentObjectInfo(repoLocation, repoName, objectName)

	if err != nil {
		return nil, err
	}

	lsObjectDetail := editor.ObjectDetailToLsObjectDetail(currentObj, currentRepoVersion)

	conflictDetail := lineage_studio.ConflictDetail{
		CurrentRepoVersion: currentRepoVersion,
		CurrentObject:      &lsObjectDetail,
	}

	return &lineage_studio.VerifyObjectLineageResponse{
		Status:   lineage_studio.WriteStatus_WRITE_STATUS_CONFLICT,
		Conflict: &conflictDetail,
	}, nil

}

func getConflictChangeLineageResponse(repoLocation string, repoName string, currentRepoVersion string, objectName string) (*lineage_studio.ChangeLineageResponse, error) {

	currentObj, err := repo.GetCurrentObjectInfo(repoLocation, repoName, objectName)

	if err != nil {
		return nil, err
	}

	lsObjectDetail := editor.ObjectDetailToLsObjectDetail(currentObj, currentRepoVersion)

	conflictDetail := lineage_studio.ConflictDetail{
		CurrentRepoVersion: currentRepoVersion,
		CurrentObject:      &lsObjectDetail,
	}

	return &lineage_studio.ChangeLineageResponse{
		Status:         lineage_studio.WriteStatus_WRITE_STATUS_CONFLICT,
		NewRepoVersion: currentRepoVersion,
		Conflict:       &conflictDetail,
	}, nil

}

func objectChangedSince(repoLocation string, repoName string, objectName string, fromVersion string, toVersion string) bool {

	if fromVersion == toVersion {
		return false
	}

	if fromVersion == "" {
		return true
	}

	before, err := repo.GetObjectInfo(repoLocation, repoName, objectName, fromVersion)

	if err != nil {
		return true
	}

	after, err := repo.GetCurrentObjectInfo(repoLocation, repoName, objectName)

	if err != nil {
		return true
	}

	return before.Object.Hash != after.Object.Hash
}

func saveCommitPushWithRetry(repoLocation string,
	repoName string,
	remoteName string,
	objectName string,
	commitMessage string,
	clientVersion string,
	mutate func(baseRepo *repo.Repo) (repo.Repo, error)) (repo.Repo, string, bool, error) {

	repoDirPath := filepath.Join(repoLocation, repoName)

	for attempt := 0; attempt < maxPushRetries; attempt++ {

		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
		}

		baseVersion, err := git.GitHeadCommit(repoDirPath)

		if err != nil {
			return repo.Repo{}, "", false, err
		}

		if attempt == 0 && objectChangedSince(repoLocation, repoName, objectName, clientVersion, baseVersion) {
			return repo.Repo{}, baseVersion, true, nil
		}

		baseRepo, err := repo.LoadRepo(repoLocation, repoName)

		if err != nil {
			return repo.Repo{}, "", false, err
		}

		baseDetail, err := baseRepo.GetCurrentObjectInfo(objectName)

		if err != nil {
			return repo.Repo{}, "", false, err
		}

		mutated, err := mutate(&baseRepo)

		if err != nil {
			return repo.Repo{}, "", false, err
		}

		if err := repo.SaveRepo(repoLocation, mutated, true); err != nil {
			_ = git.GitResetHard(repoDirPath, baseVersion)
			return repo.Repo{}, "", false, err
		}

		pushErr := git.GitAddAllCommitAndPush(repoDirPath, commitMessage, remoteName)

		if pushErr == nil {

			pushedVersion, err := git.GitHeadCommit(repoDirPath)

			return mutated, pushedVersion, false, err
		}

		if err := git.GitResetHard(repoDirPath, baseVersion); err != nil {
			return repo.Repo{}, "", false, fmt.Errorf("push failed (%v) and reset failed: %v", pushErr, err)
		}

		if err := git.GitPull(repoDirPath); err != nil {
			return repo.Repo{}, "", false, fmt.Errorf("push failed (%v) and pull failed: %v", pushErr, err)
		}

		pulledVersion, err := git.GitHeadCommit(repoDirPath)

		if err != nil {
			return repo.Repo{}, "", false, err
		}

		afterPullRepo, err := repo.LoadRepo(repoLocation, repoName)

		if err != nil {
			return repo.Repo{}, "", false, err
		}

		afterDetail, err := afterPullRepo.GetCurrentObjectInfo(objectName)

		if err != nil {
			return repo.Repo{}, "", false, err
		}

		if afterDetail.Object.Hash != baseDetail.Object.Hash {

			return repo.Repo{}, pulledVersion, true, nil
		}
	}

	return repo.Repo{}, "", false, fmt.Errorf("gave up pushing to %q after %d attempts due to repeated concurrent writes", remoteName, maxPushRetries)

}

func (server LineageStudioServer) VerifyObjectLineage(context context.Context, request *connect.Request[lineage_studio.VerifyObjectLineageRequest]) (*connect.Response[lineage_studio.VerifyObjectLineageResponse], error) {

	repoLocation := server.RepoLocation

	repoName := request.Msg.RepoName

	remoteName := server.RemoteName

	objectName := request.Msg.ObjectName

	lock := server.WriteLocks.Accquire(repoName)

	defer lock.Unlock()

	clientVersion := request.Msg.RepoVersion

	commitMessage := fmt.Sprintf("verify lineage on %v", objectName)

	_, newRepoVersion, conflict, err := saveCommitPushWithRetry(
		repoLocation,
		repoName,
		remoteName,
		objectName,
		commitMessage,
		clientVersion,
		func(baseRepo *repo.Repo) (repo.Repo, error) {
			return editor.VerifyLineage(baseRepo, objectName)
		},
	)

	if err != nil {
		return nil, err
	}

	if conflict {

		conflictResponse, err := getConflictVerifyLineageResponse(repoLocation, repoName, newRepoVersion, objectName)

		if err != nil {
			return nil, err
		}

		return connect.NewResponse(conflictResponse), nil
	}

	response := lineage_studio.VerifyObjectLineageResponse{
		Status:         lineage_studio.WriteStatus_WRITE_STATUS_OK,
		NewRepoVersion: newRepoVersion,
	}

	return connect.NewResponse(&response), nil
}

func (server LineageStudioServer) ChangeLineage(context context.Context, request *connect.Request[lineage_studio.ChangeLineageRequest]) (*connect.Response[lineage_studio.ChangeLineageResponse], error) {

	repoLocation := server.RepoLocation

	repoName := request.Msg.RepoName

	remoteName := server.RemoteName

	objectName := request.Msg.ObjectName

	lineage := request.Msg.Lineage

	clientVersion := request.Msg.RepoVersion

	lock := server.WriteLocks.Accquire(repoName)

	defer lock.Unlock()

	commitMessage := fmt.Sprintf("change lineage on %v", objectName)

	newRepo, newRepoVersion, conflict, err := saveCommitPushWithRetry(
		repoLocation,
		repoName,
		remoteName,
		objectName,
		commitMessage,
		clientVersion,
		func(baseRepo *repo.Repo) (repo.Repo, error) {
			return editor.ChangeLineage(baseRepo, objectName, lineage)
		},
	)

	if err != nil {
		return nil, err
	}

	if conflict {

		conflictResponse, err := getConflictChangeLineageResponse(repoLocation, repoName, newRepoVersion, objectName)

		if err != nil {
			return nil, err
		}

		return connect.NewResponse(conflictResponse), nil
	}

	updatedObjectDetail, err := newRepo.GetCurrentObjectInfo(objectName)

	if err != nil {
		return nil, err
	}

	lsObjectDetail := editor.ObjectDetailToLsObjectDetail(updatedObjectDetail, newRepoVersion)

	response := lineage_studio.ChangeLineageResponse{
		Status:         lineage_studio.WriteStatus_WRITE_STATUS_OK,
		NewRepoVersion: newRepoVersion,
		UpdatedObject:  &lsObjectDetail,
	}

	return connect.NewResponse(&response), nil
}
