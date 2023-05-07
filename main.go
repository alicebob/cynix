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
	"strings"

	"github.com/google/go-github/v52/github"
	"golang.org/x/oauth2"
)

var (
	flagOwner     = flag.String("owner", "alicebob", "repository owner")
	flagRepo      = flag.String("repo", "cynix", "repository name")
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
	log.Printf("using repo %s/%s", *flagOwner, *flagRepo)
	resetRunner(*flagRunnerDir)

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

	runs, _, err := client.Actions.ListRunners(ctx, *flagOwner, *flagRepo, nil)
	if err != nil {
		log.Fatalf("list runners: %s", err)
	}
	log.Printf("%d runners", len(runs.Runners))
	for _, r := range runs.Runners {
		log.Printf(" - runner: %#v", *r.Name)
	}

	tok, _, err := client.Actions.CreateRegistrationToken(ctx, *flagOwner, *flagRepo)
	if err != nil {
		log.Fatalf("create token: %s", err)
	}
	regToken := *tok.Token
	log.Printf("got registration token")

	if err := installRunner(ctx, conn, *flagRunnerDir); err != nil {
		log.Fatalf("install runner: %s", err)
	}

	var (
		configExe = *flagRunnerDir + "config.sh"
		runExe    = *flagRunnerDir + "run.sh"
	)

	if err := configRunner(configExe, conn.URL(), regToken, *flagName); err != nil {
		log.Fatalf("config: %s", err)
	}
	log.Printf("runner configured")

	if err := runRunner(ctx, runExe); err != nil {
		log.Printf("runner: %s", err)
	}
}

// download + untar runner binary
func installRunner(ctx context.Context, conn *repo, dir string) error {
	dls, _, err := conn.Client.Actions.ListRunnerApplicationDownloads(ctx, conn.Owner, conn.Repo)
	if err != nil {
		return err
	}
	for _, dl := range dls {
		if *dl.OS == "linux" && *dl.Architecture == "x64" {
			log.Printf("using runner: %s", *dl.Filename)
			return unpackRunner(*dl.DownloadURL, dir)
		}
	}
	return errors.New("no suitable runner found.")
}

func resetRunner(dir string) {
	if err := os.Mkdir(dir, 0700); err != nil {
		fmt.Printf("mkdir err: %s (ignoring)\n", err) // FIXME
	}
	os.Remove(dir + ".runner")
}

// Download and untar the action runner.
// Pretty dodgy code.
func unpackRunner(downloadURL, dir string) error {
	resp, err := http.DefaultClient.Get(downloadURL)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code (%d)", resp.StatusCode)
	}
	fh, err := os.Create(dir + "actions.tar.gz")
	if err != nil {
		return err
	}
	defer fh.Close()
	if _, err := io.Copy(fh, resp.Body); err != nil {
		return err
	}

	cmd := exec.Command("tar", "zxf", "./actions.tar.gz")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func configRunner(exe, repoURL, token, name string) error {
	cmd := exec.Command(exe,
		"--token", token,
		"--url", repoURL,
		"--name", name,
		"--unattended",
		"--disableupdate",
	)
	cmd.Stdout = &pw{pre: "config stdout"}
	cmd.Stderr = &pw{pre: "config stderr"}
	return cmd.Run()
}

func runRunner(ctx context.Context, exe string) error {
	cmd := exec.CommandContext(ctx, exe)
	cmd.Stdout = &pw{pre: "runner stdout"}
	cmd.Stderr = &pw{pre: "runner stderr"}
	return cmd.Run()
}

// logs runner output
type pw struct {
	pre string
}

func (wr pw) Write(s []byte) (int, error) {
	for _, l := range strings.SplitAfter(string(s), "\n") {
		if len(l) > 0 {
			log.Printf("%s: %s", wr.pre, strings.TrimSuffix(string(l), "\n"))
		}
	}
	return len(s), nil
}
