package bootstrap

import "testing"

func TestDirectoryLearnsUnknownIssuerTicker(t *testing.T) {
	got := xyzPreIPOIssuers([]byte(`<table><tr><td>ANT</td><td>Live</td><td>ANT references 1 share of Anthropic, PBC Class A common stock.</td></tr><tr><td>FUTUREAI</td><td>Live</td><td>1 share of OpenAI Group PBC common stock.</td></tr><tr><td>OLD</td><td>Converted</td><td>1 share of Old Company common stock</td></tr></table>`))
	if got["xyz:ANT"] != "ANTHROPIC" || got["xyz:FUTUREAI"] != "OPENAI" || len(got) != 2 {
		t.Fatalf("%v", got)
	}
}
