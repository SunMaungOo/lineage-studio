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

func getLineageRequest(views []database.ObjectInfo, procedure []database.ObjectInfo) ls.LineageRequest {

	lineageObjects := getLineageObjectInfo(views, ls.ObjectType_OBJECT_TYPE_VIEW)

	lineageObjects = append(lineageObjects, getLineageObjectInfo(procedure, ls.ObjectType_OBJECT_TYPE_PROCEDURE)...)

	return ls.LineageRequest{
		Objects: lineageObjects,
	}
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
		fmt.Println("THERE IS ERROR")
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
	fmt.Println("Arguments : <database> <repo location> <repo name> <output location> <lineage server host> <lineage server port>")
	fmt.Println("<database> = user:password@host/mydatabase")
	fmt.Println("<repo location> = folder path to get the repo snapshot")
	fmt.Println("<repo name> =  name of the snapshot repo")
	fmt.Println("<output location> = folder path to save output")
	fmt.Println("<lineage server host> = host address of lineage server")
	fmt.Println("<lineage server port> = port of lineage server (0-65535)")

}

func main() {

	args := os.Args[1:]

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

	err = repo.SaveRepo(outputLocation, newRepo)

	if err != nil {
		log.Fatal("Error saving files")
	}
}
