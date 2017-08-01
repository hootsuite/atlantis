# atlantis
[![CircleCI](https://circleci.com/gh/hootsuite/atlantis/tree/master.svg?style=shield&circle-token=08bf5b34233b0e168a9dd73e01cafdcf7dc4bf16)](https://circleci.com/gh/hootsuite/atlantis/tree/master)

A unified workflow for collaborating on Terraform through GitHub.

## Features
➜ Collaborate on Terraform with your team
- Run terraform `plan` and `apply` **from GitHub pull requests** so everyone can review the output
![atlantis plan](https://github.com/hootsuite/atlantis/raw/master/docs/atlantis-plan.gif)
- **Lock environments** until pull requests are merged to prevent concurrent modification and confusion

➜ Developers can write Terraform safely
- **No need to distribute AWS credentials** to your whole team. Developers can submit Terraform changes and run `plan` and `apply` directly from the pull request
- Optionally, require a **review and approval** prior to running `apply`

➜ Also
- No more **copy-pasted code across environments**. Atlantis supports using an `env/{env}.tfvars` file per environment so you can write your base configuration once
- Support **multiple versions of Terraform** with a simple project config file

* [atlantis](#atlantis)
  * [Features](#features)
  * [Getting Started](#getting-started)
    * [First Download Atlantis](#first-download-atlantis)
    * [Start with `atlantis bootstrap` (recommended)](#start-with-atlantis-bootstrap-recommended)
    * [Start Manually](#start-manually)
  * [Production-Ready Deployment](#production-ready-deployment)
  * [Configuration](#configuration)
    * [AWS Credentials](#aws-credentials)
      * [Assume Role](#assume-role)
  * [Environments](#environments)
  * [Locking](#locking)
  * [`atlantis.yaml` Config File](#atlantisyaml-config-file)
  * [Glossary](#glossary)
      * [Project](#project)
      * [Environment](#environment)

## Getting Started
Atlantis runs as a server that receives GitHub webhooks. Once it's running and hooked up with GitHub, you can interact with it directly through GitHub comments.

### First Download Atlantis
Download from https://github.com/hootsuite/atlantis/releases

### Start with `atlantis bootstrap` (recommended)
Run `atlantis bootstrap` to get started quickly with Atlantis.

If you want to manually run through all the steps that `bootstrap` performs, keep reading.

### Start Manually
To manually get started with Atlantis, you'll need to
- install `terraform` into your `$PATH`
	- download from https://www.terraform.io/downloads.html
	- `unzip path/to/terraform_*.zip -d /usr/local/bin`
	- check that it's installed by running `terraform version`
- Atlantis needs to be reachable on an IP address or hostname that github.com can access. By default, Atlantis runs on port `4141` (this can be changed with the `--port` flag). You can install `ngrok` to make exposing Atlantis easy for testing purposes
	- download from https://ngrok.com/download
	- `unzip path/to/ngrok*.zip -d /usr/local/bin`
	- start ngrok with `ngrok http 4141`
- Create a GitHub personal access token for Atlantis to use GitHub's API
	- follow [https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/#creating-a-token](https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/#creating-a-token)
	- copy the access token to your clipboard
- now you're ready to start Atlantis! Run:
```
$ atlantis server --atlantis-url $URL --gh-username $USERNAME --gh-token $TOKEN
2049/10/6 00:00:00 [WARN] server: Atlantis started - listening on port 4141
```

- where `$URL` is the URL that Atlantis can be reached at. If using `ngrok` it will be something like `https://68da2fdd.ngrok.io`
- where `$USERNAME` is your GitHub username
- where `$TOKEN` is the access token you created

Now that Atlantis is running, it's time to test it out. You'll need to set up a pull request first

- Fork https://github.com/hootsuite/atlantis-example to your user
- Add Atlantis as a webhook to the forked repo
	- navigate to `{your-repo-url}/settings/hooks/new`, ex. https://github.com/hootsuite/atlantis-example/settings/hooks/new
	- set **Payload URL** to `$URL/events` where `$URL` is what you used above, ex. `https://68da2fdd.ngrok.io/events`. **Be sure to add `/events` to the end**
	- set **Content type** to `application/json`
	- leave **Secret** blank
	- select **Let me select individual events**
	- check the boxes
		- **Pull request review**
		- **Push**
		- **Issue comment**
		- **Pull request**
	- leave **Active** checked
	- click **Add webhook**
- Now that Atlantis can receive events you should be able to comment on a pull request to trigger Atlantis. Let's create a pull request
	- Navigate to `{your-repo-url}/branches`, ex. https://github.com/hootsuite/atlantis-example/branches
	- click the **New pull request** button next to the `example` branch
	- Change the `base` to `{your-repo}/master`
	- click **Create pull request**
- Finally we're ready to talk to Atlantis!
	- Create a comment `atlantis help` to see what commands you can run from the pull request
	- `atlantis plan` will run `terraform plan` behind the scenes. You should see the output commented back on the pull request. You should also see some logs show up where you're running `atlantis server`
	- `atlantis apply` will run `terraform apply`. Since our pull request creates a `null_resource` (which does nothing) this is safe to do.

You're done! You can type `Ctrl-C` to stop Atlantis.

If you're ready to set up Atlantis for a production deployment, see [Production-Ready Deployment](#production-ready-deployment)


## Production-Ready Deployment

## Configuration
Atlantis configuration can be specified via command line flags or a YAML config file.
The `gh-token` flag can also be specified via an `ATLANTIS_GH_TOKEN` environment variable.
Config file values are overridden by environment variables which in turn are overridden by flags.

To use a yaml config file, run atlantis with `--config /path/to/config.yaml`.
The keys of your config file should be the same as the flag, ex.
```yaml
---
log-level: debug
```

To see a list of all flags and their descriptions run `atlantis server --help`

### AWS Credentials
Atlantis handles AWS credentials in the same way as Terraform.
It looks in the regular places for AWS credentials in this order:
1. `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables
2. The AWS credentials file located at `~/.aws/credentials`
3. Instance profile credentials if Atlantis is running on an EC2 instance

#### Assume Role
One additional feature of Atlantis is the ability to use the AWS Security Token Service (STS)
to assume a role and **dynamically name the session** with the GitHub username of whoever commented `atlantis apply`.
This is used at Hootsuite so AWS API actions can be correlated with a specific user.
To take advantage of this feature, simply set the `--aws-assume-role-arn` flag to the
role to be assumed: `arn:aws:iam::ACCOUNT_ID:role/ROLE_NAME`.

If you're using Terraform's [built-in support](https://www.terraform.io/docs/providers/aws/#assume-role) for assume role then
there is no need to set this flag unless you also want your sessions to take the name of the GitHub user.

## Environments


## Locking
When `plan` is run, the project and environment are Locked until an `apply` succeeds and the pull request is merged.
This protects against concurrent modifications to the same set of infrastructure and prevents
users from seeing a `plan` that will be invalid if another pull request is merged.

To unlock the project and environment without completing an `apply`, click the link
at the bottom of each plan to discard the plan and delete the lock.

## `atlantis.yaml` Config File

## Glossary
#### Project
A Terraform project. Multiple projects can be in a single GitHub repo.

#### Environment
A Terraform environment. See [terraform docs](https://www.terraform.io/docs/state/environments.html) for more information.

