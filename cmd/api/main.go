package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bbedward/enwind-api/config"
	user_handler "github.com/bbedward/enwind-api/internal/api/handlers/user"
	"github.com/bbedward/enwind-api/internal/api/middleware"
	"github.com/bbedward/enwind-api/internal/api/server"
	"github.com/bbedward/enwind-api/internal/common/errdefs"
	"github.com/bbedward/enwind-api/internal/common/log"
	"github.com/bbedward/enwind-api/internal/infrastructure/database"
	"github.com/bbedward/enwind-api/internal/repositories/repositories"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/gorilla/schema"
	_ "github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	_ "go.uber.org/automaxprocs"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Adding a format for form data
var decoder = schema.NewDecoder()
var urlEncodedFormat = huma.Format{
	Marshal: nil,
	Unmarshal: func(data []byte, v any) error {
		values, err := url.ParseQuery(string(data))
		if err != nil {
			return err
		}

		// WARNING: Dirty workaround!
		// During validation, Huma first parses the body into []any, map[string]any or equivalent for easy validation,
		// before parsing it into the target struct.
		// However, gorilla/schema requires a struct for decoding, so we need to map `url.Values` to a
		// `map[string]any` if this happens.
		// See: https://github.com/danielgtaylor/huma/blob/main/huma.go#L1264
		if vPtr, ok := v.(*interface{}); ok {
			m := map[string]any{}
			for k, v := range values {
				if len(v) > 1 {
					m[k] = v
				} else if len(v) == 1 {
					m[k] = v[0]
				}
			}
			*vPtr = m
			return nil
		}

		// `v` is a struct, try decode normally
		return decoder.Decode(v, values)
	},
}

func NewHumaConfig(title, version string) huma.Config {
	schemaPrefix := "#/components/schemas/"
	schemasPath := "/schemas"

	registry := huma.NewMapRegistry(schemaPrefix, huma.DefaultSchemaNamer)

	cfg := huma.Config{
		OpenAPI: &huma.OpenAPI{
			OpenAPI: "3.1.0",
			Info: &huma.Info{
				Title:   title,
				Version: version,
			},
			Components: &huma.Components{
				Schemas: registry,
			},
		},
		OpenAPIPath:   "/openapi",
		DocsPath:      "/docs",
		SchemasPath:   schemasPath,
		Formats:       huma.DefaultFormats,
		DefaultFormat: "application/json",
		// * Remove the $schma field
		// CreateHooks: []func(huma.Config) huma.Config{
		// 	func(c huma.Config) huma.Config {
		// 		// Add a link transformer to the API. This adds `Link` headers and
		// 		// puts `$schema` fields in the response body which point to the JSON
		// 		// Schema that describes the response structure.
		// 		// This is a create hook so we get the latest schema path setting.
		// 		linkTransformer := huma.NewSchemaLinkTransformer(schemaPrefix, c.SchemasPath)
		// 		c.OpenAPI.OnAddOperation = append(c.OpenAPI.OnAddOperation, linkTransformer.OnAddOperation)
		// 		c.Transformers = append(c.Transformers, linkTransformer.Transform)
		// 		return c
		// 	},
		// },
	}
	cfg.Formats["application/x-www-form-urlencoded"] = urlEncodedFormat
	cfg.Formats["x-www-form-urlencoded"] = urlEncodedFormat

	return cfg
}

func startAPI(cfg *config.Config) {
	// Create a context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-signalCh
		slog.Info("Received shutdown signal", "signal", sig)
		cancel() // This will propagate cancellation to all derived contexts
	}()

	// Load database
	dbConnInfo, err := database.GetSqlDbConn(cfg, false)
	if err != nil {
		log.Fatalf("Failed to get database connection info: %v", err)
	}
	log.Infof("Using PostgreSQL database %s@%s:%d", cfg.PostgresUser, cfg.PostgresHost, cfg.PostgresPort)
	// Initialize ent client
	db, _, err := database.NewEntClient(dbConnInfo)
	if err != nil {
		log.Fatalf("Failed to create ent client: %v", err)
	}
	if err := db.Schema.Create(ctx); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	repo := repositories.NewRepositories(db)

	// Implementation
	srvImpl := &server.Server{
		Cfg:        cfg,
		Repository: repo,
	}

	// New chi router
	r := chi.NewRouter()

	allowedOrigins := []string{
		"http://localhost:3000",
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.RealIP)
		r.Use(middleware.Logger)

		// Register huma error function
		huma.NewError = errdefs.HumaErrorFunc

		config := NewHumaConfig("Enwind API", "1.0.0")
		config.DocsPath = ""
		// config.OpenAPI.Servers = []*huma.Server{
		// 	{
		// 		URL: cfg.ExternalAPIURL,
		// 	},
		// }
		api := humachi.New(r, config)

		// Create middleware
		mw := middleware.NewMiddleware(cfg, repo, api)

		api.UseMiddleware(mw.Recoverer)

		r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<!doctype html>
			<html>
				<head>
					<title>API Reference</title>
					<meta charset="utf-8" />
					<meta
						name="viewport"
						content="width=device-width, initial-scale=1" />
				</head>
				<body>
					<script
						id="api-reference"
						data-url="/openapi.json"></script>
					<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
				</body>
			</html>`))
		})

		// /users group
		userGroup := huma.NewGroup(api, "/users")
		// userGroup.UseMiddleware(mw.Authenticate)
		userGroup.UseModifier(func(op *huma.Operation, next func(*huma.Operation)) {
			op.Tags = []string{"Users"}
			next(op)
		})
		user_handler.RegisterHandlers(srvImpl, userGroup)
	})

	// Start the server
	addr := ":8091"
	log.Infof("Starting server on %s\n", addr)

	h2s := &http2.Server{}

	server := &http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(r, h2s),
	}

	// Start the server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for context cancellation (from signal handler)
	<-ctx.Done()
	log.Info("Shutting down server...")

	// Create a shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown the HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Info("Server gracefully stopped")
}

func main() {
	// Load environment variables from .env file
	err := godotenv.Overload()
	if err != nil {
		log.Warn("Error loading .env file:", err)
	}

	cfg := config.NewConfig()
	startAPI(cfg)
}
