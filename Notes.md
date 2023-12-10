

```
go run main.go  scan -D $HOME/tmp/projects/PyDemoApp2
```

# How to configure private repos

```
go env GOPRIVATE=yourprivaterepo.com
git config --global url."ssh://git@github.com".insteadOf "https://github.com"

``


# Read standard pip.conf file to get index urls
go run main.go scan -D $HOME/tmp/projects/deepc/ --graphviz 1.dot --read-std-conf


go run main.go scan -D $HOME/tmp/projects/deepc/ --graphviz 1.dot --read-std-conf --debug -l-

go run main.go scan -D $HOME/tmp/projects/PyDemoApp2/ --csv pydemo.csv --debug -l pydemo.log

go run main.go scan -D $HOME/tmp/projects/PyDemoApp2/ --csv pydemo.csv -v --debug -l-

# Generative Python projects

* https://github.com/steven2358/awesome-generative-ai


## Other awesome python projects
https://github.com/vinta/awesome-python


## Awesome Python Applications

https://github.com/mahmoud/awesome-python-applications
