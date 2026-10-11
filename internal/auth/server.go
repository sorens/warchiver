package auth

import (
	"context"
	_ "embed"
	"fmt"
	"net"
	"net/http"
	"time"
)

//go:embed assets/WoW_Midnight_Logo_enUS.png
var logoImage []byte

// CallbackServer handles the OAuth callback
type CallbackServer struct {
	port     int
	server   *http.Server
	codeChan chan string
	errChan  chan error
}

// NewCallbackServer creates a new callback server
func NewCallbackServer(port int) *CallbackServer {
	return &CallbackServer{
		port:     port,
		codeChan: make(chan string, 1),
		errChan:  make(chan error, 1),
	}
}

var html = `<!DOCTYPE html>
<html>
<head>
    <title>warchiver auth complete</title>
    <style>
        body {
            background-color: #000000;
            color: #ffffff;
            font-family: Arial, sans-serif;
            text-align: center;
            margin-top: 50px;
        }
        .logo {
            width: 30%;
            height: auto;
        }
    </style>
</head>
<body>
    <h1>Authorization successful.</h1>
    <p>
        <img class="logo" src="/assets/WoW_Midnight_Logo_enUS.png" alt="World of Warcraft Midnight Logo">
    </p>
    <p>Close this window and return to the terminal.</p>
</body>
</html>
`

// Start begins listening for the callback
func (s *CallbackServer) Start(expectedState string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	})

	// Non-blocking sends so a duplicate callback cannot pin this goroutine.
	sendErr := func(err error) {
		select {
		case s.errChan <- err:
		default:
		}
	}
	sendCode := func(code string) {
		select {
		case s.codeChan <- code:
		default:
		}
	}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		if state != expectedState {
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error_description")
			if errMsg == "" {
				errMsg = r.URL.Query().Get("error")
			}
			sendErr(fmt.Errorf("authorization failed: %s", errMsg))
			http.Error(w, "Authorization failed", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)

		sendCode(code)
	})

	// Serve embedded image
	mux.HandleFunc("/assets/WoW_Midnight_Logo_enUS.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(logoImage)
	})

	s.server = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", s.port),
		Handler: mux,
	}

	ln, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}

	go func() {
		if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			select {
			case s.errChan <- fmt.Errorf("callback server: %w", err):
			default:
			}
		}
	}()
	return nil
}

func (s *CallbackServer) Wait(ctx context.Context, code *string) error {
	if code == nil {
		return fmt.Errorf("missing parameter")
	}
	select {
	case *code = <-s.codeChan:
		return nil
	case err := <-s.errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *CallbackServer) Shutdown() error {
	if s.server == nil {
		return nil
	}
	time.Sleep(2 * time.Second) // grace period for the browser to render the success page
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}
