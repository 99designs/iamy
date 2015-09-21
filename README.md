IAMy
=========

Dump and load your AWS IAM configuration into YAML files.

This allows for you to manage your IAM configuration in a github repo with a pull request model for changes.

Inspired by https://github.com/percolate/iamer.

## Usage

```bash
$ iamy dump
Fetching AWS IAM data
Dumping YAML IAM data

$ cat << EOD > 123456789-myaccount/users/foo.bar
Name: foo.bar
Path: /baz
EOD

$ iamy load
Loading YAML IAM data
Fetching AWS IAM data
Generating sync commands for account 123456789-myaccount

aws iam create-user --user-name foo.bar --path /baz
```
