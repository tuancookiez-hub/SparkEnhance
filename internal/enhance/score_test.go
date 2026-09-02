package enhance

import "testing"

func TestScoreEmpty(t *testing.T) {
	if s := Score(""); s != 0 {
		t.Errorf("Score(\"\") = %d, want 0", s)
	}
}

func TestScoreBlankLines(t *testing.T) {
	if s := Score("\n\n  \n"); s != 0 {
		t.Errorf("Score blank = %d, want 0", s)
	}
}

func TestScoreBaseline(t *testing.T) {
	// "fix this thing" — has verb, no number, no specifics, short
	s := Score("fix this thing")
	if s < 40 || s > 70 {
		t.Errorf("Score = %d, want 40-70 (baseline + verb)", s)
	}
}

func TestScoreStrongPrompt(t *testing.T) {
	text := `Goal: Ship a new login flow for the web app.

1. Add a /login POST endpoint that validates user credentials against the users table
2. Hash passwords with bcrypt and store the hash in the password column
3. Issue a JWT in the response when login succeeds
4. Return a 401 with a clear error message when credentials are wrong
5. Write 3 pytest cases covering happy path, bad password, missing user
6. Add a /logout endpoint that invalidates the token
7. Update the API doc with the new endpoints and example requests`
	s := Score(text)
	if s < 80 {
		t.Errorf("Score = %d, want >= 80 for a strong brief", s)
	}
}

func TestScoreSpecifics(t *testing.T) {
	// "table"/"column" fire specifics; "thing" does not
	withSpec := "Build a function to parse the user table column type from the schema"
	withoutSpec := "Fix the thing"
	diff := Score(withSpec) - Score(withoutSpec)
	if diff < 5 {
		t.Errorf("specifics diff = %d, want >=5: with=%d without=%d",
			diff, Score(withSpec), Score(withoutSpec))
	}
}

func TestScoreMax100(t *testing.T) {
	long := `Goal: Deploy a production-grade microservice.
1. Build the API server with auth middleware on /api/v1
2. Add database migrations for users and orders tables with proper indexes
3. Write 20 pytest cases covering all endpoints
4. Configure CI with GitHub Actions, lint, and test gates
5. Deploy to Kubernetes with rolling updates and health checks
6. Add Prometheus metrics and a Grafana dashboard for the /metrics endpoint
7. Document every endpoint with OpenAPI 3.0 specs and example requests`
	s := Score(long)
	if s > 100 {
		t.Errorf("Score exceeded 100: %d", s)
	}
}
