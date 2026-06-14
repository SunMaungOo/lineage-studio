package main

import (
	"fmt"
	"log"
	"os"
	"strings"

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

type Credential struct {
	User     string
	Password string
}

type HostInfo struct {
	Host     string
	Database string
}

func parseHostInfo(value string) (HostInfo, error) {

	if !strings.Contains(value, "/") {
		return HostInfo{}, fmt.Errorf("%v does not contain /", value)
	}

	lastIndex := strings.LastIndex(value, "/")

	return HostInfo{
		Host:     value[0:lastIndex],
		Database: value[lastIndex+1:],
	}, nil
}

func parseCredential(value string) (Credential, error) {

	if !strings.Contains(value, ":") {
		return Credential{}, fmt.Errorf("%v does not contain :", value)
	}

	lastIndex := strings.LastIndex(value, ":")

	return Credential{
		User:     value[0:lastIndex],
		Password: value[lastIndex+1:],
	}, nil

}

func parseDatabase(value string) (HostInfo, Credential, error) {

	if !strings.Contains(value, "@") {
		return HostInfo{}, Credential{}, fmt.Errorf("%v does not contain @", value)
	}

	index := strings.Index(value, "@")

	credentialStr := value[0:index]

	hostInfoStr := value[index+1:]

	hostInfo, err := parseHostInfo(hostInfoStr)

	if err != nil {
		return HostInfo{}, Credential{}, err
	}

	credential, err := parseCredential(credentialStr)

	if err != nil {
		return HostInfo{}, Credential{}, err
	}

	return hostInfo, credential, nil

}

func main() {

	args := os.Args[1:]

	if len(args) != 3 {
		printHelp()

		return
	}

	hostInfo, credential, err := parseDatabase(args[0])

	if err != nil {
		log.Fatal(err)
	}

	views, err := database.GetView(credential.User, credential.Password, hostInfo.Host, hostInfo.Database)

	if err != nil {
		log.Fatal(err)
	}

	procedures, err := database.GetProcedure(credential.User, credential.Password, hostInfo.Host, hostInfo.Database)

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

	err = repo.SaveRepo(repoLocation, initRepo)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Saved repo in %v/%v", repoLocation, repoName)
}
