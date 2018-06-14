# IAMy

IAMy is a tool for dumping and loading your AWS IAM configuration into YAML files.

This allows you to use an [Infrastructure as Code](https://en.wikipedia.org/wiki/Infrastructure_as_Code) model to manage your IAM configuration. For example, you might use a github repo with a pull request model for changes to IAM config.


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


## Inspiration and similar tools
- https://github.com/percolate/iamer
- https://github.com/hashicorp/terraform
