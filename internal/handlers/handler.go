package handlers

import "github.com/daniel-oluwadunsin/nombasub/internal/services"

type Handler struct {
	sc *services.Container
}

func New(sc *services.Container) *Handler {
	return &Handler{sc: sc}
}
