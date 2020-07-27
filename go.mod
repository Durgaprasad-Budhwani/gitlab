module github.com/pinpt/agent.next.gitlab

go 1.14

require (
	github.com/go-redis/redis/v8 v8.0.0-beta.6 // indirect
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/pinpt/agent.next v0.0.0-20200719023419-46571aef39f6
	github.com/pinpt/go-common v9.1.81+incompatible
	github.com/pinpt/go-common/v10 v10.0.16
	github.com/stretchr/testify v1.6.1
	golang.org/x/text v0.3.3 // indirect
)

replace github.com/pinpt/agent.next => ../agent.next
