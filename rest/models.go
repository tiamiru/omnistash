package rest

import "time"

type createNamespaceRequest struct {
	Name string `json:"name"`
}

type createNamespaceResponse struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type deleteNamespaceResponse struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
