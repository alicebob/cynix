package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/google/go-github/v52/github"
	"golang.org/x/oauth2"
)

var (
	flagOwner = flag.String("owner", "alicebob", "repository owner")
	flagRepo  = flag.String("repo", "cynix", "repository name")
	// flagToken = flag.String("token", "", "registration token")
	flagPAT       = flag.String("pat", "", "personal access token")
	flagName      = flag.String("name", "cynix", "runner name")
	flagRunnerDir = flag.String("dir", "./runner/", "runner dir (will be wiped)") // needs trailing /
)

// we always need all these three args, might as well bundle them
type repo struct {
	Client *github.Client
	Owner  string
	Repo   string
}

func (r repo) URL() string {
	return fmt.Sprintf("https://github.com/%s/%s/", r.Owner, r.Repo)
}

func main() {
	flag.Parse()
	ctx := context.Background()
	fmt.Printf("connecting to %s\n", *flagRepo)

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *flagPAT},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	conn := &repo{
		Client: client,
		Owner:  *flagOwner,
		Repo:   *flagRepo,
	}

	/*
		// list all repositories for the authenticated user
		repos, _, err := client.Repositories.List(ctx, *flagOwner, nil)
		if err != nil {
			fmt.Printf("list repos: %s\n", err)
		} else {
			fmt.Printf("%d repos:\n", len(repos))
			for _, r := range repos {
				fmt.Printf("repo: %#v\n", *r.Name)
			}
		}
	*/

	runs, _, err := client.Actions.ListRunners(ctx, *flagOwner, *flagRepo, nil)
	if err != nil {
		fmt.Printf("list runners: %s\n", err)
	} else {
		fmt.Printf("%d runners\n", len(runs.Runners))
		for _, r := range runs.Runners {
			fmt.Printf("runner: %#v\n", *r.Name)
		}
	}

	exe, err := installRunner(ctx, conn)
	if err != nil {
		log.Fatalf("install runner: %s", err)
	}
	fmt.Printf("runner: %s\n", exe)

	tok, _, err := client.Actions.CreateRegistrationToken(ctx, *flagOwner, *flagRepo)
	if err != nil {
		log.Fatalf("create token: %s", err)
	}
	regToken := *tok.Token
	fmt.Printf("token: %s\n", regToken)

	if err := configRunner(exe, conn.URL(), regToken, *flagName); err != nil {
		log.Fatalf("config: %s\n", err)
	}
	fmt.Printf("runner configured\n")
}

func installRunner(ctx context.Context, conn *repo) (string, error) {
	dls, _, err := conn.Client.Actions.ListRunnerApplicationDownloads(ctx, conn.Owner, conn.Repo)
	if err != nil {
		return "", err
	}
	for _, dl := range dls {
		if *dl.OS == "linux" && *dl.Architecture == "x64" {
			fmt.Printf("dl: %#v\n", *dl.Filename)
			return unpackRunner(*dl.DownloadURL, *flagRunnerDir)
		}
	}
	return "", errors.New("no suitable runner found.")
}

// Download and untar the action runner.
// Pretty dodgy code.
func unpackRunner(downloadURL, dir string) (string, error) {
	if err := os.Mkdir(dir, 0700); err != nil {
		fmt.Printf("mkdir err: %s (ignoring)\n", err) // FIXME
	}

	fmt.Printf("downloading %s\n", downloadURL)
	resp, err := http.DefaultClient.Get(downloadURL)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code (%d)", resp.StatusCode)
	}
	fh, err := os.Create(dir + "actions.tar.gz")
	if err != nil {
		return "", err
	}
	defer fh.Close()
	if _, err := io.Copy(fh, resp.Body); err != nil {
		return "", err
	}

	cmd := exec.Command("tar", "zxf", "./actions.tar.gz")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return dir + "config.sh", nil
}

func configRunner(configExe, repoURL, token, name string) error {
	cmd := exec.Command(configExe,
		"--token", token,
		"--url", repoURL,
		"--name", name,
		"--unattended",
		"--disableupdate",
	)
	return cmd.Run()
}
