IAMy
=========

IAMy is a tool for dumping and loading your AWS IAM configuration into YAML files.

This allows you to use a "Infrastructure as Code" model to manage your IAM configuration, and allows you to operate configuration and change management on a higher level. For example, you might use a github repo with a pull request model for changes.


## How it works

IAMy has two subcommands.

`pull` will sync IAM users, groups and policies from AWS to YAML files

`push` will sync IAM users, groups and policies from YAML files to AWS

For the `push` command, IAMy will output an execution plan as a series of [`aws` cli](https://aws.amazon.com/cli/) commands which can be optionally executed. This turns out to be a very direct and understandable way to display the changes to be made, and means you can pick and choose exactly what commands get actioned.


## Getting set up

Because IAMy uses the aws cli tool, you'll want to install it

To install the aws cli on macOS with brew:
```
brew install awscli
```

For configuration, IAMy uses the same [AWS environment variables](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html#cli-environment) as the aws cli.


## Usage

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
