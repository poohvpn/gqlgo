package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/poohvpn/gqlgo"
)

func main() {
	client := gqlgo.NewClient(`https://www.graphqlhub.com/graphql`, gqlgo.Option{
		Log: func(msg string) {
			fmt.Println(msg)
		},
	})
	singleReuqest(client)

	client.Endpoint = `https://countries.trevorblades.com/`
	batchRequest(client)

	client.Endpoint = `https://graphql.anilist.co/`
	client.NotCheckHTTPStatusCode200 = true
	handleGraphqlError(client)

	client = gqlgo.NewClient(`http://127.0.0.1:8080/v1/graphql`, gqlgo.Option{
		Log: func(msg string) {
			fmt.Println(msg)
		},
		WebSocketOption: gqlgo.WSOption{
			Log: func(msg string) {
				fmt.Println(msg)
			},
		},
	})
	subscribe(client)

}

func singleReuqest(client *gqlgo.Client) {

	fmt.Println("-----graphql single request----")
	data := struct {
		GraphQLHub string `json:"graphQLHub"`
		Reddit     struct {
			User struct {
				Username     string `json:"username"`
				CommentKarma int    `json:"commentKarma"`
				CreatedISO   string `json:"createdISO"`
			}
			SubReddit struct {
				Listing []struct {
					Title    string
					Comments []struct {
						Body   string
						Author struct {
							Username     string
							CommentKarma int
						}
					}
				} `json:"newListings"`
			} `json:"subreddit"`
		}
	}{}
	err := client.Do(context.Background(), &data, gqlgo.Request{
		Query: `
query ($username: String!, $sub: String!) {
 graphQLHub
 reddit {
   user(username: $username) {
     username
     commentKarma
     createdISO
   }
   subreddit(name: $sub) {
     newListings(limit: 2) {
       title
       comments {
         body
         author {
           username
           commentKarma
         }
       }
     }
   }
 }
}

`,
		Variables: map[string]interface{}{
			"username": "kn0thing",
			"sub":      "movies",
		},
	})

	detailErr := &gqlgo.DetailError{}
	gqlErr := gqlgo.GraphQLErrors{}
	switch {
	case err == nil:
	case errors.As(err, &detailErr):
		fmt.Println("detail error:", detailErr.Response.StatusCode, "\n", detailErr.Content)
		return
	case errors.As(err, &gqlErr):
		fmt.Println("graphql server error:", gqlErr.Error())
		return
	default:
		fmt.Println("graphql client error:", err.Error())
		return
	}

	j, _ := json.Marshal(data)
	fmt.Println("graphql request result:\n", string(j))

}

func batchRequest(client *gqlgo.Client) {

	fmt.Println("-----graphql batch request----")
	data1 := struct {
		Country struct {
			Code string
			Name string
		}
	}{}
	req1 := gqlgo.Request{
		Query: `
query($code:ID!){
  country(code:$code){
    code
    name
  }
}
`,
		Variables: map[string]interface{}{
			"code": "US",
		},
	}

	data2 := struct {
		Language struct {
			Code   string
			Name   string
			Native string
			Rtl    bool
		}
	}{}
	req2 := gqlgo.Request{
		Query: `
query($code:ID!){
  language(code:$code){
    code
    name
    native
    rtl
  }
}
`,
		Variables: map[string]interface{}{
			"code": "fr",
		},
	}

	data := []interface{}{&data1, &data2}
	err := client.Do(context.Background(), data, req1, req2)

	detailErr := &gqlgo.DetailError{}
	gqlErr := gqlgo.GraphQLErrors{}
	switch {
	case err == nil:
	case errors.As(err, &detailErr):
		fmt.Println("detail error:", detailErr.Response.StatusCode, "\n", detailErr.Content)
		return
	case errors.As(err, &gqlErr):
		fmt.Println("graphql server error:", gqlErr.Error())
		return
	default:
		fmt.Println("graphql client error:", err.Error())
		return
	}

	j, _ := json.Marshal(data)
	fmt.Println("graphql request result:\n", string(j))
}

func handleGraphqlError(client *gqlgo.Client) {

	fmt.Println("-----handle graphql error----")
	data := struct {
		User struct {
			GetValue struct {
				ID   string
				Name string
			}
		}
	}{}
	req := gqlgo.Request{
		Query: `
query{
  User(name:""){
    id
    name
  }
}
`,
		Variables: map[string]interface{}{},
	}

	err := client.Do(context.Background(), &data, req)

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

	j, _ := json.Marshal(data)
	fmt.Println("graphql request result:\n", string(j))
}

func subscribe(client *gqlgo.Client) {
	fmt.Println("-----graphql subscribe----")
	req := gqlgo.Request{
		Query: `
subscription MyQuery {
  user( where: {id: {_eq: "b00f0f1c-afcd-4455-ab64-d093658ecfc5"}}) {
    username
    id
  }
}
`,
		Variables: map[string]interface{}{},
	}

	user := struct {
		User []struct {
			ID       string
			Username string
		}
	}{}

	recved := make(chan struct{})

	id, err := client.Subscribe(req, func(data json.RawMessage, errors gqlgo.GraphQLErrors, completed bool) error {
		if completed {
			fmt.Println("server send completed")
			return nil
		}
		if errors != nil {
			fmt.Println(errors)
			return errors
		}
		if err := json.Unmarshal(data, &user); err != nil {
			fmt.Println(err)
			return err
		}
		recved <- struct{}{}
		return nil
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("subscription id: %s\n", id)

	<-recved
	if err := client.Unsubscribe(id); err != nil {
		fmt.Println(err)
	}
	fmt.Println(user)
}
