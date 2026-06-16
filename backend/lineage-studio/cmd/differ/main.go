package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/SunMaungOo/lineage-studio/internal/database"
	"github.com/SunMaungOo/lineage-studio/internal/differ"
	ls "github.com/SunMaungOo/lineage-studio/internal/lineage-server"
	"github.com/SunMaungOo/lineage-studio/internal/repo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func getLineageObjectInfo(objects []database.ObjectInfo, objectType ls.ObjectType) []*ls.ObjectInfo {

	lineageObjectInfo := make([]*ls.ObjectInfo, len(objects))

	for index, object := range objects {

		lineageObjectInfo[index] = &ls.ObjectInfo{
			Name:       object.Schema + "." + object.Name,
			Sql:        object.Definition,
			ObjectType: objectType,
		}

	}

	return lineageObjectInfo
}

func getRepoLineageObjectInfo(objects []repo.ObjectDetail) []*ls.ObjectInfo {

	lineageObjectInfo := make([]*ls.ObjectInfo, len(objects))

	for index, object := range objects {

		linkType := object.Type

		objectType := ls.ObjectType_OBJECT_TYPE_VIEW

		if linkType == "procedure" {
			objectType = ls.ObjectType_OBJECT_TYPE_PROCEDURE
		}

		lineageObjectInfo[index] = &ls.ObjectInfo{
			Name:       object.Object.Name,
			Sql:        object.Object.Detail,
			ObjectType: objectType,
		}

	}

	return lineageObjectInfo
}

func getLineageRequest(views []database.ObjectInfo, procedure []database.ObjectInfo) ls.LineageRequest {

	lineageObjects := getLineageObjectInfo(views, ls.ObjectType_OBJECT_TYPE_VIEW)

	lineageObjects = append(lineageObjects, getLineageObjectInfo(procedure, ls.ObjectType_OBJECT_TYPE_PROCEDURE)...)

	return ls.LineageRequest{
		Objects: lineageObjects,
	}
}

func getRepoLineageRequest(initRepo *repo.Repo) ls.LineageRequest {

	objectDetails := repo.GetCurrentObjectInfos(initRepo)

	return ls.LineageRequest{
		Objects: getRepoLineageObjectInfo(objectDetails),
	}
}

func getRepoLineage(lineageServerHost string, lineageServerPort int, repo *repo.Repo) map[differ.ObjectName]string {

	lineageServerUrl := fmt.Sprintf("%v:%v", lineageServerHost, lineageServerPort)

	connection, err := grpc.NewClient(lineageServerUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("Cannot create client:%v", err)
	}

	defer connection.Close()

	client := ls.NewLineageServiceClient(connection)

	lineageRequest := getRepoLineageRequest(repo)

	lineageResponse, err := client.GetLineage(context.Background(), &lineageRequest)

	if err != nil {
		log.Fatal(err)
	}

	return parseLineageResponse(lineageResponse)
}

func getLineage(lineageServerHost string, lineageServerPort int, views []database.ObjectInfo, procedures []database.ObjectInfo) map[differ.ObjectName]string {

	lineageServerUrl := fmt.Sprintf("%v:%v", lineageServerHost, lineageServerPort)

	connection, err := grpc.NewClient(lineageServerUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("Cannot create client:%v", err)
	}

	defer connection.Close()

	client := ls.NewLineageServiceClient(connection)

	lineageRequest := getLineageRequest(views, procedures)

	lineageResponse, err := client.GetLineage(context.Background(), &lineageRequest)

	if err != nil {
		log.Fatal(err)
	}

	return parseLineageResponse(lineageResponse)
}

func parseLineage(lineages []*ls.Lineage) string {

	lineageMessage := ""

	for _, lineage := range lineages {

		sourceParts := strings.Join(lineage.Sources, ",")

		lineageMessage += fmt.Sprintf("%v->%v;", sourceParts, lineage.Target)
	}

	return lineageMessage
}

func parseLineageResponse(response *ls.LineageResponse) map[differ.ObjectName]string {

	if response == nil {
		fmt.Println("NULL POINTER IS FOUND")
	}

	lineageMap := map[differ.ObjectName]string{}

	//for each object
	for _, objectLineage := range response.Lineages {

		objNameBlocks := strings.Split(objectLineage.Name, ".")

		lineageMap[differ.ObjectName{
			Schema: objNameBlocks[0],
			Name:   strings.Join(objNameBlocks[1:], ""),
		}] = parseLineage(objectLineage.Lineages)
	}

	return lineageMap
}

func printHelp() {
	fmt.Println("Commands : lineage (apply lineage) or update (update with new changes)")
	fmt.Println("Lineage Arguments : <repo location> <repo name> <lineage server host> <lineage server port>")
	fmt.Println("<repo location> = folder path to get the repo snapshot")
	fmt.Println("<repo name> =  name of the snapshot repo to apply lineage to")
	fmt.Println("<lineage server host> = host address of lineage server")
	fmt.Println("<lineage server port> = port of lineage server (0-65535)")
	fmt.Println("Update Arguments : <database> <repo location> <repo name> <output location> <lineage server host> <lineage server port>")
	fmt.Println("<database> = user:password@host/mydatabase")
	fmt.Println("<repo location> = folder path to get the repo snapshot")
	fmt.Println("<repo name> =  name of the snapshot repo")
	fmt.Println("<output location> = folder path to save output")
	fmt.Println("<lineage server host> = host address of lineage server")
	fmt.Println("<lineage server port> = port of lineage server (0-65535)")

}

func lineageCommand(args []string) {

	if len(args) != 4 {

		printHelp()

		return
	}

	repoLocation := args[0]

	repoName := args[1]

	lineageServerHost := args[2]

	lineageServerPortArgument := args[3]

	lineageServerPort, err := strconv.Atoi(lineageServerPortArgument)

	if err != nil {
		log.Fatal(err)
	}

	if !(lineageServerPort >= 0 && lineageServerPort <= 65535) {
		log.Fatal(fmt.Errorf("Invalid port number:%v", lineageServerPort))
	}

	initRepo, err := repo.LoadRepo(repoLocation, repoName)

	if err != nil {
		log.Fatal(err)
	}

	lineages := getRepoLineage(lineageServerHost, lineageServerPort, &initRepo)

	newRepo := differ.ApplyLineage(&initRepo, lineages)

	err = repo.SaveRepo(repoLocation, newRepo, true)

	if err != nil {
		log.Fatal(err)
	}

}

func updateCommand(args []string) {

	if len(args) != 6 {

		printHelp()

		return
	}

	hostInfo, credential, err := database.ParseDatabaseInfo(args[0])

	if err != nil {
		log.Fatal(err)
	}

	views, err := database.GetView(credential, hostInfo)

	if err != nil {
		log.Fatal(err)
	}

	procedures, err := database.GetProcedure(credential, hostInfo)

	if err != nil {
		log.Fatal(err)
	}

	repoLocation := args[1]

	repoName := args[2]

	outputLocation := args[3]

	lineageServerHost := args[4]

	lineageServerPortArgument := args[5]

	lineageServerPort, err := strconv.Atoi(lineageServerPortArgument)

	if err != nil {
		log.Fatal(err)
	}

	if !(lineageServerPort >= 0 && lineageServerPort <= 65535) {
		log.Fatal(fmt.Errorf("Invalid port number:%v", lineageServerPort))
	}

	initRepo, err := repo.LoadRepo(repoLocation, repoName)

	if err != nil {
		log.Fatal(err)
	}

	isRepoDiff, changedObjects := differ.GetRepoDiff(&initRepo, append(views, procedures...))

	if !isRepoDiff {
		fmt.Printf("Exiting, No differences is found in repo %v", repoLocation+"/"+repoName)
		return
	}

	lineages := getLineage(lineageServerHost, lineageServerPort, views, procedures)

	newRepo := differ.ApplyRepoChanges(&initRepo, changedObjects, lineages)

	err = repo.SaveRepo(outputLocation, newRepo, true)

	if err != nil {
		log.Fatal(err)
	}

}

func main() {

	args := os.Args[1:]

	command := ""

	if len(args) >= 1 {

		command = args[0]

	} else {

		printHelp()

		return
	}

	if command != "lineage" && command != "update" {

		printHelp()

		return
	}

	switch command {

	case "lineage":

		lineageCommand(args[1:])

	case "update":

		updateCommand(args[1:])

	}

}
