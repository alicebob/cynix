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
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/google/go-github/v52/github"
	"golang.org/x/oauth2"
)

var (
	flagOwner     = flag.String("owner", "alicebob", "repository owner")
	flagRepo      = flag.String("repo", "cynix", "repository name")
	flagPAT       = flag.String("pat", "", "personal access token")
	flagName      = flag.String("name", "cynix", "runner name")                   // unique
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
	ctx, cancel := context.WithCancel(context.Background())
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

	// mostly to test the connection.
	// (but we could make our runner's name unique...)
	runs, _, err := conn.Client.Actions.ListRunners(ctx, *flagOwner, *flagRepo, nil)
	if err != nil {
		log.Fatalf("list runners: %s", err)
	}
	log.Printf("%d runners", len(runs.Runners))
	for _, r := range runs.Runners {
		log.Printf(" - runner: %#v", *r.Name)
	}

	if err := installRunner(ctx, conn, *flagRunnerDir); err != nil {
		log.Fatalf("install runner: %s", err)
	}

	if err := setupRunner(ctx, conn, *flagRunnerDir, *flagName); err != nil {
		log.Fatalf("setup runner: %s", err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := runRunner(ctx, *flagRunnerDir+"run.sh"); err != nil {
			log.Printf("runner: %s", err)
		}
		wg.Done()
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	<-c

	log.Printf("shutting down...")
	cancel()
	wg.Wait()

	if err := unregisterRunner(context.Background(), conn, *flagRunnerDir+"config.sh", *flagName); err != nil {
		log.Printf("unreg runner: %s -- cleanup manually", err)
	}
}

func setupRunner(ctx context.Context, conn *repo, dir string, runnerName string) error {
	res, _, err := conn.Client.Actions.CreateRegistrationToken(ctx, conn.Owner, conn.Repo)
	if err != nil {
		return fmt.Errorf("create token: %w", err)
	}
	token := *res.Token
	log.Printf("got registration token")

	exe := *flagRunnerDir + "config.sh"
	cmd := exec.Command(exe,
		"--token", token,
		"--url", conn.URL(),
		"--name", runnerName,
		"--unattended",
		"--disableupdate",
	)
	cmd.Stdout = &pw{pre: "config stdout"}
	cmd.Stderr = &pw{pre: "config stderr"}
	return cmd.Run()
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

func runRunner(ctx context.Context, exe string) error {
	cmd := exec.CommandContext(ctx, exe)
	cmd.Stdout = &pw{pre: "runner stdout"}
	cmd.Stderr = &pw{pre: "runner stderr"}
	return cmd.Run()
}

func unregisterRunner(ctx context.Context, conn *repo, configExe string, runnerName string) error {
	res, _, err := conn.Client.Actions.CreateRemoveToken(ctx, conn.Owner, conn.Repo)
	if err != nil {
		return fmt.Errorf("remove token: %w", err)
	}
	token := *res.Token
	log.Printf("got remove token")

	cmd := exec.CommandContext(ctx, configExe,
		"remove",
		"--token", token,
		"--url", conn.URL(),
		"--name", runnerName,
		"--unattended",
	)
	cmd.Stdout = &pw{pre: "remove stdout"}
	cmd.Stderr = &pw{pre: "remove stderr"}
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
