package models

type Repository struct {
	Name        string
	Description string
	URL         string
	CreatedAt   string
}

type Commit struct {
	SHA     string
	Message string
	Author  string
	Date    string
	URL     string
}
