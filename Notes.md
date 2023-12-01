

```
go run main.go  scan -D /home/neo/tmp/projects/PyDemoApp2
```

# How to configure private repos

```
go env GOPRIVATE=yourprivaterepo.com
git config --global url."ssh://git@github.com".insteadOf "https://github.com"

```
