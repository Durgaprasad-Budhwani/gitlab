module github.com/pinpt/agent.next.gitlab

go 1.14

require (
	github.com/go-redis/redis/v8 v8.0.0-beta.6 // indirect
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/pinpt/agent.next v0.0.0-20200715044241-0a4f3403a457
	github.com/pinpt/go-common/v10 v10.0.15
	golang.org/x/text v0.3.3 // indirect
)

replace github.com/pinpt/agent.next => ../agent.next
