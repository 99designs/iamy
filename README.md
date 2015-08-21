IAMy
=========

Dump and load your AWS IAM configuration into YAML files.

This allows for you to manage your IAM configuration in a github repo with a pull request model
for changes. Running load is idempotent, so can be run in a CI process.

Inspired by https://github.com/percolate/iamer.

## Usage

```bash
$ iamy dump-to-yaml
Fetching AWS IAM data
Dumping YAML IAM data

$ iamy generate-sync-cmds
Loading YAML IAM data
Fetching AWS IAM data

aws iam create-user --user-name foo.bar
```
