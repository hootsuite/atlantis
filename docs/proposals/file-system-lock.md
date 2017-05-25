Runs will be locked via a file

## When to lock
- when a plan is run successfully

## When to unlock
- apply succeeded

## What gets locked
- url to github repo ex. `github.com/org/repo`
- path to TF project ex. `parentdir/subdir`
- environment

## Lockfile spec
File path
`<atlantis lockfile root>`/`locks`/`<vcs-hostname-and-path>`--`<org>`--`<repo>`--`<optional-dirs>`--`<environment>.lock.yml`
ex. `/var/lib/atlantis/locks/github.com--org--repo--parent-child--staging.lock.yml`

Lock metadata stored with lock
- time locked
- pull request id that locked the TF project

## Example
Comment `atlantis plan staging` on pull `1` repo `github.com/org/repo`
- repo is checked out, and TF projects found in `dir1` and `dir2`
- files have only been changed under `dir2`
- lock file created at `/var/lib/atlantis/locks/github.com--org--repo--dir2--staging.lock.yml`
- lock file contents
```yaml
pullRequestId: 1
timestamp: <timestamp>
user: <username>
```

## No environment
- if there is no environment then we use `default` as the environment just like in terraform 0.9

## Changes
[ ] add `/locks` endpoint, to allow for `HTTP DELETE /locks/${lock-id}` where `lock-id` is the name of the lockfile, ex. `github.com--org--repo--dir2--staging.lock.yml`
[X] instead of sending API call to Stash to check lock, just check the filesystem lock
[ ] change comment on PR to point to our endpoint on Atlantis
