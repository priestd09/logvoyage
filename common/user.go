package common

import (
	"errors"
	"github.com/Unknwon/com"
	"github.com/belogik/goes"
	"github.com/mitchellh/mapstructure"
	"net/url"
)

type User struct {
	Id           string         `json:"id"`
	Email        string         `json:"email"`
	FirstName    string         `json:"firstName"`
	LastName     string         `json:"lastName"`
	Password     string         `json:"password"`
	ApiKey       string         `json:"apiKey"`
	SourceGroups []*SourceGroup `json:"sourceGroups"`
}

// Returns index name to use in Elastic
func (u *User) GetIndexName() string {
	return u.ApiKey
}

func (u *User) GetLogTypes() []string {
	userLogTypes, _ := GetTypes(u.GetIndexName())
	return userLogTypes
}

// Source group represent group of log types.
// Each log type can be in various groups at the same time.
type SourceGroup struct {
	Id          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Types       []string `json:"types"`
}

func (u *User) AddSourceGroup(group *SourceGroup) *User {
	if group.Id == "" {
		key := com.RandomCreateBytes(5)
		group.Id = string(key)
	} else {
		u.DeleteSourceGroup(group.Id)
	}
	u.SourceGroups = append(u.SourceGroups, group)
	return u
}

func (u *User) DeleteSourceGroup(id string) {
	for i, val := range u.SourceGroups {
		if val.Id == id {
			copy(u.SourceGroups[i:], u.SourceGroups[i+1:])
			u.SourceGroups[len(u.SourceGroups)-1] = nil // or the zero value of T
			u.SourceGroups = u.SourceGroups[:len(u.SourceGroups)-1]
			return
		}
	}
}

func (u *User) GetSourceGroup(id string) (*SourceGroup, error) {
	for _, val := range u.SourceGroups {
		if val.Id == id {
			return val, nil
		}
	}
	return nil, errors.New("Source group not found")
}

func FindUserByEmail(email string) *User {
	return FindUserBy("email", email)
}

func FindUserByApiKey(apiKey string) *User {
	return FindUserBy("apiKey", apiKey)
}

func (this *User) Save() {
	doc := goes.Document{
		Index:  "users",
		Type:   "user",
		Id:     this.Id,
		Fields: this,
	}
	extraArgs := make(url.Values, 0)
	GetConnection().Index(doc, extraArgs)
}

func FindUserBy(key string, value string) *User {
	var query = map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": map[string]interface{}{
					"term": map[string]interface{}{
						key: map[string]interface{}{
							"value": value,
						},
					},
				},
			},
		},
	}

	searchResults, err := GetConnection().Search(query, []string{"users"}, []string{"user"}, url.Values{})

	if err != nil || searchResults.Hits.Total == 0 {
		return nil
	}

	user := &User{}
	mapstructure.Decode(searchResults.Hits.Hits[0].Source, user)
	user.Id = searchResults.Hits.Hits[0].Id

	return user
}
