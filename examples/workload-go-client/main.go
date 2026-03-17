package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/jwtsvid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	// Set a timeout to prevent the request from hanging if this workload is not properly registered in SPIRE.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	socketPath := os.Getenv("SPIFFE_ENDPOINT_SOCKET")
	if socketPath == "" {
		socketPath = "unix:///tmp/agent.sock"
	}

	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8082"
	}

	err :=

	svid, err := workloadapi.FetchJWTSVID(ctx, jwtsvid.Params{
		Audience: spiffeid.RequireFromString("spiffe://example.nais.io/server").String(),
	}, workloadapi.WithAddr(socketPath))
	if err != nil {
		return fmt.Errorf("unable to fetch SVID: %w", err)
	}

	req, err := http.NewRequest("GET", serverURL, nil)
	if err != nil {
		return fmt.Errorf("unable to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", svid.Marshal()))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to connect to %q: %w", serverURL, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	log.Printf("%s", body)
	return nil
}
