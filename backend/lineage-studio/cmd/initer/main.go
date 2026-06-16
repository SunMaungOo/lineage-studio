package main

import (
	"fmt"
	"log"
	"os"

	"github.com/SunMaungOo/lineage-studio/internal/database"
	"github.com/SunMaungOo/lineage-studio/internal/initer"
	"github.com/SunMaungOo/lineage-studio/internal/repo"
)

func printHelp() {

	fmt.Println("Arguments : <database> <repo location> <repo name>")
	fmt.Println("<database> = user:password@host/mydatabase")
	fmt.Println("<repo location> = folder path to save the repo")
	fmt.Println("<repo name> =  name of the repo to save as")

}

func main() {

	args := os.Args[1:]

	if len(args) != 3 {
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

	initRepo := initer.GenerateRepo(repoName, repo.Metadata{
		Type: repo.TypeMssql,
		Host: hostInfo.Host,
		DB:   hostInfo.Database,
	}, views, procedures)

	err = repo.SaveRepo(repoLocation, initRepo, false)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Saved repo in %v/%v", repoLocation, repoName)
}
