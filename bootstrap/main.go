// Package bootstrap is used to make getting started with atlantis easier
package bootstrap

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/google/go-github/github"
	"github.com/mitchellh/colorstring"
	"github.com/pkg/errors"

	"os/exec"
)

var terraformExampleRepoOwner = "airauth"
var terraformExampleRepo = "example"

var bootstrapDescription = `[white]Welcome to Atlantis bootstrap!

This mode walks you through setting up and using Atlantis. We will
- fork an example terraform project to your username
- install terraform (if not already in your PATH)
- install ngrok so we can expose Atlantis to GitHub
- start Atlantis

[underline]Press Ctrl-c at any time to exit
`
var pullRequestBody = "In this pull request we will learn how to use atlantis. There are various commands that are available for you:\n" +
	"* Start by typing `atlantis plan` in the comments. That will run a `terraform plan`.\n" +
	"* Next, lets apply that plan. Type `atlantis apply` in the comments. This will run a `terraform apply`.\n" +
	"* For other atlantis commands type `atlantis help` in the comments.\n" +
	"\nThank you for using atlantis. For more info you can go to: https://atlantis.run"

func Start() error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	colorstring.Println(bootstrapDescription)
	colorstring.Print("\n[white][bold]GitHub username: ")
	fmt.Scanln(&githubUsername)
	if githubUsername == "" {
		return fmt.Errorf("please enter a valid github username")
	}
	colorstring.Println(`
[white]To continue, we need you to create a GitHub personal access token
with [green]"repo" [white]scope so we can fork an example terraform project.

Follow these instructions to create a token (we don't store any tokens):
[green]https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/#creating-a-token
[white]- use "atlantis" for the token description
- add "repo" scope
- copy the access token
`)
	// read github password, check for error later
	colorstring.Print("[white][bold]GitHub access token (will be hidden): ")
	githubPassword, _ = readPassword()

	// create github client
	tp := github.BasicAuthTransport{
		Username: strings.TrimSpace(githubUsername),
		Password: strings.TrimSpace(githubPassword),
	}

	githubClient := &Client{client: github.NewClient(tp.Client()), ctx: context.Background()}

	// fork terraform example repo
	colorstring.Printf("\n[white]=> forking repo ")
	s.Start()
	if err := githubClient.CreateFork(terraformExampleRepoOwner, terraformExampleRepo); err != nil {
		return errors.Wrapf(err, "forking repo %s/%s", terraformExampleRepoOwner, terraformExampleRepo)
	}
	if !githubClient.CheckForkSuccess(terraformExampleRepoOwner, terraformExampleRepo) {
		return fmt.Errorf("didn't find forked repo %s/%s. fork unsuccessful", terraformExampleRepoOwner, terraformExampleRepoOwner)
	}
	s.Stop()
	colorstring.Println("\n[green]=> fork completed!")

	// detect terraform
	_, err := exec.LookPath("terraform")
	if err != nil {
		// download terraform
		colorstring.Println("[yellow]=> terraform not found in $PATH.")
		colorstring.Printf("[white]=> downloading terraform ")
		s.Start()
		terraformDownloadURL := fmt.Sprintf("%s/terraform/%s/terraform_%s_%s_%s.zip", hashicorpReleasesURL, terraformVersion, terraformVersion, runtime.GOOS, runtime.GOARCH)
		if err := downloadAndUnzip(terraformDownloadURL, "/tmp/terraform.zip", "/tmp"); err != nil {
			return errors.Wrapf(err, "downloading and unzipping terraform")
		}
		colorstring.Println("\n[green]=> downloaded terraform successfully!")
		s.Stop()

		// ask user if we can move the terraform binary
		colorstring.Printf("[white][bold]atlantis needs terraform to run. can we install terraform? press any key to continue or Ctrl + C to exit ")
		fmt.Scanln()
		terraformCmd, err := executeCmd("mv", []string{"/tmp/terraform", "/usr/local/bin/"})
		if err != nil {
			return errors.Wrapf(err, "moving terraform binary into /usr/local/bin")
		}
		terraformCmd.Wait()
		colorstring.Println("[green]=> installed terraform successfully at /usr/local/bin")
	} else {
		colorstring.Println("[green]=> terraform found in $PATH!")
	}

	// download ngrok
	colorstring.Printf("[white]=> downloading ngrok  ")
	s.Start()
	ngrokURL := fmt.Sprintf("%s/ngrok-stable-%s-%s.zip", ngrokDownloadURL, runtime.GOOS, runtime.GOARCH)
	if err := downloadAndUnzip(ngrokURL, "/tmp/ngrok.zip", "/tmp"); err != nil {
		return errors.Wrapf(err, "downloading and unzipping ngrok")
	}
	s.Stop()
	colorstring.Println("\n[green]=> downloaded ngrok successfully!")

	// create ngrok tunnel
	colorstring.Printf("[white]=> creating secure tunnel ")
	s.Start()
	ngrokCmd, err := executeCmd("/tmp/ngrok", []string{"http", "4141"})
	if err != nil {
		return errors.Wrapf(err, "creating ngrok tunnel")
	}
	defer ngrokCmd.Process.Release()
	// wait for the tunnel to be up
	time.Sleep(2 * time.Second)
	s.Stop()
	colorstring.Println("\n[green]=> started tunnel!")

	// start atlantis server
	colorstring.Printf("[white]=> starting atlantis server ")
	s.Start()
	atlantisCmd, err := executeCmd("./atlantis", []string{"server", "--gh-user", githubUsername, "--gh-password", githubPassword, "--data-dir", "/tmp/atlantis/data"})
	if err != nil {
		return errors.Wrapf(err, "creating atlantis server")
	}
	defer atlantisCmd.Process.Release()
	tunnelURL, err := getTunnelAddr()
	if err != nil {
		return errors.Wrapf(err, "getting tunnel url")
	}
	s.Stop()
	colorstring.Printf("\n[green]=> atlantis server is now securely exposed at [bold][underline]%s", tunnelURL)
	fmt.Println("")

	// create atlantis webhook
	err = githubClient.CreateWebhook(githubUsername, terraformExampleRepo, fmt.Sprintf("%s/events", tunnelURL))
	if err != nil {
		return errors.Wrapf(err, "creating atlantis webhook")
	}

	// create a new pr in the example repo
	colorstring.Printf("[white]=> creating a new pull request ")
	s.Start()
	pullRequestURL, err := githubClient.CreatePullRequest(githubUsername, "example", "example", "master")
	if err != nil {
		return errors.Wrapf(err, "creating pull new pull request for repo %s/%s", githubUsername, terraformExampleRepo)
	}
	s.Stop()
	colorstring.Println("\n[green]=> pull request created!")

	// open new pull request in the browser
	colorstring.Println("[white]=> opening pull request....")
	_, err = executeCmd("open", []string{pullRequestURL})
	if err != nil {
		colorstring.Printf("[red]=> opening pull request failed. please go to: %s on the browser", pullRequestURL)
	}

	// wait for ngrok and atlantis server process to finish
	colorstring.Printf("\n[_green_][light_green]atlantis is running ")
	s.Start()
	colorstring.Println("[green] [press Ctrl + C to exit]")
	atlantisCmd.Wait()
	ngrokCmd.Wait()

	return nil
}
