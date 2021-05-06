package main

type ServerConfig struct {
	brokerurl    string
	topic        string
	redisurl     string
	port         string
	mockMode     bool
	googleConfig GoogleConfig
}

type GoogleConfig struct {
	authorizedUsers string
	clientId        string
	clientSecret    string
	redirectUrl     string
	sessionSecret   string
}
