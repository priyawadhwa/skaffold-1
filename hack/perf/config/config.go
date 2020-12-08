package config

// Applications is a list of test applications we are going to sample
type Applications []Application

// Application represents a single test application
type Application struct {
	Name    string
	Context string
	Dev     Dev
	Labels  map[string]string
}

// Dev describes necessary info for running `skaffold dev` on a test application
type Dev struct {
	Command string
}
