package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/SunMaungOo/lineage-studio/internal/database"
	"github.com/SunMaungOo/lineage-studio/internal/initer"
	"github.com/SunMaungOo/lineage-studio/internal/repo"
)

type config struct {
	Port         string
	Host         string
	RepoLocation string
}

type initRequest struct {
	Uri  string `json:"uri"`
	Name string `json:"name"`
}

type initResponse struct {
	Status     string `json:"status"`
	DurationMs int64  `json:"durationMs"`
}

type errorResponse struct {
	Error      string `json:"error"`
	DurationMs int64  `json:"durationMs"`
}

func printHelp() {

	slog.Info("Usage instructions",
		"arguments", "<database> <repo location> <repo name>",
		"database_format_desc", "user:password@host/mydatabase",
		"repo_location_desc", "folder path to save the repo",
		"repo_name_desc", "name of the repo to save as",
	)

}

func healthEndpoint(message string) func(http.ResponseWriter, *http.Request) {

	return func(writer http.ResponseWriter, rawRequest *http.Request) {
		writer.Header().Set("Content-Type", "text/plain")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(message))
	}
}

func respondWithError(writer http.ResponseWriter, statusCode int, message string, startTime time.Time) {

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)

	json.NewEncoder(writer).Encode(errorResponse{
		Error:      message,
		DurationMs: time.Since(startTime).Milliseconds(),
	})
}

func initEndPoint(repoLocation string) func(http.ResponseWriter, *http.Request) {

	return func(writer http.ResponseWriter, rawRequest *http.Request) {

		startTime := time.Now()

		var request initRequest

		if err := json.NewDecoder(rawRequest.Body).Decode(&request); err != nil {

			slog.Error("Failed to decode request body", "error", err)

			respondWithError(writer, http.StatusBadRequest, "Invalid request body", startTime)

			return
		}

		hostInfo, credential, err := database.ParseDatabaseInfo(request.Uri)

		if err != nil {

			slog.Error("failed to parse database info", "error", err)

			respondWithError(writer, http.StatusBadRequest, "Database parsing failed", startTime)

			return
		}

		views, err := database.GetView(credential, hostInfo)

		if err != nil {

			slog.Error("failed to fetch views", "error", err)

			respondWithError(writer, http.StatusBadRequest, "Database view extraction failed", startTime)

			return
		}

		procedures, err := database.GetProcedure(credential, hostInfo)

		if err != nil {

			slog.Error("failed to fetch procedure", "error", err)

			respondWithError(writer, http.StatusBadRequest, "Database procedure extraction failed", startTime)

			return
		}

		repoName := request.Name

		initRepo := initer.GenerateRepo(repoName, repo.Metadata{
			Type: repo.TypeMssql,
			Host: hostInfo.Host,
			DB:   hostInfo.Database,
		}, views, procedures)

		err = repo.SaveRepo(repoLocation, initRepo, false)

		if err != nil {

			slog.Error("failed to save repository", "error", err)

			respondWithError(writer, http.StatusBadRequest, "Saving repository failed", startTime)

			return
		}

		durationMs := time.Since(startTime).Milliseconds()

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)

		json.NewEncoder(writer).Encode(initResponse{
			Status:     "success",
			DurationMs: durationMs,
		})
	}
}

func getDefaultEnvString(key string, fallback string) string {

	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func loadConfig() config {

	return config{
		Port:         getDefaultEnvString("PORT", "8080"),
		Host:         getDefaultEnvString("HOST", "0.0.0.0"),
		RepoLocation: getDefaultEnvString("DATA_LOCATION", "/data"),
	}
}

func webCommand() {

	cfg := loadConfig()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthEndpoint("ok"))
	mux.HandleFunc("GET /readyz", healthEndpoint("ready"))
	mux.HandleFunc("POST /init", initEndPoint(cfg.RepoLocation))

	addr := cfg.Host + ":" + cfg.Port

	slog.Info("Starting HTTP server", "addr", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {

		slog.Error("Server crashed", "error", err)

		os.Exit(1)

	}

}

func cmdCommand(args []string) {

	if len(args) != 3 {

		slog.Error("invalid arguments", slog.Int("expected", 3), slog.Int("actual", len(args)))

		printHelp()

		os.Exit(1)
	}

	hostInfo, credential, err := database.ParseDatabaseInfo(args[0])

	if err != nil {

		slog.Error("failed to parse database info", "error", err)

		os.Exit(1)
	}

	views, err := database.GetView(credential, hostInfo)

	if err != nil {

		slog.Error("failed to fetch views", "error", err)

		os.Exit(1)
	}

	procedures, err := database.GetProcedure(credential, hostInfo)

	if err != nil {

		slog.Error("failed to fetch procedure", "error", err)

		os.Exit(1)
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

		slog.Error("failed to save repository", "error", err)

		os.Exit(1)
	}

	slog.Info("repository saved successfully", "location", repoLocation, "name", repoName)

}

func main() {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	args := os.Args[1:]

	if len(args) == 1 && args[0] == "web" {

		webCommand()

	} else {

		cmdCommand(args)
	}

}
