package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func gitRun(gitFolder string, args ...string) (string, error) {

	var out bytes.Buffer

	var errBuf bytes.Buffer

	cmd := exec.Command("git", args...)
	cmd.Dir = gitFolder
	cmd.Stdout = &out
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %v: %v\n%s", strings.Join(args, " "), err, errBuf.String())
	}
	return strings.TrimSpace(out.String()), nil
}

func GitHeadCommit(gitFolder string) (string, error) {

	hash, err := gitRun(gitFolder, "rev-parse", "HEAD")

	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %v", err)
	}

	return hash, nil
}

func GitRemoteHeadCommit(gitFolder string, remote string) (string, error) {
	hash, err := gitRun(gitFolder, "ls-remote", remote, "HEAD")

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(strings.ReplaceAll(hash, "HEAD", "")), nil
}

func GitHeadCommitDateTime(gitFolder string) (time.Time, error) {

	output, err := gitRun(gitFolder, "log", "-1", "--format=%cI")

	if err != nil {
		return time.Time{}, fmt.Errorf("git log failed:%v", err)
	}

	localTime, err := time.Parse(time.RFC3339, output)

	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse git date string:%v", err)
	}

	return localTime.UTC(), nil

}

func GitPull(gitFolder string) error {

	_, err := gitRun(gitFolder, "pull")

	if err != nil {
		return err
	}

	return nil
}

// perform add . , commit and push
func GitAddAllCommitAndPush(gitFolder string, commitMessage string, remote string) error {

	if _, err := gitRun(gitFolder, "add", "-A"); err != nil {
		return fmt.Errorf("git add -A:%v", err)
	}

	if _, err := gitRun(gitFolder, "commit", "-m", commitMessage); err != nil {
		return fmt.Errorf("git commit: %v", err)
	}

	if _, err := gitRun(gitFolder, "push", "-u", remote, "master"); err != nil {
		return fmt.Errorf("git push: %v", err)
	}

	return nil

}

func GitResetHard(gitFolder string, targetCommit string) error {

	if _, err := gitRun(gitFolder, "reset", "--hard", targetCommit); err != nil {
		return fmt.Errorf("git reset --hard %s: %v", targetCommit, err)
	}

	return nil
}
