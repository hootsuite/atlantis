# Atlantis

<p align="center">
  <img src="./docs/atlantis-logo.png" alt="Atlantis Logo"/><br><br>
  A unified workflow for collaborating on Terraform through GitHub
</p>

## Walkthrough Video
[![Atlantis Walkthrough](./docs/atlantis-walkthrough-icon.png)](https://www.youtube.com/watch?v=TmIPWda0IKg)

[![CircleCI](https://circleci.com/gh/hootsuite/atlantis/tree/master.svg?style=shield&circle-token=08bf5b34233b0e168a9dd73e01cafdcf7dc4bf16)](https://circleci.com/gh/hootsuite/atlantis/tree/master)
[![SuperDopeBadge](https://img.shields.io/badge/Hightower-extra%20dope-b9f2ff.svg)](https://twitter.com/kelseyhightower/status/893260922222813184)
[![Slack Status](https://thawing-headland-22460.herokuapp.com/badge.svg)](https://thawing-headland-22460.herokuapp.com)

* [Features](#features)
* [Getting Started](#getting-started)
* [Project Structure](#project-structure)
* [Environments](#environments)
* [Terraform Versions](#terraform-versions)
* [Project-Specific Customization](#project-specific-customization)
* [Locking](#locking)
* [Approvals](#approvals)
* [Production-Ready Deployment](#production-ready-deployment)
* [Server Configuration](#server-configuration)
* [AWS Credentials](#aws-credentials)
* [Glossary](#glossary)
    * [Project](#project)
    * [Environment](#environment)
* [FAQ](#faq)

## Features
➜ Collaborate on Terraform with your team
- Run terraform `plan` and `apply` **from GitHub pull requests** so everyone can review the output
- **Lock environments** until pull requests are merged to prevent concurrent modification and confusion

➜ Developers can write Terraform safely
- **No need to distribute AWS credentials** to your whole team. Developers can submit Terraform changes and run `plan` and `apply` directly from the pull request
- Optionally, require a **review and approval** prior to running `apply`

➜ Also
- No more **copy-pasted code across environments**. Atlantis supports using an `env/{env}.tfvars` file per environment so you can write your base configuration once
- Support **multiple versions of Terraform** with a simple project config file


## Getting Started
Download from https://github.com/hootsuite/atlantis/releases

Run
```
./atlantis bootstrap
```
This will walk you through running Atlantis locally. It will
- fork an example terraform project
- install terraform (if not already in your PATH)
- install ngrok so we can expose Atlantis to GitHub
- start Atlantis

If you're ready to permanently set up Atlantis see [Production-Ready Deployment](#production-ready-deployment)

## Project Structure
Atlantis supports several Terraform project structures:
- a single Terraform project at the repo root
```
.
├── main.tf
└── ...
```
-  multiple project folders in a single repo (monorepo)
```
.
├── project1
│   ├── main.tf
|   └── ...
└── project2
    ├── main.tf
    └── ...
```
-  one folder per environment
```
.
├── staging
│   ├── main.tf
|   └── ...
└── production
    ├── main.tf
    └── ...
```
-  using `env/{env}.tfvars` to define environment specific variables. This works in both multi-project repos and single-project repos.
```
.
├── env
│   ├── production.tfvars
│   └── staging.tfvars
└── main.tf
```
or
```
.
├── project1
│   ├── env
│   │   ├── production.tfvars
│   │   └── staging.tfvars
│   └── main.tf
└── project2
    ├── env
    │   ├── production.tfvars
    │   └── staging.tfvars
    └── main.tf
```
With the above project structure you can de-duplicate your Terraform code between environments without requiring extensive use of modules. At Hootsuite we've found this project format to be very successful and use it in all of our 100+ Terraform repositories.

## Environments
Terraform recently introduced [State Environments](https://www.terraform.io/docs/state/environments.html) that
> allows a single folder of Terraform configurations to manage multiple distinct infrastructure resources

If you're using a Terraform version >= 0.9.0, Atlantis supports environments through an additional argument to the `atlantis plan` and `atlantis apply` commands.
For example,
```
atlantis plan staging
```

If an environment is specified Atlantis will use `terraform env select {env}` prior to running `terraform plan` or `terraform apply`.

If you're using the `env/{env}.tfvars` [project structure](#project-structure) we will also append `-tfvars=env/{env}.tfvars` to `plan` and `apply`.

If no environment is specified we will use `default` as the environment.

## Terraform Versions
By default, Atlantis will use the `terraform` executable that is in its path. To use a specific version of Terraform just install that version on the server that Atlantis is running on.

If you would like to use a different version of Terraform for some projects but not for others
1. Install the desired version of Terraform into the `$PATH` of where Atlantis is running and name it `terraform{version}`, ex. `terraform0.8.8`.
2. In the project root (which is not necessarily the repo root) of any project that needs a specific version, create an `atlantis.yaml` file as follows
```
---
terraform_version: 0.8.8 # set to desired version
```

So your project structure will look like
```
.
├── main.tf
└── atlantis.yaml
```
Now when Atlantis executes it will use the `terraform{version}` executable.

## Project-Specific Customization
An `atlantis.yaml` config file in your project root (which is not necessarily the repo root) can be used to customize
- what commands Atlantis runs **before** `plan` and `apply` with `pre_plan` and `pre_apply`
- what commands Atlantis runs **after** `plan` and `apply` with `post_plan` and `post_apply`
- additional arguments to be supplied to specific terraform commands with `extra_arguments`
- what version of Terraform to use (see [Terraform Versions](#terraform-versions))

The schema of the `atlantis.yaml` project config file is

```yaml
# atlantis.yaml
---
terraform_version: 0.8.8 # optional version
pre_plan:
  commands:
  - "curl http://example.com"
post_plan:
  commands:
  - "curl http://example.com"
pre_apply:
  commands:
  - "curl http://example.com"
post_apply:
  commands:
  - "curl http://example.com"
extra_arguments:
  - command: plan
    arguments:
    - "-tfvars=myvars.tfvars"
```

When running the `pre_plan`, `post_plan`, `pre_apply`, and `post_apply` commands the following environment variables are available
- `ENVIRONMENT`: if an environment argument is supplied to `atlantis plan` or `atlantis apply` this will
be the value of that argument. Else it will be `default`
- `ATLANTIS_TERRAFORM_VERSION`: local version of `terraform` or the version from `terraform_version` if specified, ex. `0.10.0`
- `WORKSPACE`: absolute path to the root of the project on disk

## Locking
When `plan` is run, the [project](#project) and [environment](#environment) are **Locked** until an `apply` succeeds **and** the pull request is merged.
This protects against concurrent modifications to the same set of infrastructure and prevents
users from seeing a `plan` that will be invalid if another pull request is merged.

To unlock the project and environment without completing an `apply` and merging, click the link
at the bottom of the plan comment to discard the plan and delete the lock.
Once a plan is discarded, you'll need to run `plan` again prior to running `apply`.

## Approvals
If you'd like to require pull requests to be approved prior to a user running `atlantis apply` simply run Atlantis with the `--require-approval` flag.
By default, no approval is required.

For more information on pull request reviews and approvals see: https://help.github.com/articles/about-pull-request-reviews/

## Production-Ready Deployment
### Install Terraform
`terraform` needs to be in the `$PATH` for Atlantis.
Download from https://www.terraform.io/downloads.html
```
unzip path/to/terraform_*.zip -d /usr/local/bin
```
Check that it's in your `$PATH`
```
$ terraform version
Terraform v0.10.0
```
If you want to use a different version of Terraform see [Terraform Versions](#terraform-versions)

### Hosting Atlantis
Atlantis needs to be hosted somewhere that github.com or your GitHub Enterprise installation can reach. Developers in your organization also need to be able to access Atlantis to view the UI and to delete locks.

By default Atlantis runs on port `4141`. This can be changed with the `--port` flag.

### Add GitHub Webhook
Once you've decided where to host Atlantis you can add it as a Webhook to GitHub.
If you already have a GitHub organization we recommend installing the webhook at the **organization level** rather than on each repository, however both methods will work.

> If you're not sure if you have a GitHub organization see https://help.github.com/articles/differences-between-user-and-organization-accounts/

If you're installing on the organization, navigate to your organization's page and click **Settings**.
If installing on a single repository, navigate to the repository home page and click **Settings**.
- Select **Webhooks** or **Hooks** in the sidebar
- Click **Add webhook**
- set **Payload URL** to `http://$URL/events` where `$URL` is where Atlantis is hosted. **Be sure to add `/events`**
- set **Content type** to `application/json`
- leave **Secret** blank or set this to a random key (https://www.random.org/strings/). If you set it, you'll need to use the `--gh-webhook-secret` option when you start Atlantis
- select **Let me select individual events**
- check the boxes
	- **Pull request review**
	- **Push**
	- **Issue comment**
	- **Pull request**
- leave **Active** checked
- click **Add webhook**

### Create a GitHub Token
We recommend creating a new user in GitHub named **atlantis** that performs all API actions however you can use any user.
Once you've created the user (or have decided to use an existing user) you need to create a personal access token.
- follow [https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/#creating-a-token](https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/#creating-a-token)
- copy the access token

### Start Atlantis
Now you're ready to start Atlantis! Run
```
$ atlantis server --atlantis-url $URL --gh-user $USERNAME --gh-token $TOKEN --gh-webhook-secret $SECRET
2049/10/6 00:00:00 [WARN] server: Atlantis started - listening on port 4141
```

- `$URL` is the URL that Atlantis can be reached at
- `$USERNAME` is the GitHub username you generated the token for
- `$TOKEN` is the access token you created. If you don't want this to be passed in as an argument for security reasons you can specify it in a config file (see [Configuration](#configuration)) or as an environment variable: `ATLANTIS_GH_TOKEN`
- `$SECRET` is the random key you used for the webhook secret. If you left the secret blank then don't specify this flag. If you don't want this to be passed in as an argument for security reasons you can specify it in a config file (see [Configuration](#configuration)) or as an environment variable: `ATLANTIS_GH_WEBHOOK_SECRET`

Atlantis is now running!
**We recommend running it under something like Systemd or Supervisord.**

### Testing Out Atlantis

If you'd like to test out Atlantis before running it on your own repositories you can fork our example repo.

- Fork https://github.com/hootsuite/atlantis-example
- If you didn't add the Webhook as to your organization add Atlantis as a Webhook to the forked repo (see [Add GitHub Webhook](#add-github-webhook))
- Now that Atlantis can receive events you should be able to comment on a pull request to trigger Atlantis. Create a pull request
	- Click **Branches** on your forked repo's homepage
	- click the **New pull request** button next to the `example` branch
	- Change the `base` to `{your-repo}/master`
	- click **Create pull request**
- Now you can test out Atlantis
	- Create a comment `atlantis help` to see what commands you can run from the pull request
	- `atlantis plan` will run `terraform plan` behind the scenes. You should see the output commented back on the pull request. You should also see some logs show up where you're running `atlantis server`
	- `atlantis apply` will run `terraform apply`. Since our pull request creates a `null_resource` (which does nothing) this is safe to do.

## Server Configuration
Atlantis configuration can be specified via command line flags or a YAML config file.
The `gh-token` flag can also be specified via an `ATLANTIS_GH_TOKEN` environment variable.
Config file values are overridden by environment variables which in turn are overridden by flags.

To use a yaml config file, run atlantis with `--config /path/to/config.yaml`.
The keys of your config file should be the same as the flag, ex.
```yaml
---
gh-token: ...
log-level: ...
```

To see a list of all flags and their descriptions run `atlantis server --help`

## AWS Credentials
Atlantis simply shells out to `terraform` so you don't need to do anything special with AWS credentials.
As long as `terraform` works where you're hosting Atlantis, then Atlantis will work.
See https://www.terraform.io/docs/providers/aws/#authentication for more detail.

### Multiple AWS Accounts
Atlantis supports multiple AWS accounts through the use of Terraform's
[AWS Authentication](https://www.terraform.io/docs/providers/aws/#authentication).

If you're using the [Shared Credentials file](https://www.terraform.io/docs/providers/aws/#shared-credentials-file)
you'll need to ensure the server that Atlantis is executing on has the corresponding credentials file.

If you're using [Assume role](https://www.terraform.io/docs/providers/aws/#assume-role)
you'll need to ensure that the credentials file has a `default` profile that is able
to assume all required roles.

[Environment variables](https://www.terraform.io/docs/providers/aws/#environment-variables) authentication
won't work for multiple accounts since Atlantis wouldn't know which environment variables to execute
Terraform with.

### Assume Role Session Names
Atlantis injects the Terraform variable `atlantis_user` and sets it to the GitHub username of
the user that is running the Atlantis command. This can be used to dynamically name the assume role
session. This is used at Hootsuite so AWS API actions can be correlated with a specific user.

To take advantage of this feature, use Terraform's [built-in support](https://www.terraform.io/docs/providers/aws/#assume-role) for assume role
and use the `atlantis_user` terraform variable

```hcl
provider "aws" {
  assume_role {
    role_arn     = "arn:aws:iam::ACCOUNT_ID:role/ROLE_NAME"
    session_name = "${var.atlantis_user}"
  }
}

# need to define the atlantis_user variable to avoid terraform errors
variable "atlantis_user" {
  default = "atlantis_user"
}
```

If you're also using the [S3 Backend](https://www.terraform.io/docs/backends/types/s3.html)
make sure to add the `role_arn` option:

```hcl
terraform {
  backend "s3" {
    bucket   = "mybucket"
    key      = "path/to/my/key"
    region   = "us-east-1"
    role_arn = "arn:aws:iam::ACCOUNT_ID:role/ROLE_NAME"
    # can't use var.atlantis_user as the session name because
    # interpolations are not allowed in backend configuration
    # session_name = "${var.atlantis_user}" WON'T WORK
  }
}
```

Terraform doesn't support interpolations in backend config so you will not be
able to use `session_name = "${var.atlantis_user}"`. However, the backend assumed
role is only used for state-related API actions. Any other API actions will be performed using
the main assumed role and will have the session named as the GitHub user.

## Glossary
#### Project
A Terraform project. Multiple projects can be in a single GitHub repo.
We identify a project by its repo **and** the path to the root of the project within that repo.

#### Environment
A Terraform environment. See [terraform docs](https://www.terraform.io/docs/state/environments.html) for more information.

## FAQ
**Q: Does Atlantis affect Terraform [remote state](https://www.terraform.io/docs/state/remote.html)?**

A: No. Atlantis does not interfere with Terraform remote state in anyway. Under the hood, Atlantis is simply executing `terraform plan` and `terraform apply`.

**Q: How does Atlantis locking interact with Terraform [locking](https://www.terraform.io/docs/state/locking.html)?**

A: Atlantis provides locking of pull requests that prevents concurrent modification of the same infrastructure (Terraform project) whereas Terraform locking only prevents two concurrent `terraform apply`'s from happening. 

Terraform locking can be used alongside Atlantis locking since Atlantis is simply executing terraform commands.

**Q: How to run Atlantis in high availability mode? Does it need to be?**

A: Atlantis server can easily be run under the supervision of a init system like `upstart` or `systemd` to make sure `atlantis server` is always running. 

Atlantis currently stores all locking and Terraform plans locally on disk under the `--data-dir` directory (defaults to `~/.atlantis`). Because of this there is currently no way to run two or more Atlantis instances concurrently.

However, if you were to lose the data, all you would need to do is run `atlantis plan` again on the pull requests that are open. If someone tries to run `atlantis apply` after the data has been lost then they will get an error back, so they will have to re-plan anyway.

**Q: How to add SSL to Atlantis server?**

A: Atlantis currently only supports HTTP. In order to add SSL you will need to front Atlantis server with NGINX or HAProxy. Follow the document [here](./docs/nginx-ssl-proxy.md) to use configure NGINX with SSL as a reverse proxy.



## Credits
* Atlantis Logo: Icon made by [freepik](https://www.flaticon.com/authors/freepik) from www.flaticon.com
