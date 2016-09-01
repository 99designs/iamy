IAMy
=========

IAMy is a tool for dumping and loading your AWS IAM configuration into YAML files.

This allows you to use a "Infrastructure as Code" model to manage your IAM configuration, and allows you to operate configuration and change management on a higher level. For example, you might use a github repo with a pull request model for changes.

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
