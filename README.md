IAMy
=========

Dump and load your AWS IAM configuration into YAML files.

This allows for you to manage your IAM configuration in a github repo with a pull request model
for changes. Running load is idempotent, so can be run in a CI process.

Inspired by https://github.com/percolate/iamer.

## Usage

```bash
$ iamy dump
Dumping users...
Dumping groups...
Dumping policies...

$ iamy load
Loading users...
Loading groups...
Loading policies...
```
