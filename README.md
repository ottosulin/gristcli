# gristcli

This is a CLI tool to manage [grist](https://github.com/gristlabs/grist-core), as currently it is only distributed as a Docker image.

Note: this is a hacky tool made quickly, kind of does the job.

Requirements:
* Docker
* Go

Usage & configurations:
* Install with: go install github.com/ottosulin/gristcli
* Configuration options:
    * -dir: by default uses /grist under user's home directory as its mount path.
    * -email: by default uses test@example.com as user email

## TODO
* Binary releases
* Better configurability, possibly even with a configuration file to pass the supported env variables