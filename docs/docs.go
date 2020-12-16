// Package classification GoTrue API
//
// GoTrue is a small open-source API written in golang, that can act as a self-standing
// API service for handling user registration and authentication for JAM projects.
//
// It's based on OAuth2 and JWT and will handle user signup, authentication and custom
// user data.
//
//     Schemes: http, https
//     BasePath: /
//     Version: 1.0.0
//     Host: localhost:9999
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Security:
//     - api_key
//
//    SecurityDefinitions:
//    api_key:
//      type: "apiKey"
//      description: "Auth token"
//      name: "Authorization"
//      in: "header"
//      tokenUrl: "/token"
//
// swagger:meta
package docs
