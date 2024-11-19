module github.com/go-gorp/gorp/v3

go 1.18

// Versions prior to 3.0.4 had a vulnerability in the dependency graph.  While we don't
// directly use yaml, I'm not comfortable encouraging people to use versions with a
// CVE - so prior versions are retracted.
//
// See CVE-2019-11254
retract [v3.0.0, v3.0.3]

require (
	github.com/go-sql-driver/mysql v1.6.0
	github.com/lib/pq v1.10.7
	github.com/mattn/go-sqlite3 v1.14.15
	github.com/poy/onpar v0.3.2
	github.com/stretchr/testify v1.8.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
