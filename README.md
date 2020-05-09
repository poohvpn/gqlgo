# GQLGo
An open and well design GraphQL Client for Gophers, it's going to be great.  
And welcome issue and PR by the way.

## Features
- [x] GraphQL
  - [x] basis request
  - [x] batch requests
  - [x] error handling
  - [x] subscriptions
  - [x] file upload
- [x] Custom HTTP Header

## Usage
You can check [example](example/main.go) faster to make the program run.

### Install
```shell script
go get -u github.com/poohvpn/gqlgo
```

### Create Client
```go
client := gqlgo.NewClient(`https://some_endpoint`)
// or
client := gqlgo.NewClient(`https://some_endpoint`, gqlgo.Option{
	HTTPClient: myHttpClient,
	BearerAuth: "token",
	Headers: ...,
	WebSocketEndpoint: "wss://some_endpoint",
})
```

### Do a GraphQL Request
```go
res := struct {
	User struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Email string `json:"email"`
	} `json:"User"`
}{}
err := client.Do(context.Background(), &res, gqlgo.Request{
	Query: `
query ($id:ID){
	User (id: $id) {
		id
		name
		email
	}
}
`,
	Variables: map[string]interface{}{
		"id": "...",
	},
})
```

### GraphQL error handling
```go
detailErr := &gqlgo.DetailError{}
gqlErr := gqlgo.GraphQLErrors{}
switch {
case err == nil:
case errors.As(err, &detailErr):
	fmt.Println("detail error:", detailErr.Response.StatusCode, "\n", detailErr.Content)
	return
case errors.As(err, &gqlErr):
	gqlOneErr := gqlErr[0]
	fmt.Println("graphql server error:",
		gqlOneErr.Message,
		"on line",
		gqlOneErr.Locations[0].Line,
		"column",
		gqlOneErr.Locations[0].Column,
	)
	return
default:
	fmt.Println("graphql client error:", err.Error())
	return
}
```

### Batch Requests
```go
req1 := gqlgo.Request{...}
req2 := gqlgo.Request{...}
data := []interface{}{&data1, &data2}
err := client.Do(context.Background(), data, req1, req2)
```

### Subscription
```go
req1 := gqlgo.Request{...}
subId,err := client.Subscribe(req, func(data json.RawMessage, errors gqlgo.GraphQLErrors, completed bool) error {
	...
})
```

## Credits
[GraphQL Spec](http://spec.graphql.org/draft/)  
[GraphQL MultiPart Request Spec](https://github.com/jaydenseric/graphql-multipart-request-spec)  
Thanks to [machinebox/graphql](https://github.com/machinebox/graphql/), learning a lot from it.  
[apollographql/subscriptions-transport-ws](https://github.com/apollographql/subscriptions-transport-ws)

## License
Apache License 2.0  
Copyright 2020 PoohVPN
