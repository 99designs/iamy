# IAMy

IAMy is a tool for dumping and loading your AWS IAM configuration into YAML files.

This allows you to use an [Infrastructure as Code](https://en.wikipedia.org/wiki/Infrastructure_as_Code) model to manage your IAM configuration. For example, you might use a github repo with a pull request model for changes to IAM config.

# Envato Notes

This code was developed by 99designs ([origin upstream](https://github.com/99designs/iamy.git)).
Envato has created it's own fork to enable us to tailor it to our needs as it appears to not be under active development as at May 2021.

This code is currently being maintained by Q-Branch, however this will transition to platform infrastructure shortly.
Contributions from all parts of the company are welcomed, sing out in #github-administration for access (currently the Envato-Developers team has write access)

**NOTE: THIS IS A PUBLIC REPOSITORY, DO NOT COMMIT ANYTHING SENSITIVE**


## How it works

IAMy has two subcommands.

`pull` will sync IAM users, groups and policies from AWS to YAML files

`push` will sync IAM users, groups and policies from YAML files to AWS

For the `push` command, IAMy will output an execution plan as a series of [`aws` cli](https://aws.amazon.com/cli/) commands which can be optionally executed. This turns out to be a very direct and understandable way to display the changes to be made, and means you can pick and choose exactly what commands get actioned.


## Getting started

You can install IAMy on macOS with `brew install iamy`, or with the go toolchain `go get -u github.com/99designs/iamy`.

Because IAMy uses the [aws cli tool](https://aws.amazon.com/cli/), you'll want to install it first.

For configuration, IAMy uses the same [AWS environment variables](http://docs.aws.amazon.com/cli/latest/userguide/cli-environment.html) as the aws cli. You might find [aws-vault](https://github.com/99designs/aws-vault) an excellent complementary tool for managing AWS credentials.


## Example Usage

```bash
$ iamy pull

$ find .
./myaccount-123456789/iam/user/joe.yml

$ mkdir -p myaccount-123456789/iam/user/foo

$ touch myaccount-123456789/iam/user/foo/bar.baz

$ cat << EOD > myaccount-123456789/iam/user/billy.blogs
Policies:
- arn:aws:iam::aws:policy/ReadOnly
EOD

$ iamy push
Commands to push changes to AWS:
        aws iam create-user --path /foo --user-name bar.baz
        aws iam create-user --user-name billy.blogs
        aws iam attach-user-policy --user-name billy.blogs --policy-arn arn:aws:iam::aws:policy/ReadOnly

Exec all aws commands? (y/N) y

> aws iam create-user --path /foo --user-name bar.baz
> aws iam create-user --user-name billy.blogs
> aws iam attach-user-policy --user-name billy.blogs --policy-arn arn:aws:iam::aws:policy/ReadOnly
```

## Accurate cloudformation matching

By default, iamy will use a simple heuristic (does it end with an ID, eg -ABCDEF1234) to determine if a given resource is managed by cloudformation. 

This behaviour is good enough for some cases, but if you want slower but more accurate matching pass `--accurate-cfn`
to enumerate all cloudformation stacks and resources to determine exactly which resources are managed. 

## Inspiration and similar tools
- https://github.com/percolate/iamer
- https://github.com/hashicorp/terraform
