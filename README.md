# GQLGo
An open and well design GraphQL Client for Gophers, it's going to be great.  
And welcome issue and PR by the way.

## Features
- [ ] GraphQL
  - [x] basis request
  - [x] batch requests
  - [x] error handling
  - [ ] subscriptions
  - [x] file upload
- [x] Custom HTTP Header
- [x] Bearer authorization
- [x] HTTP request Log

## Usage
You can check [example](example/main.go) faster to make the program run.

### Create Client
```go
client := gqlgo.NewClient(`https://some_endpoint`)
// or
client := gqlgo.NewClient(`https://some_endpoint`,&Option{
	HTTPClient: myHttpClient,
	BearerAuth: "token",
	Headers: ...,
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
httpErr := &gqlgo.HTTPError{}
jsonErr := &gqlgo.JsonError{}
gqlErr := gqlgo.GraphQLErrors{}
switch {
case err == nil:
case errors.As(err, &httpErr):
	fmt.Println("http error:", httpErr.Response.StatusCode, "\n", httpErr.SavedBody)
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
case errors.As(err, &jsonErr):
	fmt.Println("json unmarshal error:", jsonErr.Error(), "\n", jsonErr.Json)
	return
default:
	fmt.Println("graphql client error:", err.Error())
	return
}
```

## Credit
[GraphQL Spec](http://spec.graphql.org/draft/)  
[GraphQL MultiPart Request Spec](https://github.com/jaydenseric/graphql-multipart-request-spec)  
Thanks to [machinebox/graphql](https://github.com/machinebox/graphql/), learning a lot from it.  

## License
Apache License 2.0
Copyright 2020 PoohVPN
