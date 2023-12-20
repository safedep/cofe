

```
go run main.go  scan -D $HOME/tmp/projects/PyDemoApp2
```

# How to configure private repos

```
go env GOPRIVATE=yourprivaterepo.com
git config --global url."ssh://git@github.com".insteadOf "https://github.com"

``


# Read standard pip.conf file to get index urls
go run main.go scan -D $HOME/tmp/projects/xyz/ --graphviz 1.dot --read-std-conf


go run main.go scan -D $HOME/tmp/projects/xyz/ --graphviz 1.dot --read-std-conf --debug -l-

go run main.go scan -D $HOME/tmp/projects/PyDemoApp2/ --csv pydemo.csv --debug -l pydemo.log

go run main.go scan -D $HOME/tmp/projects/PyDemoApp2/ --csv pydemo.csv -v --debug -l-

go run main.go scan -D $HOME/tmp/projects/xyz/ --csv xyz.csv --read-std-conf --debug -l xyz.log

# Generative Python projects

* https://github.com/steven2358/awesome-generative-ai


## Other awesome python projects
https://github.com/vinta/awesome-python


## Awesome Python Applications

https://github.com/mahmoud/awesome-python-applications





```
Prioritized List of Packages to Upgrade as per Vulnerabilities: 
pyyaml/5.1 [Vulnerable] Severity [10] Priority [9] Path: [pydemoapp2 pyyaml]
pillow/9.4.0 [Vulnerable] Severity [9] Priority [8] Path: [pydemoapp2 pillow]
certifi/2022.12.7 [Vulnerable] Severity [8] Priority [6] Path: [pydemoapp2 requests certifi]
urllib3/1.26.9 [Vulnerable] Severity [8] Priority [6] Path: [pydemoapp2 requests urllib3]
requests/2.28.2 [Vulnerable] Severity [6] Priority [5] Path: [pydemoapp2 requests]
cryptography/39.0.1 [Vulnerable] Severity [6] Priority [4] Path: [pydemoapp2 pyjwt cryptography]
sqlparse/0.3.1 [Vulnerable] Severity [6] Priority [3] Path: [pydemoapp2 django-crispy-forms django sqlparse]

False Positives Removed after reachability analysis: 
werkzeug/2.1.2 [Vulnerable] Severity [8] Priority [7] Path: [pydemoapp2 werkzeug]
Prioritized List of Packages to Upgrade as per Scorecard Score: 
	django-heroku/0.3.1 [Poor Hygiene] Score [0.000000] Priority [9] Path: [pydemoapp2 django-heroku]
	requests-oauthlib/1.3.1 [Poor Hygiene] Score [4.500000] Priority [4] Path: [pydemoapp2 requests-oauthlib]
	dj-database-url/0.5.0 [Poor Hygiene] Score [3.400000] Priority [4] Path: [pydemoapp2 django-heroku dj-database-url]
	asgiref/3.6.0 [Poor Hygiene] Score [5.100000] Priority [3] Path: [pydemoapp2 asgiref]
	pyjwt/2.4.0 [Poor Hygiene] Score [5.900000] Priority [3] Path: [pydemoapp2 pyjwt]
	oauthlib/3.2.2 [Poor Hygiene] Score [6.000000] Priority [2] Path: [pydemoapp2 requests-oauthlib oauthlib]
	argon2-cffi-bindings/21.2.0 [Poor Hygiene] Score [6.100000] Priority [2] Path: [pydemoapp2 argon2-cffi-bindings]
	defusedxml/0.7.1 [Poor Hygiene] Score [6.800000] Priority [2] Path: [pydemoapp2 defusedxml]
	pillow/9.4.0 [Poor Hygiene] Score [7.200000] Priority [1] Path: [pydemoapp2 pillow]
	argon2-cffi/21.3.0 [Poor Hygiene] Score [7.400000] Priority [1] Path: [pydemoapp2 argon2-cffi]
	certifi/2022.12.7 [Poor Hygiene] Score [7.000000] Priority [1] Path: [pydemoapp2 requests certifi]

False Positives Removed after reachability analysis: 
	psycopg2/2.9.3 [Poor Hygiene] Score [4.900000] Priority [4] Path: [pydemoapp2 psycopg2]
	pyflakes/2.3.1 [Poor Hygiene] Score [4.100000] Priority [4] Path: [pydemoapp2 pyflakes]
	whitenoise/6.2.0 [Poor Hygiene] Score [4.900000] Priority [4] Path: [pydemoapp2 whitenoise]
	python3-openid/3.2.0 [Poor Hygiene] Score [4.200000] Priority [4] Path: [pydemoapp2 python3-openid]
	mccabe/0.6.1 [Poor Hygiene] Score [5.600000] Priority [3] Path: [pydemoapp2 mccabe]
	sqlparse/0.3.1 [Poor Hygiene] Score [6.200000] Priority [2] Path: [pydemoapp2 sqlparse]
	werkzeug/2.1.2 [Poor Hygiene] Score [6.400000] Priority [2] Path: [pydemoapp2 werkzeug]
	zipp/3.8.0 [Poor Hygiene] Score [6.600000] Priority [2] Path: [pydemoapp2 zipp]




https://cosmograph.app/run/?data=https://tesr.com&meta=https://tesr.com&gravity=0.25&repulsion=1&repulsionTheta=1.15&linkSpring=1&linkDistance=10&friction=0.85&renderLinks=true&nodeSizeScale=0.6&linkWidthScale=0.2&linkArrowsSizeScale=0.5&nodeSize=size-vuln_score&nodeColor=color-vuln_color&nodeLabel=id&linkWidth=width-avg-vuln_weight&linkColor=color-vuln_score&```
