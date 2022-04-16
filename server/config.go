package main

type ServerConfig struct {
	brokerurl    string
	topic        string
	redisurl     string
	postgresurl  string
	port         string
	mockMode     bool
	googleConfig GoogleConfig
	twilioConfig TwilioConfig
	version      string
}

type GoogleConfig struct {
	authorizedUsers string
	clientId        string
	clientSecret    string
	redirectUrl     string
	sessionSecret   string
}

type TwilioConfig struct {
	accountSID string
	authToken  string
	to         string
	from       string
}
