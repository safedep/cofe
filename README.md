<h1 align="left">
    <img alt="SafeDep Cofe" src="docs/static/img/cofe-logo.png" width="150" />
</h1>

# Welcome to Cofe
Prioritize OSS vulnerabilities beyond CVSS score, using dependency graph and vulnerability exploitability

## What is it?
It is [Safedep/Vet](https://github.com/safedep/vet) on Steroids, a powerful tool designed to prioritize library and dependency upgrades in your software projects. It uses various heuristics, such as exploitability, reachability, and distinction between internal and external libraries, to make informed decisions about what to upgrade first.

<p align="center">
    <img alt="A typical dependency graph" src="docs/static/img/complex_dep_graph.png" width="250" />
</p>
  A typical application has 1k+ direct and transitive dependencies. Typically, OSV scanner tools report vulnerabilities prioritized by CVSS score. Cofe assists security engineers and developers in finding the path from the application to the vulnerable location and helps in prioritization. 

<p align="center">
    <img alt="A typical dependency graph" src="docs/static/img/dep_graph_reachability.png" width="250" />
</p>

## Quick Start

### Install Safedep/Vet on Steroids

To install, simply run the following command:

```bash
go install github.com/safedep/cofe@main
```

### Run It

To get started with Safedep/Vet on Steroids, run:

```bash
cofe scan -D <Changeme>/<yourproject>/
```

## Advanced Usage

### Scan Your Internal Repository

Cofe allows you to scan your internal repositories with packages in your private artifact repositories. Here are some examples to scan a python project.

```bash
cofe scan -D <Changeme>/<yourproject>/ --read-std-conf
```

### Visualization

#### Via Graphviz Tool

```bash
cofe scan -D <Changeme>/<yourproject>/ --graphviz g.dot --read-std-conf
```
##### Open the dot file using xdot utility on ubuntu:

```bash
xdot g.dot
```


#### Via Chosmosgraph App (Online visualization tool)

```bash
cofe scan -D <Changeme>/<yourproject>/ --csv g.csv --read-std-conf
```
The above command will generate a few sets of files 
* g.csv - containing edges of the dependency graph after the graph is reduced via various techniques such as reachability analysis
* g.csv.metadata.csv - containing metadata related to nodes, such as score and color useful for visualization
* g.csv.orig.csv: Initial Graph without any optimization
* g.csv.orig.metadata.csv: related metadata of the initial graph


## How Does It Work?

Upcoming. Stay Tuned...

## Supported Ecosystems

Currently, Cofe supports the following ecosystem:
* Pypi / Python

## Roadmap

Future updates and expansions planned for Safedep/Vet on Steroids:

1. Add support for Java.
2. Integrate with Neo4j.
3. Expand to support NPM packages.
