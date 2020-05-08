package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/poohvpn/gqlgo"
)

func main() {
	client := gqlgo.NewClient(`https://www.graphqlhub.com/graphql`, &gqlgo.Option{
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
}

func singleReuqest(client *gqlgo.Client) {

	fmt.Println("-----single request----")
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

	httpErr := &gqlgo.HTTPError{}
	jsonErr := &gqlgo.JsonError{}
	gqlErr := gqlgo.GraphQLErrors{}
	switch {
	case err == nil:
	case errors.As(err, &httpErr):
		fmt.Println("http error:", httpErr.Response.StatusCode, "\n", httpErr.SavedBody)
		return
	case errors.As(err, &gqlErr):
		fmt.Println("graphql server error:", gqlErr.Error())
		return
	case errors.As(err, &jsonErr):
		fmt.Println("json unmarshal error:", jsonErr.Error(), "\n", jsonErr.Json)
		return
	default:
		fmt.Println("graphql client error:", err.Error())
		return
	}

	j, _ := json.Marshal(data)
	fmt.Println("graphql request result:\n", string(j))

}

func batchRequest(client *gqlgo.Client) {

	fmt.Println("-----batch request----")
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
	httpErr := &gqlgo.HTTPError{}
	jsonErr := &gqlgo.JsonError{}
	gqlErr := gqlgo.GraphQLErrors{}
	switch {
	case err == nil:
	case errors.As(err, &httpErr):
		fmt.Println("http error:", httpErr.Response.StatusCode, "\n", httpErr.SavedBody)
		return
	case errors.As(err, &gqlErr):
		fmt.Println("graphql server error:", gqlErr.Error())
		return
	case errors.As(err, &jsonErr):
		fmt.Println("json unmarshal error:", jsonErr.Error(), "\n", jsonErr.Json)
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
	type model struct {
		User struct {
			GetValue struct {
				ID   string
				Name string
			}
		}
	}
	data := model{}
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

	j, _ := json.Marshal(data)
	fmt.Println("graphql request result:\n", string(j))
}
