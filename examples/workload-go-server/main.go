package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/spiffe/go-spiffe/v2/svid/jwtsvid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

func index(w http.ResponseWriter, r *http.Request) {
	log.Println("Request received", svidClaims(r.Context()))
	_, _ = io.WriteString(w, "Success!!!")
}

type authenticator struct {
	// JWT Source used to verify token
	jwtSource *workloadapi.JWTSource
	// Expected audiences
	audiences []string
}

func (a *authenticator) authenticateClient(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fields := strings.Fields(req.Header.Get("Authorization"))
		if len(fields) != 2 || fields[0] != "Bearer" {
			log.Print("Malformed header")
			http.Error(w, "Invalid or unsupported authorization header", http.StatusUnauthorized)
			return
		}

		token := fields[1]

		// Parse and validate token against fetched bundle from jwtSource,
		// an alternative is using `workloadapi.ValidateJWTSVID` that will
		// attest against SPIRE on each call and validate token
		svid, err := jwtsvid.ParseAndValidate(token, a.jwtSource, a.audiences)
		if err != nil {
			log.Printf("Invalid token: %v\n", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		req = req.WithContext(withSVIDClaims(req.Context(), svid.Claims))
		next.ServeHTTP(w, req)
	})
}

type svidClaimsKey struct{}

func withSVIDClaims(ctx context.Context, claims map[string]interface{}) context.Context {
	return context.WithValue(ctx, svidClaimsKey{}, claims)
}

func svidClaims(ctx context.Context) map[string]interface{} {
	claims, _ := ctx.Value(svidClaimsKey{}).(map[string]interface{})
	return claims
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	socketPath := os.Getenv("SPIFFE_ENDPOINT_SOCKET")
	if socketPath == "" {
		socketPath = "unix:///tmp/agent.sock"
	}

	server := &http.Server{
		Addr:              "0.0.0.0:8082",
		ReadHeaderTimeout: time.Second * 10,
	}

	// Create a JWTSource to validate provided tokens from clients
	jwtSource, err := workloadapi.NewJWTSource(ctx, workloadapi.WithClientOptions(workloadapi.WithAddr(socketPath)))
	if err != nil {
		return fmt.Errorf("unable to create JWTSource: %w", err)
	}
	defer jwtSource.Close()

	// Add a middleware to validate presented JWT token
	auth := &authenticator{
		jwtSource: jwtSource,
		audiences: []string{"spiffe://example.nais.io/server"},
	}
	http.Handle("/", auth.authenticateClient(http.HandlerFunc(index)))

	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}
	return nil
}
