IAMy
=========

Dump and load your AWS IAM configuration into YAML files.

This allows for you to manage your IAM configuration in a github repo with a pull request model for changes.

Inspired by https://github.com/percolate/iamer.

## Usage

```bash
$ iamy pull

$ find .
./123456789-myaccount/iam/user/joe.yml

$ mkdir -p 123456789-myaccount/iam/user/foo

$ touch 123456789-myaccount/iam/user/foo/bar.baz

$ cat << EOD > 123456789-myaccount/users/billy.blogs
Policies:
- arn:aws:iam::aws:policy/ReadOnly
EOD

$ iamy push
Commands to push changes to AWS:
        aws iam create-user --path /foo --user-name bar.baz
        aws iam create-user --user-name billy.blogs
        aws iam attach-user-policy --user-name billy.blogs --policy-arn arn:aws:iam::aws:policy/ReadOnly

Exec all aws commands? (Y/n) Y

> aws iam create-user --path /foo --user-name bar.baz
> aws iam create-user --user-name billy.blogs
> aws iam attach-user-policy --user-name billy.blogs --policy-arn arn:aws:iam::aws:policy/ReadOnly
```
