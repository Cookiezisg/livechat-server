package httpserver

import (
	"net/http"

	"livechat-server/internal/http/handlers"
	"livechat-server/internal/http/middleware"
	"livechat-server/internal/store"
	"livechat-server/internal/ws"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

type RouterDeps struct {
	Store      *store.Store
	Auth       middleware.AuthMiddleware
	WSHub      *ws.Hub
	UploadDir  string
	MaxUpload  int64
	CORSOrigins []string
}

func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   deps.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	authHandler := handlers.AuthHandler{Store: deps.Store, JWTSecret: deps.Auth.Secret, TokenTTL: deps.Auth.TokenTTL}
	usersHandler := handlers.UsersHandler{Store: deps.Store}
	roomsHandler := handlers.RoomsHandler{Store: deps.Store}
	messagesHandler := handlers.MessagesHandler{Store: deps.Store}
	uploadsHandler := handlers.UploadsHandler{UploadDir: deps.UploadDir, MaxUploadMB: deps.MaxUpload}

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.Health(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("db down"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(deps.Auth.Require)
			r.Get("/auth/me", authHandler.Me)
			r.Get("/users", usersHandler.Search)
			r.Get("/rooms", roomsHandler.List)
			r.Post("/rooms", roomsHandler.Create)
			r.Post("/rooms/{roomId}/join", roomsHandler.Join)
			r.Post("/rooms/{roomId}/leave", roomsHandler.Leave)
			r.Get("/rooms/{roomId}/messages", messagesHandler.RoomHistory)
			r.Get("/dms/{userId}/messages", messagesHandler.DMHistory)
			r.Post("/uploads", uploadsHandler.Upload)
		})
	})

	r.Get("/ws", deps.WSHub.ServeWS)
	fs := http.FileServer(http.Dir(deps.UploadDir))
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", fs))

	return r
}
