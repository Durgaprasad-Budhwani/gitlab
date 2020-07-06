module github.com/pinpt/agent.next.gitlab

go 1.14

require (
	github.com/dnaeon/go-vcr v1.0.1
	github.com/go-redis/redis/v8 v8.0.0-beta.6 // indirect
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/pinpt/agent.next v0.0.0-20200706123552-efe21bae7dda
	github.com/pinpt/go-common/v10 v10.0.14
	github.com/pinpt/httpclient v0.0.0-20200627153820-d374c2f15648 // indirect
	golang.org/x/text v0.3.3 // indirect
)

replace github.com/pinpt/agent.next => ../agent.next
