# Redirectomatic

This is a barebones tool to administer and serve URL redirects on a (sub)domain. Different than a url-shortener because 
users can define their own short link. The goal of this project to keep complexity to a minimum - sticking to standard
libraries as much as possible and limiting the need for additional services in deployment.

Currently, this project uses Go's standard library templating to create a vanilla HTML form-based UI. The redirect
configurations are stored in a single-file embedded database which can be snapshot for backups.

The only development dependency is Go 1.20. Deployment can use a static binary built, removing the need for any
dependencies in production.

```shell
go build ./cmd/redirect # Creates a binary at ./redirect

go run ./cmd/redirect/main.go # Development only - run the server
```

## Roadmap

Future features and major functionality:

- Config file
- Authentication for admin UI and API interfaces (OAuth with Auth0? Slack? Google?)
- Audit log
- Bulk export and bulk import for UI-based backups
- Statistics tracking number of redirects
- Structured logging for better observability in production
- UI with React instead of template/vanilla HTML
